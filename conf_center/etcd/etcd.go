/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package etcd

import (
	"bytes"
	"context"
	clientv3 "go.etcd.io/etcd/client/v3"
	"io"
)

var ConfigCenter = &Etcd{}

type Etcd struct {
	Conf   Config
	Client *clientv3.Client
}

type Config struct {
	clientv3.Config
	Key string
}

func (e *Etcd) Type() string {
	return "etcd"
}

func (cc *Etcd) Config() any {
	return &cc.Conf
}

// TODO: 监听更改
func (e *Etcd) Handle(handle func(io.Reader) error) error {
	var err error
	if e.Client == nil {
		e.Client, err = clientv3.New(e.Conf.Config)
		if err != nil {
			return err
		}
	}

	resp, err := e.Client.Get(context.Background(), e.Conf.Key)
	if err != nil {
		return err
	}
	return handle(bytes.NewReader(resp.Kvs[0].Value))
}

func (cc *Etcd) Close() error {
	return cc.Client.Close()
}
