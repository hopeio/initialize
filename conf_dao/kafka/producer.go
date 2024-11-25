/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package kafka

import (
	"github.com/IBM/sarama"
)

type ProducerConfig Config

func (c *ProducerConfig) BeforeInject() {
}
func (c *ProducerConfig) Init() {
	(*Config)(c).Init()
}

func (c *ProducerConfig) Build() (sarama.SyncProducer, error) {
	c.Init()
	// 使用给定代理地址和配置创建一个同步生产者
	return sarama.NewSyncProducer(c.Addrs, c.Config)

}

type Producer struct {
	sarama.SyncProducer
	Conf ProducerConfig
}

func (p *Producer) Config() any {
	p.Conf.Config = sarama.NewConfig()
	return &p.Conf
}

func (p *Producer) Init() error {
	var err error
	p.SyncProducer, err = p.Conf.Build()
	return err
}

func (p *Producer) Close() error {
	if p.SyncProducer == nil {
		return nil
	}
	return p.SyncProducer.Close()
}
