/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package ristretto

import (
	"github.com/dgraph-io/ristretto/v2"
)

type Config[K ristretto.Key, V any] ristretto.Config[K, V]

// BeforeInject is a no-op satisfying the Config interface.
func (c *Config[K, V]) BeforeInject() {
}

// AfterInject sets default NumCounters, MaxCost and BufferItems when not configured.
func (c *Config[K, V]) AfterInject() {
	if c.NumCounters == 0 {
		c.NumCounters = 1e7
	}
	if c.MaxCost == 0 {
		c.MaxCost = 1e6
	}
	if c.BufferItems == 0 {
		c.BufferItems = 64
	}
}

// Init is a no-op placeholder; default values are set in AfterInject.
func (c *Config[K, V]) Init() *Config[K, V] {
	return c
}

// Build creates a Ristretto in-memory cache with the configured settings.
func (c *Config[K, V]) Build() (*ristretto.Cache[K, V], error) {
	return ristretto.NewCache((*ristretto.Config[K, V])(c))
}

// Consider swapping caches: ristretto can miss on get-after-set loops unless
// IgnoreInternalCost is set; some issues report large memory use (~1.5MB floor).
// freecache cannot store arbitrary objects without UnmarshalBinary/MarshalBinary.
// go-cache
type Cache[K ristretto.Key, V any] struct {
	*ristretto.Cache[K, V]
	Conf Config[K, V]
}

// Config returns the embedded Conf as the configuration for injection.
func (c *Cache[K, V]) Config() any {
	return &c.Conf
}

// Init creates the Ristretto cache and stores it.
func (c *Cache[K, V]) Init() error {
	var err error
	c.Cache, err = c.Conf.Build()
	return err
}

// Close closes the Ristretto cache if it is non-nil.
func (c *Cache[K, V]) Close() error {
	if c.Cache != nil {
		c.Cache.Close()
	}
	return nil
}
