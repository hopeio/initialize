package confluent

import (
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

type ConsumerConfig kafka.ConfigMap

// BeforeInject is a no-op satisfying the Config interface.
func (c ConsumerConfig) BeforeInject() {
}

// AfterInject normalizes underscore-keyed config entries to dot notation (confluent-kafka-go convention).
// It must run after unmarshaling, otherwise the map is still empty.
func (c ConsumerConfig) AfterInject() {
	normalizeKeys(kafka.ConfigMap(c))
}

// Build creates a Confluent Kafka consumer from this config map.
func (c ConsumerConfig) Build() (*kafka.Consumer, error) {
	return kafka.NewConsumer((*kafka.ConfigMap)(&c))
}

type Consumer struct {
	*kafka.Consumer
	Conf ConsumerConfig
}

// Config initializes an empty ConsumerConfig map and returns it for injection.
func (c *Consumer) Config() any {
	c.Conf = make(ConsumerConfig)
	return &c.Conf
}

// Init creates the Confluent Kafka consumer and stores it.
func (c *Consumer) Init() error {
	var err error
	c.Consumer, err = c.Conf.Build()
	return err
}

// Close closes the Confluent Kafka consumer.
func (c *Consumer) Close() error {
	if c.Consumer == nil {
		return nil
	}
	return c.Consumer.Close()
}
