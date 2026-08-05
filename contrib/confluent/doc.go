//go:build !cgo

// Package confluent wraps confluent-kafka-go and requires CGO (librdkafka).
// Build with CGO_ENABLED=1 to use Consumer/Producer.
package confluent
