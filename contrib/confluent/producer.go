package confluent

import (
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

type ProducerConfig kafka.ConfigMap

// BeforeInject is a no-op satisfying the Config interface.
func (c ProducerConfig) BeforeInject() {
}

// AfterInject normalizes underscore-keyed config entries to dot notation.
// It must run after unmarshaling, otherwise the map is still empty.
func (c ProducerConfig) AfterInject() {
	normalizeKeys(kafka.ConfigMap(c))
}

// Build creates a Confluent Kafka producer from this config map.
func (c ProducerConfig) Build() (*kafka.Producer, error) {
	return kafka.NewProducer((*kafka.ConfigMap)(&c))
}

type Producer struct {
	*kafka.Producer
	Conf ProducerConfig
}

// Config initializes an empty ProducerConfig map and returns it for injection.
func (p *Producer) Config() any {
	p.Conf = make(ProducerConfig)
	return &p.Conf
}

// Init creates the Confluent Kafka producer and stores it.
func (p *Producer) Init() error {
	var err error
	p.Producer, err = p.Conf.Build()
	return err
}

// Close closes the Confluent Kafka producer and waits for in-flight messages.
func (p *Producer) Close() error {
	if p.Producer == nil {
		return nil
	}
	p.Producer.Close()
	return nil
}
