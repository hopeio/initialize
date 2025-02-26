/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package kafka

import (
	"github.com/IBM/sarama"
)

type Config struct {
	Addrs []string
	*sarama.Config
}

func (c *Config) Default() {
}
func (c *Config) AfterInject() {
	c.Config.Version = sarama.V3_1_0_0
}
