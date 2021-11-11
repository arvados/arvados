// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package router

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"strings"

	"git.arvados.org/arvados.git/lib/controller/api"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/auth"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"git.arvados.org/arvados.git/sdk/go/httpserver"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type router struct {
	mux     *mux.Router
	backend arvados.API
	config  Config
}

type Config struct {
	// Return an error if request body exceeds this size. 0 means
	// unlimited.
	MaxRequestSize int

	// If wrapCalls is not nil, it is called once for each API
	// method, and the returned method is used in its place. This
	// can be used to install hooks before and after each API call
	// and alter responses; see localdb.WrapCallsInTransaction for
	// an example.
	WrapCalls func(api.RoutableFunc) api.RoutableFunc
}

// New returns a new router (which implements the http.Handler
// interface) that serves requests by calling Arvados API methods on
// the given backend.
func New(backend arvados.API, config Config) *router {
	rtr := &router{
		mux:     mux.NewRouter(),
		backend: backend,
		config:  config,
	}
	rtr.addRoutes()
	return rtr
}

func (rtr *router) addRoutes() {
	for _, route := range []struct {
		endpoint    arvados.APIEndpoint
		defaultOpts func() interface{}
		exec        api.RoutableFunc
	}{
		{
			arvados.EndpointConfigGet,
			func() interface{} { return &struct{}{} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.backend.ConfigGet(ctx)
			},
		},
		{
			arvados.EndpointVocabularyGet,
			func() interface{} { return &struct{}{} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.backend.VocabularyGet(ctx)
			},
		},
		{
			arvados.EndpointLogin,
			func() interface{} { return &arvados.LoginOptions{} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.backend.Login(ctx, *opts.(*arvados.LoginOptions))
			},
		},
		{
			arvados.EndpointLogout,
			func() interface{} { return &arvados.LogoutOptions{} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.backend.Logout(ctx, *opts.(*arvados.LogoutOptions))
			},
		},
		{
			arvados.EndpointCollectionCreate,
			func() interface{} { return &arvados.CreateOptions{} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.backend.CollectionCreate(ctx, *opts.(*arvados.CreateOptions))
			},
		},
		{
			arvados.EndpointCollectionUpdate,
			func() interface{} { return &arvados.UpdateOptions{} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.backend.CollectionUpdate(ctx, *opts.(*arvados.UpdateOptions))
			},
		},
		{
			arvados.EndpointCollectionGet,
			func() interface{} { return &arvados.GetOptions{} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.backend.CollectionGet(ctx, *opts.(*arvados.GetOptions))
			},
		},
		{
			arvados.EndpointCollectionList,
			func() interface{} { return &arvados.ListOptions{Limit: -1} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.backend.CollectionList(ctx, *opts.(*arvados.ListOptions))
			},
		},
		{
			arvados.EndpointCollectionProvenance,
			func() interface{} { return &arvados.GetOptions{} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.backend.CollectionProvenance(ctx, *opts.(*arvados.GetOptions))
			},
		},
		{
			arvados.EndpointCollectionUsedBy,
			func() interface{} { return &arvados.GetOptions{} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.backend.CollectionUsedBy(ctx, *opts.(*arvados.GetOptions))
			},
		},
		{
			arvados.EndpointCollectionDelete,
			func() interface{} { return &arvados.DeleteOptions{} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.backend.CollectionDelete(ctx, *opts.(*arvados.DeleteOptions))
			},
		},
		{
			arvados.EndpointCollectionTrash,
			func() interface{} { return &arvados.DeleteOptions{} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.backend.CollectionTrash(ctx, *opts.(*arvados.DeleteOptions))
			},
		},
		{
			arvados.EndpointCollectionUntrash,
			func() interface{} { return &arvados.UntrashOptions{} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.backend.CollectionUntrash(ctx, *opts.(*arvados.UntrashOptions))
			},
		},
		{
			arvados.EndpointContainerCreate,
			func() interface{} { return &arvados.CreateOptions{} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.backend.ContainerCreate(ctx, *opts.(*arvados.CreateOptions))
			},
		},
		{
			arvados.EndpointContainerUpdate,
			func() interface{} { return &arvados.UpdateOptions{} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.backend.ContainerUpdate(ctx, *opts.(*arvados.UpdateOptions))
			},
		},
		{
			arvados.EndpointContainerGet,
			func() interface{} { return &arvados.GetOptions{} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.backend.ContainerGet(ctx, *opts.(*arvados.GetOptions))
			},
		},
		{
			arvados.EndpointContainerList,
			func() interface{} { return &arvados.ListOptions{Limit: -1} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.backend.ContainerList(ctx, *opts.(*arvados.ListOptions))
			},
		},
		{
			arvados.EndpointContainerDelete,
			func() interface{} { return &arvados.DeleteOptions{} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.backend.ContainerDelete(ctx, *opts.(*arvados.DeleteOptions))
			},
		},
		{
			arvados.EndpointContainerRequestCreate,
			func() interface{} { return &arvados.CreateOptions{} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.backend.ContainerRequestCreate(ctx, *opts.(*arvados.CreateOptions))
			},
		},
		{
			arvados.EndpointContainerRequestUpdate,
			func() interface{} { return &arvados.UpdateOptions{} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.backend.ContainerRequestUpdate(ctx, *opts.(*arvados.UpdateOptions))
			},
		},
		{
			arvados.EndpointContainerRequestGet,
			func() interface{} { return &arvados.GetOptions{} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.backend.ContainerRequestGet(ctx, *opts.(*arvados.GetOptions))
			},
		},
		{
			arvados.EndpointContainerRequestList,
			func() interface{} { return &arvados.ListOptions{Limit: -1} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.backend.ContainerRequestList(ctx, *opts.(*arvados.ListOptions))
			},
		},
		{
			arvados.EndpointContainerRequestDelete,
			func() interface{} { return &arvados.DeleteOptions{} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.backend.ContainerRequestDelete(ctx, *opts.(*arvados.DeleteOptions))
			},
		},
		{
			arvados.EndpointContainerLock,
			func() interface{} {
				return &arvados.GetOptions{Select: []string{"uuid", "state", "priority", "auth_uuid", "locked_by_uuid"}}
			},
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.backend.ContainerLock(ctx, *opts.(*arvados.GetOptions))
			},
		},
		{
			arvados.EndpointContainerUnlock,
			func() interface{} {
				return &arvados.GetOptions{Select: []string{"uuid", "state", "priority", "auth_uuid", "locked_by_uuid"}}
			},
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.backend.ContainerUnlock(ctx, *opts.(*arvados.GetOptions))
			},
		},
		{
			arvados.EndpointContainerSSH,
			func() interface{} { return &arvados.ContainerSSHOptions{} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.backend.ContainerSSH(ctx, *opts.(*arvados.ContainerSSHOptions))
			},
		},
		{
			arvados.EndpointGroupCreate,
			func() interface{} { return &arvados.CreateOptions{} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.backend.GroupCreate(ctx, *opts.(*arvados.CreateOptions))
			},
		},
		{
			arvados.EndpointGroupUpdate,
			func() interface{} { return &arvados.UpdateOptions{} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.backend.GroupUpdate(ctx, *opts.(*arvados.UpdateOptions))
			},
		},
		{
			arvados.EndpointGroupList,
			func() interface{} { return &arvados.ListOptions{Limit: -1} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.backend.GroupList(ctx, *opts.(*arvados.ListOptions))
			},
		},
		{
			arvados.EndpointGroupContents,
			func() interface{} { return &arvados.GroupContentsOptions{Limit: -1} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.backend.GroupContents(ctx, *opts.(*arvados.GroupContentsOptions))
			},
		},
		{
			arvados.EndpointGroupContentsUUIDInPath,
			func() interface{} { return &arvados.GroupContentsOptions{Limit: -1} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.backend.GroupContents(ctx, *opts.(*arvados.GroupContentsOptions))
			},
		},
		{
			arvados.EndpointGroupShared,
			func() interface{} { return &arvados.ListOptions{Limit: -1} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.backend.GroupShared(ctx, *opts.(*arvados.ListOptions))
			},
		},
		{
			arvados.EndpointGroupGet,
			func() interface{} { return &arvados.GetOptions{} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.backend.GroupGet(ctx, *opts.(*arvados.GetOptions))
			},
		},
		{
			arvados.EndpointGroupDelete,
			func() interface{} { return &arvados.DeleteOptions{} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.backend.GroupDelete(ctx, *opts.(*arvados.DeleteOptions))
			},
		},
		{
			arvados.EndpointGroupTrash,
			func() interface{} { return &arvados.DeleteOptions{} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.backend.GroupTrash(ctx, *opts.(*arvados.DeleteOptions))
			},
		},
		{
			arvados.EndpointGroupUntrash,
			func() interface{} { return &arvados.UntrashOptions{} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.backend.GroupUntrash(ctx, *opts.(*arvados.UntrashOptions))
			},
		},
		{
			arvados.EndpointLinkCreate,
			func() interface{} { return &arvados.CreateOptions{} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.backend.LinkCreate(ctx, *opts.(*arvados.CreateOptions))
			},
		},
		{
			arvados.EndpointLinkUpdate,
			func() interface{} { return &arvados.UpdateOptions{} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.backend.LinkUpdate(ctx, *opts.(*arvados.UpdateOptions))
			},
		},
		{
			arvados.EndpointLinkList,
			func() interface{} { return &arvados.ListOptions{Limit: -1} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.backend.LinkList(ctx, *opts.(*arvados.ListOptions))
			},
		},
		{
			arvados.EndpointLinkGet,
			func() interface{} { return &arvados.GetOptions{} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.backend.LinkGet(ctx, *opts.(*arvados.GetOptions))
			},
		},
		{
			arvados.EndpointLinkDelete,
			func() interface{} { return &arvados.DeleteOptions{} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.backend.LinkDelete(ctx, *opts.(*arvados.DeleteOptions))
			},
		},
		{
			arvados.EndpointSpecimenCreate,
			func() interface{} { return &arvados.CreateOptions{} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.backend.SpecimenCreate(ctx, *opts.(*arvados.CreateOptions))
			},
		},
		{
			arvados.EndpointSpecimenUpdate,
			func() interface{} { return &arvados.UpdateOptions{} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.backend.SpecimenUpdate(ctx, *opts.(*arvados.UpdateOptions))
			},
		},
		{
			arvados.EndpointSpecimenGet,
			func() interface{} { return &arvados.GetOptions{} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.backend.SpecimenGet(ctx, *opts.(*arvados.GetOptions))
			},
		},
		{
			arvados.EndpointSpecimenList,
			func() interface{} { return &arvados.ListOptions{Limit: -1} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.backend.SpecimenList(ctx, *opts.(*arvados.ListOptions))
			},
		},
		{
			arvados.EndpointSpecimenDelete,
			func() interface{} { return &arvados.DeleteOptions{} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.backend.SpecimenDelete(ctx, *opts.(*arvados.DeleteOptions))
			},
		},
		{
			arvados.EndpointUserCreate,
			func() interface{} { return &arvados.CreateOptions{} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.backend.UserCreate(ctx, *opts.(*arvados.CreateOptions))
			},
		},
		{
			arvados.EndpointUserMerge,
			func() interface{} { return &arvados.UserMergeOptions{} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.backend.UserMerge(ctx, *opts.(*arvados.UserMergeOptions))
			},
		},
		{
			arvados.EndpointUserActivate,
			func() interface{} { return &arvados.UserActivateOptions{} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.backend.UserActivate(ctx, *opts.(*arvados.UserActivateOptions))
			},
		},
		{
			arvados.EndpointUserSetup,
			func() interface{} { return &arvados.UserSetupOptions{} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.backend.UserSetup(ctx, *opts.(*arvados.UserSetupOptions))
			},
		},
		{
			arvados.EndpointUserUnsetup,
			func() interface{} { return &arvados.GetOptions{} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.backend.UserUnsetup(ctx, *opts.(*arvados.GetOptions))
			},
		},
		{
			arvados.EndpointUserGetCurrent,
			func() interface{} { return &arvados.GetOptions{} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.backend.UserGetCurrent(ctx, *opts.(*arvados.GetOptions))
			},
		},
		{
			arvados.EndpointUserGetSystem,
			func() interface{} { return &arvados.GetOptions{} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.backend.UserGetSystem(ctx, *opts.(*arvados.GetOptions))
			},
		},
		{
			arvados.EndpointUserGet,
			func() interface{} { return &arvados.GetOptions{} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.backend.UserGet(ctx, *opts.(*arvados.GetOptions))
			},
		},
		{
			arvados.EndpointUserUpdate,
			func() interface{} { return &arvados.UpdateOptions{} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.backend.UserUpdate(ctx, *opts.(*arvados.UpdateOptions))
			},
		},
		{
			arvados.EndpointUserList,
			func() interface{} { return &arvados.ListOptions{Limit: -1} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.backend.UserList(ctx, *opts.(*arvados.ListOptions))
			},
		},
		{
			arvados.EndpointUserBatchUpdate,
			func() interface{} { return &arvados.UserBatchUpdateOptions{} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.backend.UserBatchUpdate(ctx, *opts.(*arvados.UserBatchUpdateOptions))
			},
		},
		{
			arvados.EndpointUserDelete,
			func() interface{} { return &arvados.DeleteOptions{} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.backend.UserDelete(ctx, *opts.(*arvados.DeleteOptions))
			},
		},
		{
			arvados.EndpointUserAuthenticate,
			func() interface{} { return &arvados.UserAuthenticateOptions{} },
			func(ctx context.Context, opts interface{}) (interface{}, error) {
				return rtr.backend.UserAuthenticate(ctx, *opts.(*arvados.UserAuthenticateOptions))
			},
		},
	} {
		exec := route.exec
		if rtr.config.WrapCalls != nil {
			exec = rtr.config.WrapCalls(exec)
		}
		rtr.addRoute(route.endpoint, route.defaultOpts, exec)
	}
	rtr.mux.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		httpserver.Errors(w, []string{"API endpoint not found"}, http.StatusNotFound)
	})
	rtr.mux.MethodNotAllowedHandler = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		httpserver.Errors(w, []string{"API endpoint not found"}, http.StatusMethodNotAllowed)
	})
}

var altMethod = map[string]string{
	"PATCH": "PUT",  // Accept PUT as a synonym for PATCH
	"GET":   "HEAD", // Accept HEAD at any GET route
}

func (rtr *router) addRoute(endpoint arvados.APIEndpoint, defaultOpts func() interface{}, exec api.RoutableFunc) {
	methods := []string{endpoint.Method}
	if alt, ok := altMethod[endpoint.Method]; ok {
		methods = append(methods, alt)
	}
	rtr.mux.Methods(methods...).Path("/" + endpoint.Path).HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
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
		err = creds.LoadTokensFromHTTPRequestBody(req)
		if err != nil {
			rtr.sendError(w, fmt.Errorf("error loading tokens from request body: %s", err))
			return
		}
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
		rtr.sendResponse(w, req, resp, respOpts)
	})
}

func (rtr *router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch strings.SplitN(strings.TrimLeft(r.URL.Path, "/"), "/", 2)[0] {
	case "login", "logout", "auth":
	default:
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, PUT, POST, PATCH, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Http-Method-Override")
		w.Header().Set("Access-Control-Max-Age", "86486400")
	}
	if r.Method == "OPTIONS" {
		return
	}
	if r.Body != nil {
		// Wrap r.Body in a http.MaxBytesReader(), otherwise
		// r.ParseForm() uses a default max request body size
		// of 10 megabytes. Note we rely on the Nginx
		// configuration to enforce the real max body size.
		max := int64(rtr.config.MaxRequestSize)
		if max < 1 {
			max = math.MaxInt64 - 1
		}
		r.Body = http.MaxBytesReader(w, r.Body, max)
	}
	if r.Method == "POST" {
		err := r.ParseForm()
		if err != nil {
			if err.Error() == "http: request body too large" {
				err = httpError(http.StatusRequestEntityTooLarge, err)
			}
			rtr.sendError(w, err)
			return
		}
		if m := r.FormValue("_method"); m != "" {
			r2 := *r
			r = &r2
			r.Method = m
		} else if m = r.Header.Get("X-Http-Method-Override"); m != "" {
			r2 := *r
			r = &r2
			r.Method = m
		}
	}
	rtr.mux.ServeHTTP(w, r)
}
