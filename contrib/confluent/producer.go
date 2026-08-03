package confluent

import (
	"strings"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

type ProducerConfig kafka.ConfigMap

func (c ProducerConfig) BeforeInject() {
	for k, v := range c {
		c[strings.ReplaceAll(k, "_", ".")] = v
	}
}
func (c ProducerConfig) AfterInject() {
}

func (c ProducerConfig) Build() (*kafka.Producer, error) {
	// 使用给定代理地址和配置创建一个同步生产者
	return kafka.NewProducer((*kafka.ConfigMap)(&c))

}

type Producer struct {
	*kafka.Producer
	Conf ProducerConfig
}

func (p *Producer) Config() any {
	p.Conf = make(ProducerConfig)
	return &p.Conf
}

func (p *Producer) Init() error {
	var err error
	p.Producer, err = p.Conf.Build()
	return err
}

func (p *Producer) Close() error {
	if p.Producer == nil {
		return nil
	}
	p.Producer.Close()
	return nil
}
