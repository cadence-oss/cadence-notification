// Copyright (c) 2021 Cadence workflow OSS organization
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package service

import (
	"sync/atomic"

	"github.com/uber-go/tally"
	"github.com/uber/cadence/common"
	"github.com/uber/cadence/common/log"
	"github.com/uber/cadence/common/log/tag"
	"github.com/uber/cadence/common/messaging/kafka"
	"github.com/uber/cadence/common/metrics"
	"github.com/uber/cadence/common/service"

	"github.com/cadence-oss/cadence-notification/common/config"
)

type (
	// Service represents the cadence notification service. This service hosts background processing for delivering notifications
	Service struct {
		status      int32
		stopC       chan struct{}
		logger      log.Logger
		metricScope tally.Scope
		config      *config.Config
	}
)

// NewService builds a new cadence-worker service
func NewService(
	config *config.Config,
	logger log.Logger,
	metricScope tally.Scope,
) (*Service, error) {
	return &Service{
		status:      common.DaemonStatusInitialized,
		config:      config,
		logger:      logger,
		metricScope: metricScope,
		stopC:       make(chan struct{}),
	}, nil
}

// Start is called to start the service
func (s *Service) Start() {
	if !atomic.CompareAndSwapInt32(&s.status, common.DaemonStatusInitialized, common.DaemonStatusStarted) {
		return
	}
	s.logger.Info("notification service starting")

	metricsClient := metrics.NewClient(s.metricScope, service.GetMetricsServiceIdx(common.WorkerServiceName, s.logger))
	kafkaClient := kafka.NewKafkaClient(&s.config.Kafka, metricsClient, s.logger, s.metricScope, false)
	var notifiers []*notifier
	for _, sub := range s.config.Service.Subscribers {
		n, err := newNotifier(kafkaClient, &sub, s.logger, s.metricScope)
		if err != nil {
			s.logger.Fatal("failed to start notifier", tag.Error(err))
		}
		err = n.Start()
		if err != nil {
			s.logger.Fatal("failed to start notifier", tag.Error(err))
		}
		notifiers = append(notifiers, n)
	}
	s.logger.Info("notification service started")
	<-s.stopC
	for _, n := range notifiers {
		n.Stop()
	}
}

// Stop is called to stop the service
func (s *Service) Stop() {
	if !atomic.CompareAndSwapInt32(&s.status, common.DaemonStatusStarted, common.DaemonStatusStopped) {
		return
	}
	close(s.stopC)
}