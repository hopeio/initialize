/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package nsq

import (
	"github.com/nsqio/go-nsq"
)

type ConsumerConfig struct {
	NSQLookupdAddrs []string
	NSQdAddrs       []string
	Topic           string
	Channel         string
	*nsq.Config
}

// BeforeInject is a no-op satisfying the Config interface.
func (c *ConsumerConfig) BeforeInject() {
}

// AfterInject is a no-op satisfying the Config interface.
func (c *ConsumerConfig) AfterInject() {
}

// Init is a no-op placeholder for future default initialization.
func (c *ConsumerConfig) Init() {
}

// Build creates an NSQ consumer and connects it to configured lookupd/nsqd addresses.
func (c *ConsumerConfig) Build() (*nsq.Consumer, error) {
	consumer, err := nsq.NewConsumer(c.Topic, c.Channel, c.Config)
	if err != nil {
		return nil, err
	}

	if len(c.NSQLookupdAddrs) > 0 {
		if err := consumer.ConnectToNSQLookupds(c.NSQLookupdAddrs); err != nil {
			return consumer, err
		}
	}
	if len(c.NSQdAddrs) > 0 {
		if err = consumer.ConnectToNSQDs(c.NSQdAddrs); err != nil {
			return consumer, err
		}

	}
	return consumer, nil

}

type Consumer struct {
	*nsq.Consumer
	Conf ConsumerConfig
}

// Config returns the embedded Conf (with a fresh nsq.Config) as the configuration for injection.
func (c *Consumer) Config() any {
	c.Conf.Config = nsq.NewConfig()
	return &c.Conf
}

// Init creates the NSQ consumer and stores it.
func (c *Consumer) Init() error {
	var err error
	c.Consumer, err = c.Conf.Build()
	return err
}

// Close stops the NSQ consumer gracefully.
func (c *Consumer) Close() error {
	if c.Consumer != nil {
		c.Consumer.Stop()
	}
	return nil
}
