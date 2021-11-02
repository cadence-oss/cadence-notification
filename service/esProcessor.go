// Copyright (c) 2017 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package indexer

import (
	"context"
	"fmt"
	"time"

	"github.com/uber-go/tally"

	"github.com/uber/cadence/.gen/go/indexer"
	"github.com/uber/cadence/common"
	"github.com/uber/cadence/common/codec"
	"github.com/uber/cadence/common/collection"
	es "github.com/uber/cadence/common/elasticsearch"
	"github.com/uber/cadence/common/log"
	"github.com/uber/cadence/common/log/tag"
	"github.com/uber/cadence/common/messaging"
	"github.com/uber/cadence/common/metrics"
)

type (
	// esProcessorImpl implements ESProcessor, it's an agent of GenericBulkProcessor
	esProcessorImpl struct {
		processor     es.GenericBulkProcessor
		mapToKafkaMsg collection.ConcurrentTxMap // used to map ES request to kafka message
		config        *Config
		logger        log.Logger
		metricsClient metrics.Client
		msgEncoder    codec.BinaryEncoder
	}

	kafkaMessageWithMetrics struct { // value of esProcessorImpl.mapToKafkaMsg
		message        messaging.Message
		swFromAddToAck *tally.Stopwatch // metric from message add to process, to message ack/nack
	}
)

const (
	// retry configs for es bulk processor
	esProcessorInitialRetryInterval = 200 * time.Millisecond
	esProcessorMaxRetryInterval     = 20 * time.Second
)

// newESProcessorAndStart create new ESProcessor and start
func newESProcessorAndStart(config *Config, client es.GenericClient, processorName string,
	logger log.Logger, metricsClient metrics.Client, msgEncoder codec.BinaryEncoder) (*esProcessorImpl, error) {
	p := &esProcessorImpl{
		config:        config,
		logger:        logger.WithTags(tag.ComponentIndexerESProcessor),
		metricsClient: metricsClient,
		msgEncoder:    msgEncoder,
	}

	params := &es.BulkProcessorParameters{
		Name:          processorName,
		NumOfWorkers:  config.ESProcessorNumOfWorkers(),
		BulkActions:   config.ESProcessorBulkActions(),
		BulkSize:      config.ESProcessorBulkSize(),
		FlushInterval: config.ESProcessorFlushInterval(),
		Backoff:       es.NewExponentialBackoff(esProcessorInitialRetryInterval, esProcessorMaxRetryInterval),
		BeforeFunc:    p.bulkBeforeAction,
		AfterFunc:     p.bulkAfterAction,
	}
	processor, err := client.RunBulkProcessor(context.Background(), params)
	if err != nil {
		return nil, err
	}

	p.processor = processor
	p.mapToKafkaMsg = collection.NewShardedConcurrentTxMap(1024, p.hashFn)
	return p, nil
}

func (p *esProcessorImpl) Stop() {
	p.processor.Stop() //nolint:errcheck
	p.mapToKafkaMsg = nil
}

// Add an ES request, and an map item for kafka message
func (p *esProcessorImpl) Add(request *es.GenericBulkableAddRequest, key string, kafkaMsg messaging.Message) {
	actionWhenFoundDuplicates := func(key interface{}, value interface{}) error {
		return kafkaMsg.Ack()
	}
	sw := p.metricsClient.StartTimer(metrics.ESProcessorScope, metrics.ESProcessorProcessMsgLatency)
	mapVal := newKafkaMessageWithMetrics(kafkaMsg, &sw)
	_, isDup, _ := p.mapToKafkaMsg.PutOrDo(key, mapVal, actionWhenFoundDuplicates)
	if isDup {
		return
	}
	p.processor.Add(request)
}

// bulkBeforeAction is triggered before bulk processor commit
func (p *esProcessorImpl) bulkBeforeAction(executionID int64, requests []es.GenericBulkableRequest) {
	p.metricsClient.AddCounter(metrics.ESProcessorScope, metrics.ESProcessorRequests, int64(len(requests)))
}

// bulkAfterAction is triggered after bulk processor commit
func (p *esProcessorImpl) bulkAfterAction(id int64, requests []es.GenericBulkableRequest, response *es.GenericBulkResponse, err *es.GenericError) {
	if err != nil {
		// This happens after configured retry, which means something bad happens on cluster or index
		// When cluster back to live, processor will re-commit those failure requests
		p.logger.Error("Error commit bulk request.", tag.Error(err.Details))

		isRetryable := isResponseRetriable(err.Status)
		for _, request := range requests {
			if !isRetryable {
				key := p.processor.RetrieveKafkaKey(request, p.logger, p.metricsClient)
				if key == "" {
					continue
				}
				wid, rid, domainID := p.getMsgWithInfo(key)
				p.logger.Error("ES request failed.",
					tag.ESResponseStatus(err.Status),
					tag.ESRequest(request.String()),
					tag.WorkflowID(wid),
					tag.WorkflowRunID(rid),
					tag.WorkflowDomainID(domainID))
				p.nackKafkaMsg(key)
			} else {
				p.logger.Error("ES request failed.", tag.ESRequest(request.String()))
			}
			p.metricsClient.IncCounter(metrics.ESProcessorScope, metrics.ESProcessorFailures)
		}
		return
	}

	responseItems := response.Items
	for i := 0; i < len(requests); i++ {
		key := p.processor.RetrieveKafkaKey(requests[i], p.logger, p.metricsClient)
		if key == "" {
			continue
		}
		responseItem := responseItems[i]
		for _, resp := range responseItem {
			switch {
			case isResponseSuccess(resp.Status):
				p.ackKafkaMsg(key)
			case !isResponseRetriable(resp.Status):
				wid, rid, domainID := p.getMsgWithInfo(key)
				p.logger.Error("ES request failed.",
					tag.ESResponseStatus(resp.Status), tag.ESResponseError(getErrorMsgFromESResp(resp)), tag.WorkflowID(wid), tag.WorkflowRunID(rid),
					tag.WorkflowDomainID(domainID))
				p.nackKafkaMsg(key)
			default: // bulk processor will retry
				p.logger.Info("ES request retried.", tag.ESResponseStatus(resp.Status))
				p.metricsClient.IncCounter(metrics.ESProcessorScope, metrics.ESProcessorRetries)
			}
		}
	}
}

func (p *esProcessorImpl) ackKafkaMsg(key string) {
	p.ackKafkaMsgHelper(key, false)
}

func (p *esProcessorImpl) nackKafkaMsg(key string) {
	p.ackKafkaMsgHelper(key, true)
}

func (p *esProcessorImpl) ackKafkaMsgHelper(key string, nack bool) {
	kafkaMsg, ok := p.getKafkaMsg(key)
	if !ok {
		return
	}

	if nack {
		kafkaMsg.Nack()
	} else {
		kafkaMsg.Ack()
	}

	p.mapToKafkaMsg.Remove(key)
}

func (p *esProcessorImpl) getKafkaMsg(key string) (kafkaMsg *kafkaMessageWithMetrics, ok bool) {
	msg, ok := p.mapToKafkaMsg.Get(key)
	if !ok {
		return // duplicate kafka message
	}
	kafkaMsg, ok = msg.(*kafkaMessageWithMetrics)
	if !ok { // must be bug in code and bad deployment
		p.logger.Fatal("Message is not kafka message.", tag.ESKey(key))
	}
	return kafkaMsg, ok
}

func (p *esProcessorImpl) getMsgWithInfo(key string) (wid string, rid string, domainID string) {
	kafkaMsg, ok := p.getKafkaMsg(key)
	if !ok {
		return
	}

	var msg indexer.Message
	if err := p.msgEncoder.Decode(kafkaMsg.message.Value(), &msg); err != nil {
		p.logger.Error("failed to deserialize kafka message.", tag.Error(err))
		return
	}
	return msg.GetWorkflowID(), msg.GetRunID(), msg.GetDomainID()
}

func (p *esProcessorImpl) hashFn(key interface{}) uint32 {
	id, ok := key.(string)
	if !ok {
		return 0
	}
	numOfShards := p.config.IndexerConcurrency()
	return uint32(common.WorkflowIDToHistoryShard(id, numOfShards))
}

// 409 - Version Conflict
// 404 - Not Found
func isResponseSuccess(status int) bool {
	if status >= 200 && status < 300 || status == 409 || status == 404 {
		return true
	}
	return false
}

// isResponseRetriable is complaint with GenericBulkProcessorService.RetryItemStatusCodes
// responses with these status will be kept in queue and retried until success
// 408 - Request Timeout
// 429 - Too Many Requests
// 500 - Node not connected
// 503 - Service Unavailable
// 507 - Insufficient Storage
var retryableStatusCode = map[int]struct{}{408: {}, 429: {}, 500: {}, 503: {}, 507: {}}

func isResponseRetriable(status int) bool {
	_, ok := retryableStatusCode[status]
	return ok
}

func getErrorMsgFromESResp(resp *es.GenericBulkResponseItem) string {
	var errMsg string
	if resp.Error != nil {
		errMsg = fmt.Sprintf("%v", resp.Error)
	}
	return errMsg
}

func newKafkaMessageWithMetrics(kafkaMsg messaging.Message, stopwatch *tally.Stopwatch) *kafkaMessageWithMetrics {
	return &kafkaMessageWithMetrics{
		message:        kafkaMsg,
		swFromAddToAck: stopwatch,
	}
}

func (km *kafkaMessageWithMetrics) Ack() {
	km.message.Ack() //nolint:errcheck
	if km.swFromAddToAck != nil {
		km.swFromAddToAck.Stop()
	}
}

func (km *kafkaMessageWithMetrics) Nack() {
	km.message.Nack() //nolint:errcheck
	if km.swFromAddToAck != nil {
		km.swFromAddToAck.Stop()
	}
}
