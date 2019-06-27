// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvadostest

import (
	"context"
	"errors"
	"sync"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
)

var ErrStubUnimplemented = errors.New("stub unimplemented")

type APIStub struct {
	// The error to return from every stubbed API method.
	Error error
	calls []APIStubCall
	mtx   sync.Mutex
}

func (as *APIStub) CollectionCreate(ctx context.Context, options arvados.CreateOptions) (arvados.Collection, error) {
	as.appendCall(as.CollectionCreate, ctx, options)
	return arvados.Collection{}, as.Error
}
func (as *APIStub) CollectionUpdate(ctx context.Context, options arvados.UpdateOptions) (arvados.Collection, error) {
	as.appendCall(as.CollectionUpdate, ctx, options)
	return arvados.Collection{}, as.Error
}
func (as *APIStub) CollectionGet(ctx context.Context, options arvados.GetOptions) (arvados.Collection, error) {
	as.appendCall(as.CollectionGet, ctx, options)
	return arvados.Collection{}, as.Error
}
func (as *APIStub) CollectionList(ctx context.Context, options arvados.ListOptions) (arvados.CollectionList, error) {
	as.appendCall(as.CollectionList, ctx, options)
	return arvados.CollectionList{}, as.Error
}
func (as *APIStub) CollectionProvenance(ctx context.Context, options arvados.GetOptions) (map[string]interface{}, error) {
	as.appendCall(as.CollectionProvenance, ctx, options)
	return nil, as.Error
}
func (as *APIStub) CollectionUsedBy(ctx context.Context, options arvados.GetOptions) (map[string]interface{}, error) {
	as.appendCall(as.CollectionUsedBy, ctx, options)
	return nil, as.Error
}
func (as *APIStub) CollectionDelete(ctx context.Context, options arvados.DeleteOptions) (arvados.Collection, error) {
	as.appendCall(as.CollectionDelete, ctx, options)
	return arvados.Collection{}, as.Error
}
func (as *APIStub) CollectionTrash(ctx context.Context, options arvados.DeleteOptions) (arvados.Collection, error) {
	as.appendCall(as.CollectionTrash, ctx, options)
	return arvados.Collection{}, as.Error
}
func (as *APIStub) CollectionUntrash(ctx context.Context, options arvados.UntrashOptions) (arvados.Collection, error) {
	as.appendCall(as.CollectionUntrash, ctx, options)
	return arvados.Collection{}, as.Error
}
func (as *APIStub) ContainerCreate(ctx context.Context, options arvados.CreateOptions) (arvados.Container, error) {
	as.appendCall(as.ContainerCreate, ctx, options)
	return arvados.Container{}, as.Error
}
func (as *APIStub) ContainerUpdate(ctx context.Context, options arvados.UpdateOptions) (arvados.Container, error) {
	as.appendCall(as.ContainerUpdate, ctx, options)
	return arvados.Container{}, as.Error
}
func (as *APIStub) ContainerGet(ctx context.Context, options arvados.GetOptions) (arvados.Container, error) {
	as.appendCall(as.ContainerGet, ctx, options)
	return arvados.Container{}, as.Error
}
func (as *APIStub) ContainerList(ctx context.Context, options arvados.ListOptions) (arvados.ContainerList, error) {
	as.appendCall(as.ContainerList, ctx, options)
	return arvados.ContainerList{}, as.Error
}
func (as *APIStub) ContainerDelete(ctx context.Context, options arvados.DeleteOptions) (arvados.Container, error) {
	as.appendCall(as.ContainerDelete, ctx, options)
	return arvados.Container{}, as.Error
}
func (as *APIStub) ContainerLock(ctx context.Context, options arvados.GetOptions) (arvados.Container, error) {
	as.appendCall(as.ContainerLock, ctx, options)
	return arvados.Container{}, as.Error
}
func (as *APIStub) ContainerUnlock(ctx context.Context, options arvados.GetOptions) (arvados.Container, error) {
	as.appendCall(as.ContainerUnlock, ctx, options)
	return arvados.Container{}, as.Error
}
func (as *APIStub) SpecimenCreate(ctx context.Context, options arvados.CreateOptions) (arvados.Specimen, error) {
	as.appendCall(as.SpecimenCreate, ctx, options)
	return arvados.Specimen{}, as.Error
}
func (as *APIStub) SpecimenUpdate(ctx context.Context, options arvados.UpdateOptions) (arvados.Specimen, error) {
	as.appendCall(as.SpecimenUpdate, ctx, options)
	return arvados.Specimen{}, as.Error
}
func (as *APIStub) SpecimenGet(ctx context.Context, options arvados.GetOptions) (arvados.Specimen, error) {
	as.appendCall(as.SpecimenGet, ctx, options)
	return arvados.Specimen{}, as.Error
}
func (as *APIStub) SpecimenList(ctx context.Context, options arvados.ListOptions) (arvados.SpecimenList, error) {
	as.appendCall(as.SpecimenList, ctx, options)
	return arvados.SpecimenList{}, as.Error
}
func (as *APIStub) SpecimenDelete(ctx context.Context, options arvados.DeleteOptions) (arvados.Specimen, error) {
	as.appendCall(as.SpecimenDelete, ctx, options)
	return arvados.Specimen{}, as.Error
}
func (as *APIStub) APIClientAuthorizationCurrent(ctx context.Context, options arvados.GetOptions) (arvados.APIClientAuthorization, error) {
	as.appendCall(as.APIClientAuthorizationCurrent, ctx, options)
	return arvados.APIClientAuthorization{}, as.Error
}

func (as *APIStub) appendCall(method interface{}, ctx context.Context, options interface{}) {
	as.mtx.Lock()
	defer as.mtx.Unlock()
	as.calls = append(as.calls, APIStubCall{method, ctx, options})
}

func (as *APIStub) Calls(method interface{}) []APIStubCall {
	as.mtx.Lock()
	defer as.mtx.Unlock()
	var calls []APIStubCall
	for _, call := range as.calls {
		if method == nil || call.Method == method {
			calls = append(calls, call)
		}
	}
	return calls
}

type APIStubCall struct {
	Method  interface{}
	Context context.Context
	Options interface{}
}
