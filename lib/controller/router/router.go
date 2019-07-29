// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package router

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/auth"
	"git.curoverse.com/arvados.git/sdk/go/ctxlog"
	"git.curoverse.com/arvados.git/sdk/go/httpserver"
	"github.com/julienschmidt/httprouter"
	"github.com/sirupsen/logrus"
)

type router struct {
	mux *httprouter.Router
	fed arvados.API
}

func New(fed arvados.API) *router {
	rtr := &router{
		mux: httprouter.New(),
		fed: fed,
	}
	rtr.addRoutes()
	return rtr
}

type routableFunc func(ctx context.Context, opts interface{}) (interface{}, error)

func (rtr *router) addRoutes() {
	for _, route := range []struct {
		endpoint    arvados.APIEndpoint
		defaultOpts func() interface{}
		exec        routableFunc
	}{
		{
			arvados.EndpointConfigGet,
			func() interface{} { return &struct{}{} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.fed.ConfigGet(ctx)
			},
		},
		{
			arvados.EndpointCollectionCreate,
			func() interface{} { return &arvados.CreateOptions{} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.fed.CollectionCreate(ctx, *opts.(*arvados.CreateOptions))
			},
		},
		{
			arvados.EndpointCollectionUpdate,
			func() interface{} { return &arvados.UpdateOptions{} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.fed.CollectionUpdate(ctx, *opts.(*arvados.UpdateOptions))
			},
		},
		{
			arvados.EndpointCollectionGet,
			func() interface{} { return &arvados.GetOptions{} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.fed.CollectionGet(ctx, *opts.(*arvados.GetOptions))
			},
		},
		{
			arvados.EndpointCollectionList,
			func() interface{} { return &arvados.ListOptions{Limit: -1} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.fed.CollectionList(ctx, *opts.(*arvados.ListOptions))
			},
		},
		{
			arvados.EndpointCollectionProvenance,
			func() interface{} { return &arvados.GetOptions{} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.fed.CollectionProvenance(ctx, *opts.(*arvados.GetOptions))
			},
		},
		{
			arvados.EndpointCollectionUsedBy,
			func() interface{} { return &arvados.GetOptions{} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.fed.CollectionUsedBy(ctx, *opts.(*arvados.GetOptions))
			},
		},
		{
			arvados.EndpointCollectionDelete,
			func() interface{} { return &arvados.DeleteOptions{} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.fed.CollectionDelete(ctx, *opts.(*arvados.DeleteOptions))
			},
		},
		{
			arvados.EndpointCollectionTrash,
			func() interface{} { return &arvados.DeleteOptions{} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.fed.CollectionTrash(ctx, *opts.(*arvados.DeleteOptions))
			},
		},
		{
			arvados.EndpointCollectionUntrash,
			func() interface{} { return &arvados.UntrashOptions{} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.fed.CollectionUntrash(ctx, *opts.(*arvados.UntrashOptions))
			},
		},
		{
			arvados.EndpointContainerCreate,
			func() interface{} { return &arvados.CreateOptions{} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.fed.ContainerCreate(ctx, *opts.(*arvados.CreateOptions))
			},
		},
		{
			arvados.EndpointContainerUpdate,
			func() interface{} { return &arvados.UpdateOptions{} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.fed.ContainerUpdate(ctx, *opts.(*arvados.UpdateOptions))
			},
		},
		{
			arvados.EndpointContainerGet,
			func() interface{} { return &arvados.GetOptions{} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.fed.ContainerGet(ctx, *opts.(*arvados.GetOptions))
			},
		},
		{
			arvados.EndpointContainerList,
			func() interface{} { return &arvados.ListOptions{Limit: -1} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.fed.ContainerList(ctx, *opts.(*arvados.ListOptions))
			},
		},
		{
			arvados.EndpointContainerDelete,
			func() interface{} { return &arvados.DeleteOptions{} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.fed.ContainerDelete(ctx, *opts.(*arvados.DeleteOptions))
			},
		},
		{
			arvados.EndpointContainerLock,
			func() interface{} {
				return &arvados.GetOptions{Select: []string{"uuid", "state", "priority", "auth_uuid", "locked_by_uuid"}}
			},
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.fed.ContainerLock(ctx, *opts.(*arvados.GetOptions))
			},
		},
		{
			arvados.EndpointContainerUnlock,
			func() interface{} {
				return &arvados.GetOptions{Select: []string{"uuid", "state", "priority", "auth_uuid", "locked_by_uuid"}}
			},
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.fed.ContainerUnlock(ctx, *opts.(*arvados.GetOptions))
			},
		},
		{
			arvados.EndpointSpecimenCreate,
			func() interface{} { return &arvados.CreateOptions{} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.fed.SpecimenCreate(ctx, *opts.(*arvados.CreateOptions))
			},
		},
		{
			arvados.EndpointSpecimenUpdate,
			func() interface{} { return &arvados.UpdateOptions{} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.fed.SpecimenUpdate(ctx, *opts.(*arvados.UpdateOptions))
			},
		},
		{
			arvados.EndpointSpecimenGet,
			func() interface{} { return &arvados.GetOptions{} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.fed.SpecimenGet(ctx, *opts.(*arvados.GetOptions))
			},
		},
		{
			arvados.EndpointSpecimenList,
			func() interface{} { return &arvados.ListOptions{Limit: -1} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.fed.SpecimenList(ctx, *opts.(*arvados.ListOptions))
			},
		},
		{
			arvados.EndpointSpecimenDelete,
			func() interface{} { return &arvados.DeleteOptions{} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.fed.SpecimenDelete(ctx, *opts.(*arvados.DeleteOptions))
			},
		},
	} {
		rtr.addRoute(route.endpoint, route.defaultOpts, route.exec)
		if route.endpoint.Method == "PATCH" {
			// Accept PUT as a synonym for PATCH.
			endpointPUT := route.endpoint
			endpointPUT.Method = "PUT"
			rtr.addRoute(endpointPUT, route.defaultOpts, route.exec)
		}
	}
	rtr.mux.NotFound = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		httpserver.Errors(w, []string{"API endpoint not found"}, http.StatusNotFound)
	})
	rtr.mux.MethodNotAllowed = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		httpserver.Errors(w, []string{"API endpoint not found"}, http.StatusMethodNotAllowed)
	})
}

func (rtr *router) addRoute(endpoint arvados.APIEndpoint, defaultOpts func() interface{}, exec routableFunc) {
	rtr.mux.HandlerFunc(endpoint.Method, "/"+endpoint.Path, func(w http.ResponseWriter, req *http.Request) {
		logger := ctxlog.FromContext(req.Context())
		params, err := rtr.loadRequestParams(req, endpoint.AttrsKey)
		if err != nil {
			logger.WithFields(logrus.Fields{
				"req":      req,
				"method":   endpoint.Method,
				"endpoint": endpoint,
			}).WithError(err).Debug("error loading request params")
			rtr.sendError(w, err)
			return
		}
		opts := defaultOpts()
		err = rtr.transcode(params, opts)
		if err != nil {
			logger.WithField("params", params).WithError(err).Debugf("error transcoding params to %T", opts)
			rtr.sendError(w, err)
			return
		}
		respOpts, err := rtr.responseOptions(opts)
		if err != nil {
			logger.WithField("opts", opts).WithError(err).Debugf("error getting response options from %T", opts)
			rtr.sendError(w, err)
			return
		}

		creds := auth.CredentialsFromRequest(req)
		if rt, _ := params["reader_tokens"].([]interface{}); len(rt) > 0 {
			for _, t := range rt {
				if t, ok := t.(string); ok {
					creds.Tokens = append(creds.Tokens, t)
				}
			}
		}
		ctx := auth.NewContext(req.Context(), creds)
		ctx = arvados.ContextWithRequestID(ctx, req.Header.Get("X-Request-Id"))
		logger.WithFields(logrus.Fields{
			"apiEndpoint": endpoint,
			"apiOptsType": fmt.Sprintf("%T", opts),
			"apiOpts":     opts,
		}).Debug("exec")
		resp, err := exec(ctx, opts)
		if err != nil {
			logger.WithError(err).Debugf("returning error type %T", err)
			rtr.sendError(w, err)
			return
		}
		rtr.sendResponse(w, resp, respOpts)
	})
}

func (rtr *router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch strings.SplitN(strings.TrimLeft(r.URL.Path, "/"), "/", 2)[0] {
	case "login", "logout", "auth":
	default:
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, PUT, POST, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		w.Header().Set("Access-Control-Max-Age", "86486400")
	}
	if r.Method == "OPTIONS" {
		return
	}
	r.ParseForm()
	if m := r.FormValue("_method"); m != "" {
		r2 := *r
		r = &r2
		r.Method = m
	}
	rtr.mux.ServeHTTP(w, r)
}
