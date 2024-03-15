/*
 * Copyright (c) 2024 jhuix. All rights reserved.
 * Use of this source code is governed by a license
 * that can be found in the LICENSE file.
 */

package balancer

import (
	`context`
	`errors`
	`strings`

	`github.com/jhuix-go/ebus/pkg/discovery/common/attributes`
)

type Address struct {
	Name string
	Addr string
	// Attributes contains arbitrary data about this address intended for
	// consumption by the SubConn.
	Attributes *attributes.Attributes
}

type PickInfo struct {
	Name string
	Ctx  context.Context
}

type DoneInfo struct {
	// Err is the rpc error the RPC finished with. It could be nil.
	Err error
}

type PickResult struct {
	Node any
	Done func(DoneInfo)
}

type Picker interface {
	Pick(info PickInfo) (PickResult, error)
}

type SubConnInfo struct {
	Address Address
}

type PickerBuildInfo struct {
	ReadySCs map[any]SubConnInfo
}

// PickerBuilder creates Picker.
type PickerBuilder interface {
	// Build returns a picker that will be used by client to pick a SubConn.metadata.go
	Build(info PickerBuildInfo) Picker
	Name() string
}

var m = make(map[string]PickerBuilder)

func Register(b PickerBuilder) {
	m[strings.ToLower(b.Name())] = b
}

func Builder(name string) PickerBuilder {
	b, _ := m[strings.ToLower(name)]
	return b
}

var (
	// ErrNoSubConnAvailable indicates no SubConn is available for pick().
	ErrNoSubConnAvailable = errors.New("no SubConn is available")
	ErrNoContextAvailable = errors.New("no context is available")
)

// NewErrPicker returns a Picker that always returns err on Pick().
func NewErrPicker(err error) Picker {
	return &errPicker{err: err}
}

type errPicker struct {
	err error // Pick() always returns this err.
}

func (p *errPicker) Pick(info PickInfo) (PickResult, error) {
	return PickResult{}, p.err
}
