// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package localdb

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"git.arvados.org/arvados.git/lib/controller/railsproxy"
	"git.arvados.org/arvados.git/lib/controller/rpc"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"git.arvados.org/arvados.git/sdk/go/httpserver"
	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"
)

type railsProxy = rpc.Conn

type Conn struct {
	cluster          *arvados.Cluster
	*railsProxy      // handles API methods that aren't defined on Conn itself
	vocabularyCache  *arvados.Vocabulary
	reloadVocabulary bool
	loginController
}

func NewConn(cluster *arvados.Cluster) *Conn {
	railsProxy := railsproxy.NewConn(cluster)
	railsProxy.RedactHostInErrors = true
	conn := Conn{
		cluster:    cluster,
		railsProxy: railsProxy,
	}
	conn.loginController = chooseLoginController(cluster, &conn)
	return &conn
}

func (conn *Conn) checkProperties(ctx context.Context, properties interface{}) error {
	if properties == nil {
		return nil
	}
	var props map[string]interface{}
	switch properties := properties.(type) {
	case string:
		err := json.Unmarshal([]byte(properties), &props)
		if err != nil {
			return err
		}
	case map[string]interface{}:
		props = properties
	default:
		return fmt.Errorf("unexpected properties type %T", properties)
	}
	voc, err := conn.VocabularyGet(ctx)
	if err != nil {
		return err
	}
	err = voc.Check(props)
	if err != nil {
		return httpErrorf(http.StatusBadRequest, voc.Check(props).Error())
	}
	return nil
}

func watchVocabulary(logger logrus.FieldLogger, vocPath string, fn func()) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logger.WithError(err).Error("vocabulary fsnotify setup failed")
		return
	}
	defer watcher.Close()

	err = watcher.Add(vocPath)
	if err != nil {
		logger.WithError(err).Error("vocabulary file watcher failed")
		return
	}

	for {
		select {
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			logger.WithError(err).Warn("vocabulary file watcher error")
		case _, ok := <-watcher.Events:
			if !ok {
				return
			}
			for len(watcher.Events) > 0 {
				<-watcher.Events
			}
			fn()
		}
	}
}

func (conn *Conn) loadVocabularyFile() error {
	vf, err := os.ReadFile(conn.cluster.API.VocabularyPath)
	if err != nil {
		return fmt.Errorf("couldn't read vocabulary file %q: %v", conn.cluster.API.VocabularyPath, err)
	}
	mk := make([]string, 0, len(conn.cluster.Collections.ManagedProperties))
	for k := range conn.cluster.Collections.ManagedProperties {
		mk = append(mk, k)
	}
	voc, err := arvados.NewVocabulary(vf, mk)
	if err != nil {
		return fmt.Errorf("while loading vocabulary file %q: %s", conn.cluster.API.VocabularyPath, err)
	}
	err = voc.Validate()
	if err != nil {
		return fmt.Errorf("while validating vocabulary file %q: %s", conn.cluster.API.VocabularyPath, err)
	}
	conn.vocabularyCache = voc
	return nil
}

// VocabularyGet refreshes the vocabulary cache if necessary and returns it.
func (conn *Conn) VocabularyGet(ctx context.Context) (arvados.Vocabulary, error) {
	if conn.cluster.API.VocabularyPath == "" {
		return arvados.Vocabulary{
			Tags: map[string]arvados.VocabularyTag{},
		}, nil
	}
	logger := ctxlog.FromContext(ctx)
	if conn.vocabularyCache == nil {
		// Initial load of vocabulary file.
		err := conn.loadVocabularyFile()
		if err != nil {
			logger.WithError(err).Error("error loading vocabulary file")
			return arvados.Vocabulary{
				Tags: map[string]arvados.VocabularyTag{},
			}, err
		}
		go watchVocabulary(logger, conn.cluster.API.VocabularyPath, func() {
			logger.Info("vocabulary file changed, it'll be reloaded next time it's needed")
			conn.reloadVocabulary = true
		})
	} else if conn.reloadVocabulary {
		// Requested reload of vocabulary file.
		conn.reloadVocabulary = false
		err := conn.loadVocabularyFile()
		if err != nil {
			logger.WithError(err).Error("error reloading vocabulary file - ignoring")
		} else {
			logger.Info("vocabulary file reloaded successfully")
		}
	}
	return *conn.vocabularyCache, nil
}

// Logout handles the logout of conn giving to the appropriate loginController
func (conn *Conn) Logout(ctx context.Context, opts arvados.LogoutOptions) (arvados.LogoutResponse, error) {
	return conn.loginController.Logout(ctx, opts)
}

// Login handles the login of conn giving to the appropriate loginController
func (conn *Conn) Login(ctx context.Context, opts arvados.LoginOptions) (arvados.LoginResponse, error) {
	return conn.loginController.Login(ctx, opts)
}

// UserAuthenticate handles the User Authentication of conn giving to the appropriate loginController
func (conn *Conn) UserAuthenticate(ctx context.Context, opts arvados.UserAuthenticateOptions) (arvados.APIClientAuthorization, error) {
	return conn.loginController.UserAuthenticate(ctx, opts)
}

func (conn *Conn) GroupContents(ctx context.Context, options arvados.GroupContentsOptions) (arvados.ObjectList, error) {
	// The requested UUID can be a user (virtual home project), which we just pass on to
	// the API server.
	if strings.Index(options.UUID, "-j7d0g-") != 5 {
		return conn.railsProxy.GroupContents(ctx, options)
	}

	var resp arvados.ObjectList

	// Get the group object
	respGroup, err := conn.GroupGet(ctx, arvados.GetOptions{UUID: options.UUID})
	if err != nil {
		return resp, err
	}

	// If the group has groupClass 'filter', apply the filters before getting the contents.
	if respGroup.GroupClass == "filter" {
		if filters, ok := respGroup.Properties["filters"].([]interface{}); ok {
			for _, f := range filters {
				// f is supposed to be a []string
				tmp, ok2 := f.([]interface{})
				if !ok2 || len(tmp) < 3 {
					return resp, fmt.Errorf("filter unparsable: %T, %+v, original field: %T, %+v\n", tmp, tmp, f, f)
				}
				var filter arvados.Filter
				if attr, ok2 := tmp[0].(string); ok2 {
					filter.Attr = attr
				} else {
					return resp, fmt.Errorf("filter unparsable: attribute must be string: %T, %+v, filter: %T, %+v\n", tmp[0], tmp[0], f, f)
				}
				if operator, ok2 := tmp[1].(string); ok2 {
					filter.Operator = operator
				} else {
					return resp, fmt.Errorf("filter unparsable: operator must be string: %T, %+v, filter: %T, %+v\n", tmp[1], tmp[1], f, f)
				}
				filter.Operand = tmp[2]
				options.Filters = append(options.Filters, filter)
			}
		} else {
			return resp, fmt.Errorf("filter unparsable: not an array\n")
		}
		// Use the generic /groups/contents endpoint for filter groups
		options.UUID = ""
	}

	return conn.railsProxy.GroupContents(ctx, options)
}

func httpErrorf(code int, format string, args ...interface{}) error {
	return httpserver.ErrorWithStatus(fmt.Errorf(format, args...), code)
}
