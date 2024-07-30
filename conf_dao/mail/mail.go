package mail

import (
	"errors"
	"net/smtp"
	"strings"
)

type Config struct {
	AuthType string `comment:"CRAMMD5,PLAIN"`
	Identity string
	Host     string
	Port     string
	UserName string
	Password string
	From     string
}

func (c *Config) InitBeforeInject() {

}

func (c *Config) Init() {
}

func (c *Config) Build() (smtp.Auth, error) {
	c.Init()
	if strings.ToUpper(c.AuthType) == "PLAIN" {
		return smtp.PlainAuth(c.Identity, c.UserName, c.Password, c.Host), nil
	}
	if strings.ToUpper(c.AuthType) == "CRAMMD5" {
		return smtp.CRAMMD5Auth(c.UserName, c.Password), nil
	}

	return nil, errors.New("邮箱配置AuthType必填,PLAIN|CRAMMD5")
}

type Mail struct {
	smtp.Auth
	Conf Config
}

func (m *Mail) Config() any {
	return &m.Conf
}

func (m *Mail) Init() error {
	var err error
	m.Auth, err = m.Conf.Build()
	return err
}

func (m *Mail) Close() error {
	return nil
}
