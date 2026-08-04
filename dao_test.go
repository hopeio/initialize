/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package initialize

import (
	"errors"
	"testing"
)

type daoTestClient struct {
	ID int
}

type daoOkConfig struct {
	client *daoTestClient
}

func (c *daoOkConfig) Build() (*daoTestClient, CloseFunc, error) {
	c.client = &daoTestClient{ID: 42}
	return c.client, func() error { return nil }, nil
}

type daoFailConfig struct{}

func (c *daoFailConfig) Build() (*daoTestClient, CloseFunc, error) {
	return nil, nil, errors.New("build failed")
}

type daoCloseConfig struct {
	closed bool
}

func (c *daoCloseConfig) Build() (*daoTestClient, CloseFunc, error) {
	return &daoTestClient{ID: 1}, func() error {
		c.closed = true
		return nil
	}, nil
}

func TestDaoG_Config(t *testing.T) {
	conf := &daoOkConfig{}
	d := &DaoG[*daoOkConfig, daoTestClient]{Conf: conf}
	if d.Config() != conf {
		t.Fatal("Config() should return Conf")
	}
}

func TestDaoG_Init_Success(t *testing.T) {
	conf := &daoOkConfig{}
	d := &DaoG[*daoOkConfig, daoTestClient]{Conf: conf}
	if err := d.Init(); err != nil {
		t.Fatal(err)
	}
	if d.Client == nil || d.Client.ID != 42 {
		t.Fatalf("Client=%v want ID=42", d.Client)
	}
	if d.close == nil {
		t.Fatal("close func should be set after successful Init")
	}
}

func TestDaoG_Init_BuildFails(t *testing.T) {
	d := &DaoG[*daoFailConfig, daoTestClient]{Conf: &daoFailConfig{}}
	err := d.Init()
	if err == nil {
		t.Fatal("Init want error when Build fails")
	}
	if d.Client != nil {
		t.Fatal("Client should remain nil on Build failure")
	}
}

func TestDaoG_Close_NilCloseFunc(t *testing.T) {
	d := &DaoG[*daoOkConfig, daoTestClient]{Conf: &daoOkConfig{}}
	if err := d.Close(); err != nil {
		t.Fatalf("Close with nil close func: %v", err)
	}
}

func TestDaoG_Close_InvokesCloseFunc(t *testing.T) {
	conf := &daoCloseConfig{}
	d := &DaoG[*daoCloseConfig, daoTestClient]{Conf: conf}
	if err := d.Init(); err != nil {
		t.Fatal(err)
	}
	if err := d.Close(); err != nil {
		t.Fatal(err)
	}
	if !conf.closed {
		t.Fatal("Close func was not invoked")
	}
}
