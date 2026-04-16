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
	Keys []string
}

func (e *Etcd) Type() string {
	return "etcd"
}

func (cc *Etcd) Config() any {
	return &cc.Conf
}

func (e *Etcd) Handle(ctx context.Context, merge func(io.Reader) error, onChange func(io.Reader) error) error {
	var err error
	if e.Client == nil {
		e.Client, err = clientv3.New(e.Conf.Config)
		if err != nil {
			return err
		}
	}

	for _, key := range e.Conf.Keys {
		resp, err := e.Client.Get(ctx, key)
		if err != nil {
			return err
		}
		merge(bytes.NewReader(resp.Kvs[0].Value))
		go func() {
			watchChan := e.Client.Watch(ctx, key)
			for watchResp := range watchChan {
				for _, event := range watchResp.Events {
					if event.Type == clientv3.EventTypePut {
						onChange(bytes.NewReader(event.Kv.Value))
					}
				}
			}
		}()
	}
	return nil
}

func (cc *Etcd) Close() error {
	if cc.Client == nil {
		return nil
	}
	return cc.Client.Close()
}
