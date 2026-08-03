package minio

import (
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Config struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	Token           string
	SignerType      credentials.SignatureType
	minio.Options
}

func (c *Config) BeforeInject() {
}

func (c *Config) AfterInject() {
	if c.Creds == nil {
		c.Creds = credentials.NewStatic(c.AccessKeyID, c.SecretAccessKey, c.Token, c.SignerType)
	}
}

func (c *Config) Build() (*minio.Client, error) {
	return minio.New(c.Endpoint, &c.Options)
}

type Client struct {
	*minio.Client
	Conf Config
}

func (e *Client) Config() any {
	return &e.Conf
}

func (e *Client) Init() error {
	var err error
	e.Client, err = e.Conf.Build()
	return err
}

func (e *Client) Close() error {
	return nil
}
