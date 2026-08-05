/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package initialize

import (
	"io"
	"reflect"
)

var DaoFieldType = reflect.TypeOf((*DaoField)(nil)).Elem()

type DaoField interface {
	Config() any
	Init() error
	io.Closer
}

type CloseFunc func() error

type DaoConfig[D any] interface {
	Build() (*D, CloseFunc, error)
}

type DaoG[C DaoConfig[D], D any] struct {
	Conf   C
	Client *D
	close  CloseFunc
}

// Config returns the embedded Conf pointer as the configuration object for injection.
func (d *DaoG[C, D]) Config() any {
	return d.Conf
}

// Init calls Build on the configuration and stores the resulting client.
func (d *DaoG[C, D]) Init() error {
	var err error
	d.Client, d.close, err = d.Conf.Build()
	return err
}

// Close calls the cleanup function returned by Build, if any.
func (d *DaoG[C, D]) Close() error {
	if d.close != nil {
		return d.close()
	}
	return nil
}
