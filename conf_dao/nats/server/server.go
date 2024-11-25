/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package server

import "github.com/nats-io/nats-server/v2/server"

type Config server.Options

func (c *Config) Build() (*server.Server, error) {
	return server.NewServer((*server.Options)(c))
}
