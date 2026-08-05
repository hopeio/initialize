/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package sarama

import (
	"github.com/IBM/sarama"
)

type ProducerConfig Config

// BeforeInject is a no-op satisfying the Config interface.
func (c *ProducerConfig) BeforeInject() {
}

// AfterInject delegates to Config.AfterInject.
func (c *ProducerConfig) AfterInject() {
	(*Config)(c).AfterInject()
}

// Build creates a Sarama synchronous Kafka producer.
func (c *ProducerConfig) Build() (sarama.SyncProducer, error) {
	return sarama.NewSyncProducer(c.Addrs, c.Config)

}

type Producer struct {
	sarama.SyncProducer
	Conf ProducerConfig
}

// Config returns the embedded Conf (with a fresh sarama.Config) as the configuration for injection.
func (p *Producer) Config() any {
	p.Conf.Config = sarama.NewConfig()
	return &p.Conf
}

// Init creates the Sarama sync producer and stores it.
func (p *Producer) Init() error {
	var err error
	p.SyncProducer, err = p.Conf.Build()
	return err
}

// Close closes the Sarama sync producer.
func (p *Producer) Close() error {
	if p.SyncProducer == nil {
		return nil
	}
	return p.SyncProducer.Close()
}
