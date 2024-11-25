/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package client

import "github.com/nats-io/nats.go"

type Config nats.Options

func (c *Config) Build() (*nats.Conn, error) {
	return (*nats.Options)(c).Connect()
}
