package client

import "github.com/nats-io/nats.go"

type Config nats.Options

func (c *Config) Build() (*nats.Conn, error) {
	return (*nats.Options)(c).Connect()
}
