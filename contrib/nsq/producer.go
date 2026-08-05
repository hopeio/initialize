/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package nsq

import "github.com/nsqio/go-nsq"

type ProducerConfig struct {
	Addr string
	*nsq.Config
}

// BeforeInject is a no-op satisfying the Config interface.
func (c *ProducerConfig) BeforeInject() {
}

// AfterInject is a no-op satisfying the Config interface.
func (c *ProducerConfig) AfterInject() {
}

// Init is a no-op placeholder for future default initialization.
func (c *ProducerConfig) Init() {
}

// Build creates an NSQ producer connected to the configured address.
func (c *ProducerConfig) Build() (*nsq.Producer, error) {
	return nsq.NewProducer(c.Addr, c.Config)
}

type Producer struct {
	*nsq.Producer
	Conf ProducerConfig
}

// Config returns the embedded Conf (with a fresh nsq.Config) as the configuration for injection.
func (p *Producer) Config() any {
	p.Conf.Config = nsq.NewConfig()
	return &p.Conf
}

// Init creates the NSQ producer and stores it.
func (p *Producer) Init() error {
	var err error
	p.Producer, err = p.Conf.Build()
	return err
}

// Close stops the NSQ producer gracefully.
func (p *Producer) Close() error {
	if p.Producer != nil {
		p.Producer.Stop()
	}
	return nil
}
