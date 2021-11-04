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

package config

import (
	"encoding/json"
	"time"

	cconfig "github.com/uber/cadence/common/config"
)

type (
	// Config contains the configuration for a set of cadence services
	Config struct {
		// Log is the logging config
		Log cconfig.Logger `yaml:"log"`
		// Service the service config
		Service Service `yaml:"service"`
		// Kafka is the config for connecting to kafka
		Kafka cconfig.KafkaConfig `yaml:"kafka"`
	}

	// Service contains the service specific config items
	Service struct {
		// Metrics is the metrics subsystem configuration
		Metrics cconfig.Metrics `yaml:"metrics"`
		// Notifications is the config for delivering notifications
		Notifications []Notification `yaml:"notifications"`
 	}

 	// Notification contains config to send notification
	Notification struct {
		// name of this notification app
		ApplicationName string  `yaml:"applicationName"`
		// prefix of the Callback REST URL
		CallbackUrlPrefix string  `yaml:"callbackUrlPrefix"`
		// Kafka consumer group name
		ConsumerGroup string  `yaml:"consumerGroup"`
		// Kafka topic to send DLQ after maxing out retries
		ConsumerGroupDlqTopic string  `yaml:"consumerGroupDlqTopic"`
		// "newest" or "oldest" for consumer group first time to consume
		InitialOffset string  `yaml:"initialOffset"`
		// interval for retry when not receiving 200 from callback
		RetryInterval time.Duration `yaml:"retryInterval"`
		// max number of retries on error(not receiving 200)
		MaxRetries    int  `yaml:"maxRetries"`
		// context timeout of callback requests
		CallbackRequestTimeout time.Duration  `yaml:"callbackRequestTimeout"`
		// notifications of which domains will be sent. empty means selecting all
		SelectedDomains []string  `yaml:"selectedDomains"`
	}
)

// String converts the config object into a string
func (c *Config) String() string {
	out, _ := json.MarshalIndent(c, "", "    ")
	return string(out)
}