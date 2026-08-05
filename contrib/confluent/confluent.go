//go:build cgo

package confluent

type Config struct {
	BootstrapServers string
	GroupId          string
	AutoOffsetReset  string
}
