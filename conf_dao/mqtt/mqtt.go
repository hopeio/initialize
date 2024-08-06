package mqtt

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/hopeio/utils/crypto/tls"
	"github.com/hopeio/utils/log"
	"github.com/hopeio/utils/validation"
	"time"
)

type Config struct {
	*mqtt.ClientOptions
	Brokers []string
	CAFile  string `json:"ca_file,omitempty"`
}

func (c *Config) InitBeforeInject() {
	c.ClientOptions = mqtt.NewClientOptions()
}

func (c *Config) Init() {
	tlsConfig, err := tls.NewClientTLSConfig(c.CAFile, "")
	if err != nil {
		log.Fatal(err)
	}
	c.TLSConfig = tlsConfig
	for _, broker := range c.Brokers {
		c.ClientOptions.AddBroker(broker)
	}

	validation.DurationNotify("PingTimeout", c.PingTimeout, time.Second)
	validation.DurationNotify("ConnectTimeout", c.ConnectTimeout, time.Second)
	validation.DurationNotify("MaxReconnectInterval", c.MaxReconnectInterval, time.Second)
	validation.DurationNotify("ConnectRetryInterval", c.ConnectRetryInterval, time.Second)
	validation.DurationNotify("WriteTimeout", c.WriteTimeout, time.Second)
}

func (c *Config) Build() (mqtt.Client, error) {
	c.Init()
	client := mqtt.NewClient(c.ClientOptions)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return client, token.Error()
	}
	return client, nil
}

type Client struct {
	Conf Config
	mqtt.Client
}

func (c *Client) Config() any {
	return &c.Conf
}

func (c *Client) Init() error {
	var err error
	c.Client, err = c.Conf.Build()
	return err
}

func (c *Client) Close() error {
	c.Client.Disconnect(5)
	return nil
}
