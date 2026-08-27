package minio

import (
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Config struct {
	Endpoint        string
	AccessKeyID     string `flag:"env:MINIO_ACCESS_KEY"`
	SecretAccessKey string `flag:"env:MINIO_SECRET_KEY"`
	Token           string
	SignerType      credentials.SignatureType
	minio.Options
}

// BeforeInject is a no-op satisfying the Config interface.
func (c *Config) BeforeInject() {
}

// AfterInject initializes default credentials when not explicitly set.
func (c *Config) AfterInject() {
	if c.Creds == nil {
		c.Creds = credentials.NewStatic(c.AccessKeyID, c.SecretAccessKey, c.Token, c.SignerType)
	}
}

// Build creates a MinIO client using the configured endpoint and credentials.
func (c *Config) Build() (*minio.Client, error) {
	return minio.New(c.Endpoint, &c.Options)
}

type Client struct {
	*minio.Client
	Conf Config
}

// Config returns the embedded Conf as the configuration for injection.
func (e *Client) Config() any {
	return &e.Conf
}

// Init creates the MinIO client and stores it.
func (e *Client) Init() error {
	var err error
	e.Client, err = e.Conf.Build()
	return err
}

// Close is a no-op; MinIO client requires no explicit cleanup.
func (e *Client) Close() error {
	return nil
}
