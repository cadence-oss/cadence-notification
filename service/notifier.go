// Copyright (c) 2021 Cadence workflow OSS organization
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

package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/uber-go/tally"
	"github.com/uber/cadence/.gen/go/indexer"
	"github.com/uber/cadence/common"
	"github.com/uber/cadence/common/codec"
	"github.com/uber/cadence/common/definition"
	"github.com/uber/cadence/common/log"
	"github.com/uber/cadence/common/log/tag"
	"github.com/uber/cadence/common/messaging"
	"github.com/uber/cadence/common/types"

	"github.com/cadence-oss/cadence-notification/common/config"
)

const defaultConcurrency = 10

// notifier consumes visibility message from Kafka topic and notifier external systems
type notifier struct {
	consumer         messaging.Consumer
	subscriberConfig *config.Subscriber
	consumerConfig   *config.KafkaConsumer

	msgEncoder  codec.BinaryEncoder
	logger      log.Logger
	metricScope tally.Scope

	isStarted  int32
	isStopped  int32
	shutdownWG sync.WaitGroup
	shutdownCh chan struct{}
}

var (
	errUnknownMessageType = &types.BadRequestError{Message: "unknown message type"}
)

func newNotifier(kafkaClient messaging.Client, subscriberConfig *config.Subscriber, logger log.Logger, metricScope tally.Scope) (*notifier, error) {
	consumerConfig := subscriberConfig.Consumer
	consumer, err := kafkaClient.NewConsumer(subscriberConfig.Name, consumerConfig.ConsumerGroup)
	if err != nil {
		return nil, err
	}
	return &notifier{
		consumerConfig:   &consumerConfig,
		consumer:         consumer,
		subscriberConfig: subscriberConfig,

		msgEncoder:  codec.NewThriftRWEncoder(),
		logger:      logger.WithTags(tag.Name("Notifier-" + subscriberConfig.Name)),
		metricScope: metricScope,
		shutdownCh:  make(chan struct{}),
	}, nil
}

func (p *notifier) Start() error {
	if !atomic.CompareAndSwapInt32(&p.isStarted, 0, 1) {
		return nil
	}
	p.logger.Info("notifier state changed", tag.LifeCycleStarting)

	if err := p.consumer.Start(); err != nil {
		p.logger.Info("notifier state changed error", tag.LifeCycleStartFailed, tag.Error(err))
		return err
	}

	p.shutdownWG.Add(1)
	go p.processorPump()

	p.logger.Info("notifier state changed", tag.LifeCycleStarted)
	return nil
}

func (p *notifier) Stop() {
	if !atomic.CompareAndSwapInt32(&p.isStopped, 0, 1) {
		return
	}

	p.logger.Info("notifier state changed", tag.LifeCycleStopping)
	defer p.logger.Info("notifier state changed", tag.LifeCycleStopped)

	if atomic.LoadInt32(&p.isStarted) == 1 {
		close(p.shutdownCh)
	}

	if success := common.AwaitWaitGroup(&p.shutdownWG, time.Minute); !success {
		p.logger.Info("notifier state changed error", tag.LifeCycleStopTimedout)
	}
}

func (p *notifier) processorPump() {
	defer p.shutdownWG.Done()

	var workerWG sync.WaitGroup
	concurrency := defaultConcurrency
	if p.consumerConfig.Concurrency > 0 {
		concurrency = p.consumerConfig.Concurrency
	}

	for workerID := 0; workerID < concurrency; workerID++ {
		workerWG.Add(1)
		go p.messageProcessLoop(&workerWG)
	}

	<-p.shutdownCh
	// Processor is shutting down, close the underlying consumer
	p.consumer.Stop()

	p.logger.Info("notifier pump shutting down.")
	if success := common.AwaitWaitGroup(&workerWG, 10*time.Second); !success {
		p.logger.Warn("notifier timed out on worker shutdown.")
	}
}

func (p *notifier) messageProcessLoop(workerWG *sync.WaitGroup) {
	defer workerWG.Done()

	for msg := range p.consumer.Messages() {
		sw := p.metricScope.Timer(processLatency).Start()
		err := p.process(msg)
		sw.Stop()
		if err != nil {
			_ = msg.Nack()
		}
	}
}

func (p *notifier) process(kafkaMsg messaging.Message) error {
	logger := p.logger.WithTags(tag.KafkaPartition(kafkaMsg.Partition()), tag.KafkaOffset(kafkaMsg.Offset()), tag.AttemptStart(time.Now()))

	decodedMsg, err := p.deserialize(kafkaMsg.Value())
	if err != nil {
		logger.Error("Failed to deserialize index messages.", tag.Error(err))
		p.metricScope.Counter(corruptedData)
		return err
	}
	webhook, _ := p.getWebhookForMsg()
	return p.notifySubscriber(decodedMsg, kafkaMsg, webhook)
}

// TODO: retrieve webhook info for a message
func (p *notifier) getWebhookForMsg() (*config.Webhook, error) {
	url, _ := url.Parse("http://localhost:8081/")
	return &config.Webhook{
		URL: *url,
	}, nil
}

func (p *notifier) deserialize(payload []byte) (*indexer.Message, error) {
	var msg indexer.Message
	if err := p.msgEncoder.Decode(payload, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

func (p *notifier) notifySubscriber(decodedMsg *indexer.Message, kafkaMsg messaging.Message, webhook *config.Webhook) error {

	switch decodedMsg.GetMessageType() {
	case indexer.MessageTypeIndex:
		id := fmt.Sprintf("%v-%v", kafkaMsg.Partition(), kafkaMsg.Offset())
		notification, err := p.generateNotification(decodedMsg, id)
		if err != nil {
			_ = kafkaMsg.Nack()
		}
		p.sendMessageToWebhook(notification, webhook)
		_ = kafkaMsg.Ack()
	case indexer.MessageTypeDelete:
		// this is when workflow run passes retention, noop for now
		_ = kafkaMsg.Ack()
	default:
		p.logger.Error("Unknown message type")
		p.metricScope.Counter(corruptedData)
		return errUnknownMessageType
	}

	return nil
}

func (p *notifier) generateNotification(msg *indexer.Message, id string) (*Notification, error) {
	searchAttrs, memo, err := p.dumpAllFieldsToMap(msg.Fields)
	p.logger.Info("for testing notification consuming, will be removed in next PR: maps", tag.Value(searchAttrs))
	if err != nil {
		return nil, err
	}
	notification := &Notification{
		ID:         id,
		DomainID:   msg.GetDomainID(),
		WorkflowID: msg.GetWorkflowID(),
		RunID:      msg.GetRunID(),
		// TODO WorkflowType, startedTime, closedTime
		SearchAttributes: searchAttrs,
		Memo:             memo,
	}
	return notification, nil
}

// return search attributes, memo, and error
func (p *notifier) dumpAllFieldsToMap(fields map[string]*indexer.Field) (map[string]interface{}, map[string]interface{}, error) {
	sa := make(map[string]interface{})
	memo := make(map[string]interface{})
	for k, v := range fields {
		switch v.GetType() {
		case indexer.FieldTypeString:
			sa[k] = v.GetStringData()
		case indexer.FieldTypeInt:
			sa[k] = v.GetIntData()
		case indexer.FieldTypeBool:
			sa[k] = v.GetBoolData()
		case indexer.FieldTypeBinary:
			if k == definition.Memo {
				memo[k] = v.GetBinaryData()
			} else { // custom search attributes
				sa[k] = p.decodeSearchAttrBinary(v.GetBinaryData(), k)
			}
		default:
			// must be bug in code and bad deployment, check data sent from producer
			p.logger.Error("Unknown field type")
			return nil, nil, fmt.Errorf("unknown field type " + v.GetType().String())
		}
	}
	return sa, memo, nil
}

func (p *notifier) decodeSearchAttrBinary(bytes []byte, key string) interface{} {
	var val interface{}
	err := json.Unmarshal(bytes, &val)
	if err != nil {
		p.logger.Error("Error when decode search attributes values.", tag.Error(err), tag.ESField(key))
		p.metricScope.Counter(corruptedData)
	}
	return val
}

func (p *notifier) sendMessageToWebhook(notification *Notification, webhook *config.Webhook) error {
	var jsonStr = []byte(fmt.Sprintf("%v", notification))
	req, err := http.NewRequest("POST", webhook.URL.String(), bytes.NewBuffer(jsonStr))
	if err != nil {
		return err
	}
	req.Header.Set("X-Custom-Header", "myvalue")
	req.Header.Set("Content-Type", "application/json")

	// TODO: setup retry logic using webhook config
	client := &http.Client{}
	p.logger.Info("sending http request")
	resp, err := client.Do(req)
	if err != nil {
		p.logger.Error(err.Error())
		return err
	}
	defer resp.Body.Close()

	p.logger.Info(fmt.Sprintf("response Status: %v", resp.Status))
	p.logger.Info(fmt.Sprintf("response Headers: %v", resp.Header))
	body, _ := ioutil.ReadAll(resp.Body)
	p.logger.Info(fmt.Sprintf("response Body: %v", string(body)))
	return nil
}
