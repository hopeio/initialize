/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package sarama

import (
	"github.com/IBM/sarama"
)

type ConsumerConfig Config

// BeforeInject is a no-op satisfying the Config interface.
func (c *ConsumerConfig) BeforeInject() {
}

// AfterInject delegates to Config.AfterInject.
func (c *ConsumerConfig) AfterInject() {
	(*Config)(c).AfterInject()
}

// Build creates a Sarama Kafka consumer.
func (c *ConsumerConfig) Build() (sarama.Consumer, error) {
	return sarama.NewConsumer(c.Addrs, c.Config)
}

type Consumer struct {
	sarama.Consumer
	Conf ConsumerConfig
}

// Config returns the embedded Conf (with a fresh sarama.Config) as the configuration for injection.
func (c *Consumer) Config() any {
	c.Conf.Config = sarama.NewConfig()
	return &c.Conf
}

// Init creates the Sarama consumer and stores it.
func (c *Consumer) Init() error {
	var err error
	c.Consumer, err = c.Conf.Build()
	return err
}

// Close closes the Sarama consumer.
func (c *Consumer) Close() error {
	if c.Consumer == nil {
		return nil
	}
	return c.Consumer.Close()
}
