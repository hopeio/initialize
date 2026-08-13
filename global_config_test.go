/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package initialize

import (
	"errors"
	"os"
	"testing"
)

type closingDaoField struct {
	closed bool
}

func (c *closingDaoField) Config() any { return c }

func (c *closingDaoField) Init() error { return nil }

func (c *closingDaoField) Close() error {
	c.closed = true
	return nil
}

type testDaoWithClosingField struct {
	EmbeddedPresets
	Store closingDaoField
}

func TestGlobalConfig_Cleanup_RunsDefers(t *testing.T) {
	oldArgs := os.Args
	t.Cleanup(func() { os.Args = oldArgs })
	os.Args = []string{"test"}

	gc := NewGlobalConfig[*UserConfig]()
	var deferCalled bool
	gc.Defer(func() { deferCalled = true })
	gc.Cleanup()
	if !deferCalled {
		t.Fatal("Cleanup should invoke Defer callbacks")
	}
}

func TestGlobalConfig_Cleanup_DefersReverseOrder(t *testing.T) {
	oldArgs := os.Args
	t.Cleanup(func() { os.Args = oldArgs })
	os.Args = []string{"test"}

	gc := NewGlobalConfig[*UserConfig]()
	var order []int
	gc.Defer(func() { order = append(order, 1) })
	gc.Defer(func() { order = append(order, 2) })
	gc.Cleanup()
	if len(order) != 2 || order[0] != 2 || order[1] != 1 {
		t.Fatalf("defer order=%v want [2 1]", order)
	}
}

func TestCloseDao_ClosesDaoFields(t *testing.T) {
	dao := &testDaoWithClosingField{}
	if err := closeDao(dao); err != nil {
		t.Fatal(err)
	}
	if !dao.Store.closed {
		t.Fatal("closeDao should call DaoField.Close")
	}
}

func TestCloseDao_IgnoresScalarAndUnexportedFields(t *testing.T) {
	type daoMixedFields struct {
		EmbeddedPresets
		Name    string
		Count   int
		Tags    []string
		hidden  int
		Store   closingDaoField
		NilPtr  *closingDaoField
		Handler func()
	}
	d := &daoMixedFields{}
	_ = d.hidden
	// 普通字段/未导出字段/nil 指针曾导致 closeDao panic
	if err := closeDao(d); err != nil {
		t.Fatal(err)
	}
	if !d.Store.closed {
		t.Fatal("Store should be closed")
	}
}

func TestCloseDao_AggregatesErrors(t *testing.T) {
	type errField struct {
		EmbeddedPresets
		A closingDaoField
		B errClosingField
	}
	dao := &errField{}
	err := closeDao(dao)
	if err == nil {
		t.Fatal("closeDao want aggregated error")
	}
}

type errClosingField struct{}

func (e *errClosingField) Config() any { return e }

func (e *errClosingField) Init() error { return nil }

func (e *errClosingField) Close() error { return errors.New("close err") }

func TestStart_ReturnsCleanup(t *testing.T) {
	oldArgs := os.Args
	t.Cleanup(func() { os.Args = oldArgs })
	os.Args = []string{"test"}

	conf := &UserConfig{}
	dao := &EmbeddedPresets{}
	cleanup := Start(conf, dao)
	if cleanup == nil {
		t.Fatal("Start should return Cleanup func")
	}
	cleanup()
}
