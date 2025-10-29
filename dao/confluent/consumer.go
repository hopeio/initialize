package confluent

import (
	"strings"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

type ConsumerConfig kafka.ConfigMap

func (c ConsumerConfig) BeforeInject() {
	for k, v := range c {
		c[strings.ReplaceAll(k, "_", ".")] = v
	}
}

func (c ConsumerConfig) AfterInject() {

}

func (c ConsumerConfig) Build() (*kafka.Consumer, error) {
	return kafka.NewConsumer((*kafka.ConfigMap)(&c))
}

type Consumer struct {
	*kafka.Consumer
	Conf ConsumerConfig
}

func (c *Consumer) Config() any {
	c.Conf = make(ConsumerConfig)
	return &c.Conf
}

func (c *Consumer) Init() error {
	var err error
	c.Consumer, err = c.Conf.Build()
	return err
}

func (c *Consumer) Close() error {
	if c.Consumer == nil {
		return nil
	}
	return c.Consumer.Close()
}
