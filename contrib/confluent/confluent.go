package confluent

import (
	"strings"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

type Config struct {
	BootstrapServers string
	GroupId          string
	AutoOffsetReset  string
}

// normalizeKeys rewrites underscore-separated keys to librdkafka's dot notation,
// removing the original keys so unknown properties are not passed through.
func normalizeKeys(m kafka.ConfigMap) {
	for k, v := range m {
		nk := strings.ReplaceAll(k, "_", ".")
		if nk != k {
			m[nk] = v
			delete(m, k)
		}
	}
}
