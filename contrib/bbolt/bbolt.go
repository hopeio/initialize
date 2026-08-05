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

// BeforeInject is a no-op satisfying the Config interface.
func (c *Config) BeforeInject() {
}

// AfterInject sets a default file mode of 0600 if none is configured.
func (c *Config) AfterInject() {
	if c.Mode == 0 {
		c.Mode = 0600
	}
}

// Build opens a bbolt database at the configured path with the given mode.
func (c *Config) Build() (*bbolt.DB, error) {
	return bbolt.Open(c.Path, c.Mode, &c.Options)
}

type DB struct {
	*bbolt.DB
	Conf Config
}

// Config returns the embedded Conf as the configuration for injection.
func (c *DB) Config() any {
	return &c.Conf
}

// Init opens the bbolt database and stores it.
func (c *DB) Init() error {
	var err error
	c.DB, err = c.Conf.Build()
	return err
}

// Close closes the bbolt database.
func (c *DB) Close() error {
	return c.DB.Close()
}
