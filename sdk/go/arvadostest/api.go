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
	as.appendCall(ctx, as.ConfigGet, nil)
	return nil, as.Error
}
func (as *APIStub) Login(ctx context.Context, options arvados.LoginOptions) (arvados.LoginResponse, error) {
	as.appendCall(ctx, as.Login, options)
	return arvados.LoginResponse{}, as.Error
}
func (as *APIStub) Logout(ctx context.Context, options arvados.LogoutOptions) (arvados.LogoutResponse, error) {
	as.appendCall(ctx, as.Logout, options)
	return arvados.LogoutResponse{}, as.Error
}
func (as *APIStub) CollectionCreate(ctx context.Context, options arvados.CreateOptions) (arvados.Collection, error) {
	as.appendCall(ctx, as.CollectionCreate, options)
	return arvados.Collection{}, as.Error
}
func (as *APIStub) CollectionUpdate(ctx context.Context, options arvados.UpdateOptions) (arvados.Collection, error) {
	as.appendCall(ctx, as.CollectionUpdate, options)
	return arvados.Collection{}, as.Error
}
func (as *APIStub) CollectionGet(ctx context.Context, options arvados.GetOptions) (arvados.Collection, error) {
	as.appendCall(ctx, as.CollectionGet, options)
	return arvados.Collection{}, as.Error
}
func (as *APIStub) CollectionList(ctx context.Context, options arvados.ListOptions) (arvados.CollectionList, error) {
	as.appendCall(ctx, as.CollectionList, options)
	return arvados.CollectionList{}, as.Error
}
func (as *APIStub) CollectionProvenance(ctx context.Context, options arvados.GetOptions) (map[string]interface{}, error) {
	as.appendCall(ctx, as.CollectionProvenance, options)
	return nil, as.Error
}
func (as *APIStub) CollectionUsedBy(ctx context.Context, options arvados.GetOptions) (map[string]interface{}, error) {
	as.appendCall(ctx, as.CollectionUsedBy, options)
	return nil, as.Error
}
func (as *APIStub) CollectionDelete(ctx context.Context, options arvados.DeleteOptions) (arvados.Collection, error) {
	as.appendCall(ctx, as.CollectionDelete, options)
	return arvados.Collection{}, as.Error
}
func (as *APIStub) CollectionTrash(ctx context.Context, options arvados.DeleteOptions) (arvados.Collection, error) {
	as.appendCall(ctx, as.CollectionTrash, options)
	return arvados.Collection{}, as.Error
}
func (as *APIStub) CollectionUntrash(ctx context.Context, options arvados.UntrashOptions) (arvados.Collection, error) {
	as.appendCall(ctx, as.CollectionUntrash, options)
	return arvados.Collection{}, as.Error
}
func (as *APIStub) ContainerCreate(ctx context.Context, options arvados.CreateOptions) (arvados.Container, error) {
	as.appendCall(ctx, as.ContainerCreate, options)
	return arvados.Container{}, as.Error
}
func (as *APIStub) ContainerUpdate(ctx context.Context, options arvados.UpdateOptions) (arvados.Container, error) {
	as.appendCall(ctx, as.ContainerUpdate, options)
	return arvados.Container{}, as.Error
}
func (as *APIStub) ContainerGet(ctx context.Context, options arvados.GetOptions) (arvados.Container, error) {
	as.appendCall(ctx, as.ContainerGet, options)
	return arvados.Container{}, as.Error
}
func (as *APIStub) ContainerList(ctx context.Context, options arvados.ListOptions) (arvados.ContainerList, error) {
	as.appendCall(ctx, as.ContainerList, options)
	return arvados.ContainerList{}, as.Error
}
func (as *APIStub) ContainerDelete(ctx context.Context, options arvados.DeleteOptions) (arvados.Container, error) {
	as.appendCall(ctx, as.ContainerDelete, options)
	return arvados.Container{}, as.Error
}
func (as *APIStub) ContainerLock(ctx context.Context, options arvados.GetOptions) (arvados.Container, error) {
	as.appendCall(ctx, as.ContainerLock, options)
	return arvados.Container{}, as.Error
}
func (as *APIStub) ContainerUnlock(ctx context.Context, options arvados.GetOptions) (arvados.Container, error) {
	as.appendCall(ctx, as.ContainerUnlock, options)
	return arvados.Container{}, as.Error
}
func (as *APIStub) ContainerSSH(ctx context.Context, options arvados.ContainerSSHOptions) (arvados.ContainerSSHConnection, error) {
	as.appendCall(ctx, as.ContainerSSH, options)
	return arvados.ContainerSSHConnection{}, as.Error
}
func (as *APIStub) ContainerRequestCreate(ctx context.Context, options arvados.CreateOptions) (arvados.ContainerRequest, error) {
	as.appendCall(ctx, as.ContainerRequestCreate, options)
	return arvados.ContainerRequest{}, as.Error
}
func (as *APIStub) ContainerRequestUpdate(ctx context.Context, options arvados.UpdateOptions) (arvados.ContainerRequest, error) {
	as.appendCall(ctx, as.ContainerRequestUpdate, options)
	return arvados.ContainerRequest{}, as.Error
}
func (as *APIStub) ContainerRequestGet(ctx context.Context, options arvados.GetOptions) (arvados.ContainerRequest, error) {
	as.appendCall(ctx, as.ContainerRequestGet, options)
	return arvados.ContainerRequest{}, as.Error
}
func (as *APIStub) ContainerRequestList(ctx context.Context, options arvados.ListOptions) (arvados.ContainerRequestList, error) {
	as.appendCall(ctx, as.ContainerRequestList, options)
	return arvados.ContainerRequestList{}, as.Error
}
func (as *APIStub) ContainerRequestDelete(ctx context.Context, options arvados.DeleteOptions) (arvados.ContainerRequest, error) {
	as.appendCall(ctx, as.ContainerRequestDelete, options)
	return arvados.ContainerRequest{}, as.Error
}
func (as *APIStub) SpecimenCreate(ctx context.Context, options arvados.CreateOptions) (arvados.Specimen, error) {
	as.appendCall(ctx, as.SpecimenCreate, options)
	return arvados.Specimen{}, as.Error
}
func (as *APIStub) SpecimenUpdate(ctx context.Context, options arvados.UpdateOptions) (arvados.Specimen, error) {
	as.appendCall(ctx, as.SpecimenUpdate, options)
	return arvados.Specimen{}, as.Error
}
func (as *APIStub) SpecimenGet(ctx context.Context, options arvados.GetOptions) (arvados.Specimen, error) {
	as.appendCall(ctx, as.SpecimenGet, options)
	return arvados.Specimen{}, as.Error
}
func (as *APIStub) SpecimenList(ctx context.Context, options arvados.ListOptions) (arvados.SpecimenList, error) {
	as.appendCall(ctx, as.SpecimenList, options)
	return arvados.SpecimenList{}, as.Error
}
func (as *APIStub) SpecimenDelete(ctx context.Context, options arvados.DeleteOptions) (arvados.Specimen, error) {
	as.appendCall(ctx, as.SpecimenDelete, options)
	return arvados.Specimen{}, as.Error
}
func (as *APIStub) UserCreate(ctx context.Context, options arvados.CreateOptions) (arvados.User, error) {
	as.appendCall(ctx, as.UserCreate, options)
	return arvados.User{}, as.Error
}
func (as *APIStub) UserUpdate(ctx context.Context, options arvados.UpdateOptions) (arvados.User, error) {
	as.appendCall(ctx, as.UserUpdate, options)
	return arvados.User{}, as.Error
}
func (as *APIStub) UserUpdateUUID(ctx context.Context, options arvados.UpdateUUIDOptions) (arvados.User, error) {
	as.appendCall(ctx, as.UserUpdateUUID, options)
	return arvados.User{}, as.Error
}
func (as *APIStub) UserActivate(ctx context.Context, options arvados.UserActivateOptions) (arvados.User, error) {
	as.appendCall(ctx, as.UserActivate, options)
	return arvados.User{}, as.Error
}
func (as *APIStub) UserSetup(ctx context.Context, options arvados.UserSetupOptions) (map[string]interface{}, error) {
	as.appendCall(ctx, as.UserSetup, options)
	return nil, as.Error
}
func (as *APIStub) UserUnsetup(ctx context.Context, options arvados.GetOptions) (arvados.User, error) {
	as.appendCall(ctx, as.UserUnsetup, options)
	return arvados.User{}, as.Error
}
func (as *APIStub) UserGet(ctx context.Context, options arvados.GetOptions) (arvados.User, error) {
	as.appendCall(ctx, as.UserGet, options)
	return arvados.User{}, as.Error
}
func (as *APIStub) UserGetCurrent(ctx context.Context, options arvados.GetOptions) (arvados.User, error) {
	as.appendCall(ctx, as.UserGetCurrent, options)
	return arvados.User{}, as.Error
}
func (as *APIStub) UserGetSystem(ctx context.Context, options arvados.GetOptions) (arvados.User, error) {
	as.appendCall(ctx, as.UserGetSystem, options)
	return arvados.User{}, as.Error
}
func (as *APIStub) UserList(ctx context.Context, options arvados.ListOptions) (arvados.UserList, error) {
	as.appendCall(ctx, as.UserList, options)
	return arvados.UserList{}, as.Error
}
func (as *APIStub) UserDelete(ctx context.Context, options arvados.DeleteOptions) (arvados.User, error) {
	as.appendCall(ctx, as.UserDelete, options)
	return arvados.User{}, as.Error
}
func (as *APIStub) UserMerge(ctx context.Context, options arvados.UserMergeOptions) (arvados.User, error) {
	as.appendCall(ctx, as.UserMerge, options)
	return arvados.User{}, as.Error
}
func (as *APIStub) UserBatchUpdate(ctx context.Context, options arvados.UserBatchUpdateOptions) (arvados.UserList, error) {
	as.appendCall(ctx, as.UserBatchUpdate, options)
	return arvados.UserList{}, as.Error
}
func (as *APIStub) UserAuthenticate(ctx context.Context, options arvados.UserAuthenticateOptions) (arvados.APIClientAuthorization, error) {
	as.appendCall(ctx, as.UserAuthenticate, options)
	return arvados.APIClientAuthorization{}, as.Error
}
func (as *APIStub) APIClientAuthorizationCurrent(ctx context.Context, options arvados.GetOptions) (arvados.APIClientAuthorization, error) {
	as.appendCall(ctx, as.APIClientAuthorizationCurrent, options)
	return arvados.APIClientAuthorization{}, as.Error
}

func (as *APIStub) appendCall(ctx context.Context, method interface{}, options interface{}) {
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
