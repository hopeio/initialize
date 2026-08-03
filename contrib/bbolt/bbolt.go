package bbolt

import (
	"go.etcd.io/bbolt"
	"os"
)

type Config struct {
	Path string
	Mode os.FileMode
	bbolt.Options
}

func (c *Config) BeforeInject() {
}
func (c *Config) AfterInject() {
	if c.Mode == 0 {
		c.Mode = 0600
	}
}

func (c *Config) Build() (*bbolt.DB, error) {
	return bbolt.Open(c.Path, c.Mode, &c.Options)
}

type DB struct {
	*bbolt.DB
	Conf Config
}

func (c *DB) Config() any {
	return &c.Conf
}

func (c *DB) Init() error {
	var err error
	c.DB, err = c.Conf.Build()
	return err
}

func (c *DB) Close() error {
	return c.DB.Close()
}
