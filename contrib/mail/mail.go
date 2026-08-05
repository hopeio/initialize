/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

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

// BeforeInject is a no-op satisfying the Config interface.
func (c *Config) BeforeInject() {

}

// AfterInject calls Init after config fields are populated.
func (c *Config) AfterInject() {
	c.Init()
}

// Init is a no-op placeholder for future default initialization.
func (c *Config) Init() *Config {
	return c
}

// Build returns an smtp.Auth based on the AuthType field ("PLAIN" or "CRAMMD5").
// Returns an error if AuthType is missing or unrecognized.
func (c *Config) Build() (smtp.Auth, error) {
	if strings.ToUpper(c.AuthType) == "PLAIN" {
		return smtp.PlainAuth(c.Identity, c.UserName, c.Password, c.Host), nil
	}
	if strings.ToUpper(c.AuthType) == "CRAMMD5" {
		return smtp.CRAMMD5Auth(c.UserName, c.Password), nil
	}

	return nil, errors.New("mail config AuthType is required, must be PLAIN or CRAMMD5")
}

type Mail struct {
	smtp.Auth
	Conf Config
}

// Config returns the embedded Conf as the configuration for injection.
func (m *Mail) Config() any {
	return &m.Conf
}

// Init creates the smtp.Auth and stores it in Mail.
func (m *Mail) Init() error {
	var err error
	m.Auth, err = m.Conf.Build()
	return err
}

// Close is a no-op; smtp.Auth requires no explicit cleanup.
func (m *Mail) Close() error {
	return nil
}
