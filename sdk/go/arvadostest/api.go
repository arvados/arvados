// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvadostest

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"reflect"
	"runtime"
	"sync"

	"git.arvados.org/arvados.git/sdk/go/arvados"
)

var ErrStubUnimplemented = errors.New("stub unimplemented")

type APIStub struct {
	// The error to return from every stubbed API method.
	Error error
	calls []APIStubCall
	mtx   sync.Mutex
}

// BaseURL implements federation.backend
func (as *APIStub) BaseURL() url.URL {
	return url.URL{Scheme: "https", Host: "apistub.example.com"}
}
func (as *APIStub) ConfigGet(ctx context.Context) (json.RawMessage, error) {
	as.appendCall(as.ConfigGet, ctx, nil)
	return nil, as.Error
}
func (as *APIStub) Login(ctx context.Context, options arvados.LoginOptions) (arvados.LoginResponse, error) {
	as.appendCall(as.Login, ctx, options)
	return arvados.LoginResponse{}, as.Error
}
func (as *APIStub) Logout(ctx context.Context, options arvados.LogoutOptions) (arvados.LogoutResponse, error) {
	as.appendCall(as.Logout, ctx, options)
	return arvados.LogoutResponse{}, as.Error
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
func (as *APIStub) UserCreate(ctx context.Context, options arvados.CreateOptions) (arvados.User, error) {
	as.appendCall(as.UserCreate, ctx, options)
	return arvados.User{}, as.Error
}
func (as *APIStub) UserUpdate(ctx context.Context, options arvados.UpdateOptions) (arvados.User, error) {
	as.appendCall(as.UserUpdate, ctx, options)
	return arvados.User{}, as.Error
}
func (as *APIStub) UserUpdateUUID(ctx context.Context, options arvados.UpdateUUIDOptions) (arvados.User, error) {
	as.appendCall(as.UserUpdateUUID, ctx, options)
	return arvados.User{}, as.Error
}
func (as *APIStub) UserActivate(ctx context.Context, options arvados.UserActivateOptions) (arvados.User, error) {
	as.appendCall(as.UserActivate, ctx, options)
	return arvados.User{}, as.Error
}
func (as *APIStub) UserSetup(ctx context.Context, options arvados.UserSetupOptions) (map[string]interface{}, error) {
	as.appendCall(as.UserSetup, ctx, options)
	return nil, as.Error
}
func (as *APIStub) UserUnsetup(ctx context.Context, options arvados.GetOptions) (arvados.User, error) {
	as.appendCall(as.UserUnsetup, ctx, options)
	return arvados.User{}, as.Error
}
func (as *APIStub) UserGet(ctx context.Context, options arvados.GetOptions) (arvados.User, error) {
	as.appendCall(as.UserGet, ctx, options)
	return arvados.User{}, as.Error
}
func (as *APIStub) UserGetCurrent(ctx context.Context, options arvados.GetOptions) (arvados.User, error) {
	as.appendCall(as.UserGetCurrent, ctx, options)
	return arvados.User{}, as.Error
}
func (as *APIStub) UserGetSystem(ctx context.Context, options arvados.GetOptions) (arvados.User, error) {
	as.appendCall(as.UserGetSystem, ctx, options)
	return arvados.User{}, as.Error
}
func (as *APIStub) UserList(ctx context.Context, options arvados.ListOptions) (arvados.UserList, error) {
	as.appendCall(as.UserList, ctx, options)
	return arvados.UserList{}, as.Error
}
func (as *APIStub) UserDelete(ctx context.Context, options arvados.DeleteOptions) (arvados.User, error) {
	as.appendCall(as.UserDelete, ctx, options)
	return arvados.User{}, as.Error
}
func (as *APIStub) UserMerge(ctx context.Context, options arvados.UserMergeOptions) (arvados.User, error) {
	as.appendCall(as.UserMerge, ctx, options)
	return arvados.User{}, as.Error
}
func (as *APIStub) UserBatchUpdate(ctx context.Context, options arvados.UserBatchUpdateOptions) (arvados.UserList, error) {
	as.appendCall(as.UserBatchUpdate, ctx, options)
	return arvados.UserList{}, as.Error
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
		if method == nil || (runtime.FuncForPC(reflect.ValueOf(call.Method).Pointer()).Name() ==
			runtime.FuncForPC(reflect.ValueOf(method).Pointer()).Name()) {
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
