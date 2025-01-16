/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package conf_dao

type CloseFunc func() error

type DaoConfig[D any] interface {
	Build() (*D, CloseFunc, error)
}

type DaoT[C DaoConfig[D], D any] struct {
	Conf   C
	Client *D
	close  CloseFunc
}

func (d *DaoT[C, D]) Config() any {
	return d.Conf
}

func (d *DaoT[C, D]) Init() error {
	var err error
	d.Client, d.close, err = d.Conf.Build()
	return err
}

func (d *DaoT[C, D]) Close() error {
	if d.close != nil {
		return d.close()
	}
	return nil
}
