// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepweb

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"git.arvados.org/arvados.git/lib/config"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadosclient"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"git.arvados.org/arvados.git/sdk/go/auth"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"git.arvados.org/arvados.git/sdk/go/keepclient"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/html"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&UnitSuite{})

func init() {
	arvados.DebugLocksPanicMode = true
}

type UnitSuite struct {
	cluster *arvados.Cluster
	handler *handler
}

func (s *UnitSuite) SetUpTest(c *check.C) {
	logger := ctxlog.TestLogger(c)
	ldr := config.NewLoader(&bytes.Buffer{}, logger)
	cfg, err := ldr.Load()
	c.Assert(err, check.IsNil)
	cc, err := cfg.GetCluster("")
	c.Assert(err, check.IsNil)
	s.cluster = cc
	s.handler = &handler{
		Cluster: cc,
		Cache: cache{
			cluster:  cc,
			logger:   logger,
			registry: prometheus.NewRegistry(),
		},
		metrics: newMetrics(prometheus.NewRegistry()),
	}
}

func newCollection(collID string) *arvados.Collection {
	coll := &arvados.Collection{UUID: collID}
	manifestKey := collID
	if pdh, ok := arvadostest.TestCollectionUUIDToPDH[collID]; ok {
		coll.PortableDataHash = pdh
		manifestKey = pdh
	}
	if mtext, ok := arvadostest.TestCollectionPDHToManifest[manifestKey]; ok {
		coll.ManifestText = mtext
	}
	return coll
}

func newRequest(method, urlStr string) *http.Request {
	u := mustParseURL(urlStr)
	return &http.Request{
		Method:     method,
		Host:       u.Host,
		URL:        u,
		RequestURI: u.RequestURI(),
		RemoteAddr: "10.20.30.40:56789",
		Header:     http.Header{},
	}
}

func newLoggerAndContext() (*bytes.Buffer, context.Context) {
	var logbuf bytes.Buffer
	logger := logrus.New()
	logger.Out = &logbuf
	return &logbuf, ctxlog.Context(context.Background(), logger)
}

func (s *UnitSuite) TestLogEventTypes(c *check.C) {
	collURL := "http://keep-web.example/c=" + arvadostest.FooCollection
	for method, expected := range map[string]string{
		"GET":  "file_download",
		"POST": "file_upload",
		"PUT":  "file_upload",
	} {
		filePath := "/" + method
		req := newRequest(method, collURL+filePath)
		actual := newFileEventLog(s.handler, req, filePath, nil, nil, "")
		if !c.Check(actual, check.NotNil) {
			continue
		}
		c.Check(actual.eventType, check.Equals, expected)
	}
}

func (s *UnitSuite) TestUnloggedEventTypes(c *check.C) {
	collURL := "http://keep-web.example/c=" + arvadostest.FooCollection
	for _, method := range []string{"DELETE", "HEAD", "OPTIONS", "PATCH"} {
		filePath := "/" + method
		req := newRequest(method, collURL+filePath)
		actual := newFileEventLog(s.handler, req, filePath, nil, nil, "")
		c.Check(actual, check.IsNil,
			check.Commentf("%s request made a log event", method))
	}
}

func (s *UnitSuite) TestLogFilePath(c *check.C) {
	coll := newCollection(arvadostest.FooCollection)
	collURL := "http://keep-web.example/c=" + arvadostest.FooCollection
	for _, filePath := range []string{"/foo", "/Foo", "/foo/bar"} {
		req := newRequest("GET", collURL+filePath)
		actual := newFileEventLog(s.handler, req, filePath, coll, nil, "")
		if !c.Check(actual, check.NotNil) {
			continue
		}
		c.Check(actual.collFilePath, check.Equals, filePath)
	}
}

func (s *UnitSuite) TestLogRemoteAddr(c *check.C) {
	collURL := "http://keep-web.example/c=" + arvadostest.FooCollection
	filePath := "/foo"
	req := newRequest("GET", collURL+filePath)

	for _, addr := range []string{"10.20.30.55", "192.168.144.120", "192.0.2.4"} {
		req.RemoteAddr = addr + ":57914"
		actual := newFileEventLog(s.handler, req, filePath, nil, nil, "")
		if !c.Check(actual, check.NotNil) {
			continue
		}
		c.Check(actual.clientAddr, check.Equals, addr)
	}

	for _, addr := range []string{"100::20:30:40", "2001:db8::90:100", "3fff::30"} {
		req.RemoteAddr = fmt.Sprintf("[%s]:57916", addr)
		actual := newFileEventLog(s.handler, req, filePath, nil, nil, "")
		if !c.Check(actual, check.NotNil) {
			continue
		}
		c.Check(actual.clientAddr, check.Equals, addr)
	}
}

func (s *UnitSuite) TestLogXForwardedFor(c *check.C) {
	collURL := "http://keep-web.example/c=" + arvadostest.FooCollection
	filePath := "/foo"
	req := newRequest("GET", collURL+filePath)
	for xff, expected := range map[string]string{
		"10.20.30.55":                          "10.20.30.55",
		"192.168.144.120, 10.20.30.120":        "10.20.30.120",
		"192.0.2.4, 192.0.2.6, 192.0.2.8":      "192.0.2.8",
		"192.0.2.4,192.168.2.4":                "192.168.2.4",
		"10.20.30.60,192.168.144.40,192.0.2.4": "192.0.2.4",
		"100::20:30:50":                        "100::20:30:50",
		"2001:db8::80:90, 100::100":            "100::100",
		"3fff::ff, 3fff::ee, 3fff::fe":         "3fff::fe",
		"3fff::3f,100::1000":                   "100::1000",
		"2001:db8::88,100::88,3fff::88":        "3fff::88",
		"10.20.30.60, 2001:db8::60":            "2001:db8::60",
		"2001:db8::20,10.20.30.20":             "10.20.30.20",
		", 10.20.30.123, 100::123":             "100::123",
		",100::321,10.30.20.10":                "10.30.20.10",
	} {
		req.Header.Set("X-Forwarded-For", xff)
		actual := newFileEventLog(s.handler, req, filePath, nil, nil, "")
		if !c.Check(actual, check.NotNil) {
			continue
		}
		c.Check(actual.clientAddr, check.Equals, expected)
	}
}

func (s *UnitSuite) TestLogXForwardedForMalformed(c *check.C) {
	collURL := "http://keep-web.example/c=" + arvadostest.FooCollection
	filePath := "/foo"
	req := newRequest("GET", collURL+filePath)
	for _, xff := range []string{"", ",", "10.20,30.40", "foo, bar"} {
		req.Header.Set("X-Forwarded-For", xff)
		actual := newFileEventLog(s.handler, req, filePath, nil, nil, "")
		if !c.Check(actual, check.NotNil) {
			continue
		}
		c.Check(actual.clientAddr, check.Equals, "10.20.30.40")
	}
}

func (s *UnitSuite) TestLogXForwardedForMultivalue(c *check.C) {
	collURL := "http://keep-web.example/c=" + arvadostest.FooCollection
	filePath := "/foo"
	req := newRequest("GET", collURL+filePath)
	req.Header.Set("X-Forwarded-For", ", ")
	req.Header.Add("X-Forwarded-For", "2001:db8::db9:dbd")
	req.Header.Add("X-Forwarded-For", "10.20.30.90")
	actual := newFileEventLog(s.handler, req, filePath, nil, nil, "")
	c.Assert(actual, check.NotNil)
	c.Check(actual.clientAddr, check.Equals, "10.20.30.90")
}

func (s *UnitSuite) TestLogClientAddressCanonicalization(c *check.C) {
	collURL := "http://keep-web.example/c=" + arvadostest.FooCollection
	filePath := "/foo"
	req := newRequest("GET", collURL+filePath)
	expected := "2001:db8::12:0"

	req.RemoteAddr = "[2001:db8::012:0000]:57918"
	a := newFileEventLog(s.handler, req, filePath, nil, nil, "")
	c.Assert(a, check.NotNil)
	c.Check(a.clientAddr, check.Equals, expected)

	req.RemoteAddr = "10.20.30.40:57919"
	req.Header.Set("X-Forwarded-For", "2001:db8:0::0:12:00")
	b := newFileEventLog(s.handler, req, filePath, nil, nil, "")
	c.Assert(b, check.NotNil)
	c.Check(b.clientAddr, check.Equals, expected)
}

func (s *UnitSuite) TestLogAnonymousUser(c *check.C) {
	collURL := "http://keep-web.example/c=" + arvadostest.FooCollection
	filePath := "/foo"
	req := newRequest("GET", collURL+filePath)
	actual := newFileEventLog(s.handler, req, filePath, nil, nil, arvadostest.AnonymousToken)
	c.Assert(actual, check.NotNil)
	c.Check(actual.userUUID, check.Equals, s.handler.Cluster.ClusterID+"-tpzed-anonymouspublic")
	c.Check(actual.userFullName, check.Equals, "")
	c.Check(actual.clientToken, check.Equals, arvadostest.AnonymousToken)
}

func (s *UnitSuite) TestLogUser(c *check.C) {
	collURL := "http://keep-web.example/c=" + arvadostest.FooCollection
	for _, trial := range []struct{ uuid, fullName, token string }{
		{arvadostest.ActiveUserUUID, "Active User", arvadostest.ActiveToken},
		{arvadostest.SpectatorUserUUID, "Spectator User", arvadostest.SpectatorToken},
	} {
		filePath := "/" + trial.uuid
		req := newRequest("GET", collURL+filePath)
		user := &arvados.User{
			UUID:     trial.uuid,
			FullName: trial.fullName,
		}
		actual := newFileEventLog(s.handler, req, filePath, nil, user, trial.token)
		if !c.Check(actual, check.NotNil) {
			continue
		}
		c.Check(actual.userUUID, check.Equals, trial.uuid)
		c.Check(actual.userFullName, check.Equals, trial.fullName)
		c.Check(actual.clientToken, check.Equals, trial.token)
	}
}

func (s *UnitSuite) TestLogCollectionByUUID(c *check.C) {
	for collUUID, collPDH := range arvadostest.TestCollectionUUIDToPDH {
		collURL := "http://keep-web.example/c=" + collUUID
		filePath := "/" + collUUID
		req := newRequest("GET", collURL+filePath)
		coll := newCollection(collUUID)
		actual := newFileEventLog(s.handler, req, filePath, coll, nil, "")
		if !c.Check(actual, check.NotNil) {
			continue
		}
		c.Check(actual.collUUID, check.Equals, collUUID)
		c.Check(actual.collPDH, check.Equals, collPDH)
	}
}

func (s *UnitSuite) TestLogCollectionByPDH(c *check.C) {
	for _, collPDH := range arvadostest.TestCollectionUUIDToPDH {
		collURL := "http://keep-web.example/c=" + collPDH
		filePath := "/PDHFile"
		req := newRequest("GET", collURL+filePath)
		coll := newCollection(collPDH)
		actual := newFileEventLog(s.handler, req, filePath, coll, nil, "")
		if !c.Check(actual, check.NotNil) {
			continue
		}
		c.Check(actual.collPDH, check.Equals, collPDH)
		c.Check(actual.collUUID, check.Equals, "")
	}
}

func (s *UnitSuite) TestLogGETUUIDAsDict(c *check.C) {
	filePath := "/foo"
	reqPath := "/c=" + arvadostest.FooCollection + filePath
	req := newRequest("GET", "http://keep-web.example"+reqPath)
	coll := newCollection(arvadostest.FooCollection)
	logEvent := newFileEventLog(s.handler, req, filePath, coll, nil, "")
	c.Assert(logEvent, check.NotNil)
	c.Check(logEvent.asDict(), check.DeepEquals, arvadosclient.Dict{
		"event_type":  "file_download",
		"object_uuid": s.handler.Cluster.ClusterID + "-tpzed-anonymouspublic",
		"properties": arvadosclient.Dict{
			"reqPath":              reqPath,
			"collection_uuid":      arvadostest.FooCollection,
			"collection_file_path": filePath,
			"portable_data_hash":   arvadostest.FooCollectionPDH,
		},
	})
}

func (s *UnitSuite) TestLogGETPDHAsDict(c *check.C) {
	filePath := "/Foo"
	reqPath := "/c=" + arvadostest.FooCollectionPDH + filePath
	req := newRequest("GET", "http://keep-web.example"+reqPath)
	coll := newCollection(arvadostest.FooCollectionPDH)
	user := &arvados.User{
		UUID:     arvadostest.ActiveUserUUID,
		FullName: "Active User",
	}
	logEvent := newFileEventLog(s.handler, req, filePath, coll, user, "")
	c.Assert(logEvent, check.NotNil)
	c.Check(logEvent.asDict(), check.DeepEquals, arvadosclient.Dict{
		"event_type":  "file_download",
		"object_uuid": arvadostest.ActiveUserUUID,
		"properties": arvadosclient.Dict{
			"reqPath":              reqPath,
			"portable_data_hash":   arvadostest.FooCollectionPDH,
			"collection_uuid":      "",
			"collection_file_path": filePath,
		},
	})
}

func (s *UnitSuite) TestLogUploadAsDict(c *check.C) {
	coll := newCollection(arvadostest.FooCollection)
	user := &arvados.User{
		UUID:     arvadostest.ActiveUserUUID,
		FullName: "Active User",
	}
	for _, method := range []string{"POST", "PUT"} {
		filePath := "/" + method + "File"
		reqPath := "/c=" + arvadostest.FooCollection + filePath
		req := newRequest(method, "http://keep-web.example"+reqPath)
		logEvent := newFileEventLog(s.handler, req, filePath, coll, user, "")
		if !c.Check(logEvent, check.NotNil) {
			continue
		}
		c.Check(logEvent.asDict(), check.DeepEquals, arvadosclient.Dict{
			"event_type":  "file_upload",
			"object_uuid": arvadostest.ActiveUserUUID,
			"properties": arvadosclient.Dict{
				"reqPath":              reqPath,
				"collection_uuid":      arvadostest.FooCollection,
				"collection_file_path": filePath,
			},
		})
	}
}

func (s *UnitSuite) TestLogGETUUIDAsFields(c *check.C) {
	filePath := "/foo"
	reqPath := "/c=" + arvadostest.FooCollection + filePath
	req := newRequest("GET", "http://keep-web.example"+reqPath)
	coll := newCollection(arvadostest.FooCollection)
	logEvent := newFileEventLog(s.handler, req, filePath, coll, nil, "")
	c.Assert(logEvent, check.NotNil)
	c.Check(logEvent.asFields(), check.DeepEquals, logrus.Fields{
		"user_uuid":            s.handler.Cluster.ClusterID + "-tpzed-anonymouspublic",
		"collection_uuid":      arvadostest.FooCollection,
		"collection_file_path": filePath,
		"portable_data_hash":   arvadostest.FooCollectionPDH,
	})
}

func (s *UnitSuite) TestLogGETPDHAsFields(c *check.C) {
	filePath := "/Foo"
	reqPath := "/c=" + arvadostest.FooCollectionPDH + filePath
	req := newRequest("GET", "http://keep-web.example"+reqPath)
	coll := newCollection(arvadostest.FooCollectionPDH)
	user := &arvados.User{
		UUID:     arvadostest.ActiveUserUUID,
		FullName: "Active User",
	}
	logEvent := newFileEventLog(s.handler, req, filePath, coll, user, "")
	c.Assert(logEvent, check.NotNil)
	c.Check(logEvent.asFields(), check.DeepEquals, logrus.Fields{
		"user_uuid":            arvadostest.ActiveUserUUID,
		"user_full_name":       "Active User",
		"collection_uuid":      "",
		"collection_file_path": filePath,
		"portable_data_hash":   arvadostest.FooCollectionPDH,
	})
}

func (s *UnitSuite) TestLogUploadAsFields(c *check.C) {
	coll := newCollection(arvadostest.FooCollection)
	user := &arvados.User{
		UUID:     arvadostest.ActiveUserUUID,
		FullName: "Active User",
	}
	for _, method := range []string{"POST", "PUT"} {
		filePath := "/" + method + "File"
		reqPath := "/c=" + arvadostest.FooCollection + filePath
		req := newRequest(method, "http://keep-web.example"+reqPath)
		logEvent := newFileEventLog(s.handler, req, filePath, coll, user, "")
		if !c.Check(logEvent, check.NotNil) {
			continue
		}
		c.Check(logEvent.asFields(), check.DeepEquals, logrus.Fields{
			"user_uuid":            arvadostest.ActiveUserUUID,
			"user_full_name":       "Active User",
			"collection_uuid":      arvadostest.FooCollection,
			"collection_file_path": filePath,
		})
	}
}

func (s *UnitSuite) TestCORSPreflight(c *check.C) {
	h := s.handler
	u := mustParseURL("http://keep-web.example/c=" + arvadostest.FooCollection + "/foo")
	req := &http.Request{
		Method:     "OPTIONS",
		Host:       u.Host,
		URL:        u,
		RequestURI: u.RequestURI(),
		Header: http.Header{
			"Origin":                        {"https://workbench.example"},
			"Access-Control-Request-Method": {"POST"},
		},
	}

	// Check preflight for an allowed request
	resp := httptest.NewRecorder()
	h.ServeHTTP(resp, req)
	c.Check(resp.Code, check.Equals, http.StatusOK)
	c.Check(resp.Body.String(), check.Equals, "")
	c.Check(resp.Header().Get("Access-Control-Allow-Origin"), check.Equals, "*")
	c.Check(resp.Header().Get("Access-Control-Allow-Methods"), check.Equals, "COPY, DELETE, GET, LOCK, MKCOL, MOVE, OPTIONS, POST, PROPFIND, PROPPATCH, PUT, RMCOL, UNLOCK")
	c.Check(resp.Header().Get("Access-Control-Allow-Headers"), check.Equals, "Authorization, Content-Type, Range, Depth, Destination, If, Lock-Token, Overwrite, Timeout, Cache-Control")

	// Check preflight for a disallowed request
	resp = httptest.NewRecorder()
	req.Header.Set("Access-Control-Request-Method", "MAKE-COFFEE")
	h.ServeHTTP(resp, req)
	c.Check(resp.Body.String(), check.Equals, "")
	c.Check(resp.Code, check.Equals, http.StatusMethodNotAllowed)
}

func (s *UnitSuite) TestWebdavPrefixAndSource(c *check.C) {
	for _, trial := range []struct {
		method   string
		path     string
		prefix   string
		source   string
		notFound bool
		seeOther bool
	}{
		{
			method: "PROPFIND",
			path:   "/",
		},
		{
			method: "PROPFIND",
			path:   "/dir1",
		},
		{
			method: "PROPFIND",
			path:   "/dir1/",
		},
		{
			method: "PROPFIND",
			path:   "/dir1/foo",
			prefix: "/dir1",
			source: "/dir1",
		},
		{
			method: "PROPFIND",
			path:   "/prefix/dir1/foo",
			prefix: "/prefix/",
			source: "",
		},
		{
			method: "PROPFIND",
			path:   "/prefix/dir1/foo",
			prefix: "/prefix",
			source: "",
		},
		{
			method: "PROPFIND",
			path:   "/prefix/dir1/foo",
			prefix: "/prefix/",
			source: "/",
		},
		{
			method: "PROPFIND",
			path:   "/prefix/foo",
			prefix: "/prefix/",
			source: "/dir1/",
		},
		{
			method: "GET",
			path:   "/prefix/foo",
			prefix: "/prefix/",
			source: "/dir1/",
		},
		{
			method: "PROPFIND",
			path:   "/prefix/",
			prefix: "/prefix",
			source: "/dir1",
		},
		{
			method: "PROPFIND",
			path:   "/prefix",
			prefix: "/prefix",
			source: "/dir1/",
		},
		{
			method:   "GET",
			path:     "/prefix",
			prefix:   "/prefix",
			source:   "/dir1",
			seeOther: true,
		},
		{
			method:   "PROPFIND",
			path:     "/dir1/foo",
			prefix:   "",
			source:   "/dir1",
			notFound: true,
		},
	} {
		c.Logf("trial %+v", trial)
		u := mustParseURL("http://" + arvadostest.FooBarDirCollection + ".keep-web.example" + trial.path)
		req := &http.Request{
			Method:     trial.method,
			Host:       u.Host,
			URL:        u,
			RequestURI: u.RequestURI(),
			Header: http.Header{
				"Authorization":   {"Bearer " + arvadostest.ActiveTokenV2},
				"X-Webdav-Prefix": {trial.prefix},
				"X-Webdav-Source": {trial.source},
			},
			Body: ioutil.NopCloser(bytes.NewReader(nil)),
		}

		resp := httptest.NewRecorder()
		s.handler.ServeHTTP(resp, req)
		if trial.notFound {
			c.Check(resp.Code, check.Equals, http.StatusNotFound)
		} else if trial.method == "PROPFIND" {
			c.Check(resp.Code, check.Equals, http.StatusMultiStatus)
			c.Check(resp.Body.String(), check.Matches, `(?ms).*>\n?$`)
		} else if trial.seeOther {
			c.Check(resp.Code, check.Equals, http.StatusSeeOther)
		} else {
			c.Check(resp.Code, check.Equals, http.StatusOK)
		}
	}
}

func (s *UnitSuite) TestEmptyResponse(c *check.C) {
	// Ensure we start with an empty cache
	defer os.Setenv("HOME", os.Getenv("HOME"))
	os.Setenv("HOME", c.MkDir())
	s.handler.Cluster.Collections.WebDAVLogDownloadInterval = arvados.Duration(0)

	for _, trial := range []struct {
		dataExists    bool
		sendIMSHeader bool
		expectStatus  int
		logRegexp     string
	}{
		// If we return no content due to a Keep read error,
		// we should emit a log message.
		{false, false, http.StatusOK, `(?ms).*only wrote 0 bytes.*`},

		// If we return no content because the client sent an
		// If-Modified-Since header, our response should be
		// 304.  We still expect a "File download" log since it
		// counts as a file access for auditing.
		{true, true, http.StatusNotModified, `(?ms).*msg="File download".*`},
	} {
		c.Logf("trial: %+v", trial)
		arvadostest.StartKeep(2, true)
		if trial.dataExists {
			arv, err := arvadosclient.MakeArvadosClient()
			c.Assert(err, check.IsNil)
			arv.ApiToken = arvadostest.ActiveToken
			kc, err := keepclient.MakeKeepClient(arv)
			c.Assert(err, check.IsNil)
			_, _, err = kc.PutB([]byte("foo"))
			c.Assert(err, check.IsNil)
		}

		u := mustParseURL("http://" + arvadostest.FooCollection + ".keep-web.example/foo")
		req := &http.Request{
			Method:     "GET",
			Host:       u.Host,
			URL:        u,
			RequestURI: u.RequestURI(),
			Header: http.Header{
				"Authorization": {"Bearer " + arvadostest.ActiveToken},
			},
		}
		if trial.sendIMSHeader {
			req.Header.Set("If-Modified-Since", strings.Replace(time.Now().UTC().Format(time.RFC1123), "UTC", "GMT", -1))
		}

		var logbuf bytes.Buffer
		logger := logrus.New()
		logger.Out = &logbuf
		req = req.WithContext(ctxlog.Context(context.Background(), logger))

		resp := httptest.NewRecorder()
		s.handler.ServeHTTP(resp, req)
		c.Check(resp.Code, check.Equals, trial.expectStatus)
		c.Check(resp.Body.String(), check.Equals, "")

		c.Log(logbuf.String())
		c.Check(logbuf.String(), check.Matches, trial.logRegexp)
	}
}

func (s *UnitSuite) TestInvalidUUID(c *check.C) {
	bogusID := strings.Replace(arvadostest.FooCollectionPDH, "+", "-", 1) + "-"
	token := arvadostest.ActiveToken
	for _, trial := range []string{
		"http://keep-web/c=" + bogusID + "/foo",
		"http://keep-web/c=" + bogusID + "/t=" + token + "/foo",
		"http://keep-web/collections/download/" + bogusID + "/" + token + "/foo",
		"http://keep-web/collections/" + bogusID + "/foo",
		"http://" + bogusID + ".keep-web/" + bogusID + "/foo",
		"http://" + bogusID + ".keep-web/t=" + token + "/" + bogusID + "/foo",
	} {
		c.Log(trial)
		u := mustParseURL(trial)
		req := &http.Request{
			Method:     "GET",
			Host:       u.Host,
			URL:        u,
			RequestURI: u.RequestURI(),
		}
		resp := httptest.NewRecorder()
		s.cluster.Users.AnonymousUserToken = arvadostest.AnonymousToken
		s.handler.ServeHTTP(resp, req)
		c.Check(resp.Code, check.Equals, http.StatusNotFound)
	}
}

func mustParseURL(s string) *url.URL {
	r, err := url.Parse(s)
	if err != nil {
		panic("parse URL: " + s)
	}
	return r
}

func (s *IntegrationSuite) TestVhost404(c *check.C) {
	for _, testURL := range []string{
		arvadostest.NonexistentCollection + ".example.com/theperthcountyconspiracy",
		arvadostest.NonexistentCollection + ".example.com/t=" + arvadostest.ActiveToken + "/theperthcountyconspiracy",
	} {
		resp := httptest.NewRecorder()
		u := mustParseURL(testURL)
		req := &http.Request{
			Method:     "GET",
			URL:        u,
			RequestURI: u.RequestURI(),
		}
		s.handler.ServeHTTP(resp, req)
		c.Check(resp.Code, check.Equals, http.StatusNotFound)
		c.Check(resp.Body.String(), check.Equals, notFoundMessage+"\n")
	}
}

// An authorizer modifies an HTTP request to make use of the given
// token -- by adding it to a header, cookie, query param, or whatever
// -- and returns the HTTP status code we should expect from keep-web if
// the token is invalid.
type authorizer func(*http.Request, string) int

// We still need to accept "OAuth2 ..." as equivalent to "Bearer ..."
// for compatibility with older clients, including SDKs before 3.0.
func (s *IntegrationSuite) TestVhostViaAuthzHeaderOAuth2(c *check.C) {
	s.doVhostRequests(c, authzViaAuthzHeaderOAuth2)
}
func authzViaAuthzHeaderOAuth2(r *http.Request, tok string) int {
	r.Header.Add("Authorization", "OAuth2 "+tok)
	return http.StatusUnauthorized
}

func (s *IntegrationSuite) TestVhostViaAuthzHeaderBearer(c *check.C) {
	s.doVhostRequests(c, authzViaAuthzHeaderBearer)
}
func authzViaAuthzHeaderBearer(r *http.Request, tok string) int {
	r.Header.Add("Authorization", "Bearer "+tok)
	return http.StatusUnauthorized
}

func (s *IntegrationSuite) TestVhostViaCookieValue(c *check.C) {
	s.doVhostRequests(c, authzViaCookieValue)
}
func authzViaCookieValue(r *http.Request, tok string) int {
	r.AddCookie(&http.Cookie{
		Name:  "arvados_api_token",
		Value: auth.EncodeTokenCookie([]byte(tok)),
	})
	return http.StatusUnauthorized
}

func (s *IntegrationSuite) TestVhostViaHTTPBasicAuth(c *check.C) {
	s.doVhostRequests(c, authzViaHTTPBasicAuth)
}
func authzViaHTTPBasicAuth(r *http.Request, tok string) int {
	r.AddCookie(&http.Cookie{
		Name:  "arvados_api_token",
		Value: auth.EncodeTokenCookie([]byte(tok)),
	})
	return http.StatusUnauthorized
}

func (s *IntegrationSuite) TestVhostViaHTTPBasicAuthWithExtraSpaceChars(c *check.C) {
	s.doVhostRequests(c, func(r *http.Request, tok string) int {
		r.AddCookie(&http.Cookie{
			Name:  "arvados_api_token",
			Value: auth.EncodeTokenCookie([]byte(" " + tok + "\n")),
		})
		return http.StatusUnauthorized
	})
}

func (s *IntegrationSuite) TestVhostViaPath(c *check.C) {
	s.doVhostRequests(c, authzViaPath)
}
func authzViaPath(r *http.Request, tok string) int {
	r.URL.Path = "/t=" + tok + r.URL.Path
	return http.StatusNotFound
}

func (s *IntegrationSuite) TestVhostViaQueryString(c *check.C) {
	s.doVhostRequests(c, authzViaQueryString)
}
func authzViaQueryString(r *http.Request, tok string) int {
	r.URL.RawQuery = "api_token=" + tok
	return http.StatusUnauthorized
}

func (s *IntegrationSuite) TestVhostViaPOST(c *check.C) {
	s.doVhostRequests(c, authzViaPOST)
}
func authzViaPOST(r *http.Request, tok string) int {
	r.Method = "POST"
	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	r.Body = ioutil.NopCloser(strings.NewReader(
		url.Values{"api_token": {tok}}.Encode()))
	return http.StatusUnauthorized
}

func (s *IntegrationSuite) TestVhostViaXHRPOST(c *check.C) {
	s.doVhostRequests(c, authzViaPOST)
}
func authzViaXHRPOST(r *http.Request, tok string) int {
	r.Method = "POST"
	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Add("Origin", "https://origin.example")
	r.Body = ioutil.NopCloser(strings.NewReader(
		url.Values{
			"api_token":   {tok},
			"disposition": {"attachment"},
		}.Encode()))
	return http.StatusUnauthorized
}

// Try some combinations of {url, token} using the given authorization
// mechanism, and verify the result is correct.
func (s *IntegrationSuite) doVhostRequests(c *check.C, authz authorizer) {
	for _, hostPath := range []string{
		arvadostest.FooCollection + ".example.com/foo",
		arvadostest.FooCollection + "--collections.example.com/foo",
		arvadostest.FooCollection + "--collections.example.com/_/foo",
		arvadostest.FooCollectionPDH + ".example.com/foo",
		strings.Replace(arvadostest.FooCollectionPDH, "+", "-", -1) + "--collections.example.com/foo",
		arvadostest.FooBarDirCollection + ".example.com/dir1/foo",
	} {
		c.Log("doRequests: ", hostPath)
		s.doVhostRequestsWithHostPath(c, authz, hostPath)
	}
}

func (s *IntegrationSuite) doVhostRequestsWithHostPath(c *check.C, authz authorizer, hostPath string) {
	for _, tok := range []string{
		arvadostest.ActiveToken,
		arvadostest.ActiveToken[:15],
		arvadostest.SpectatorToken,
		"bogus",
		"",
	} {
		u := mustParseURL("http://" + hostPath)
		req := &http.Request{
			Method:     "GET",
			Host:       u.Host,
			URL:        u,
			RequestURI: u.RequestURI(),
			Header:     http.Header{},
		}
		failCode := authz(req, tok)
		req, resp := s.doReq(req)
		code, body := resp.Code, resp.Body.String()

		// If the initial request had a (non-empty) token
		// showing in the query string, we should have been
		// redirected in order to hide it in a cookie.
		c.Check(req.URL.String(), check.Not(check.Matches), `.*api_token=.+`)

		if tok == arvadostest.ActiveToken {
			c.Check(code, check.Equals, http.StatusOK)
			c.Check(body, check.Equals, "foo")
		} else {
			c.Check(code >= 400, check.Equals, true)
			c.Check(code < 500, check.Equals, true)
			if tok == arvadostest.SpectatorToken {
				// Valid token never offers to retry
				// with different credentials.
				c.Check(code, check.Equals, http.StatusNotFound)
			} else {
				// Invalid token can ask to retry
				// depending on the authz method.
				c.Check(code, check.Equals, failCode)
			}
			if code == 404 {
				c.Check(body, check.Equals, notFoundMessage+"\n")
			} else {
				c.Check(body, check.Equals, unauthorizedMessage+"\n")
			}
		}
	}
}

func (s *IntegrationSuite) TestVhostPortMatch(c *check.C) {
	for _, host := range []string{"download.example.com", "DOWNLOAD.EXAMPLE.COM"} {
		for _, port := range []string{"80", "443", "8000"} {
			s.handler.Cluster.Services.WebDAVDownload.ExternalURL.Host = fmt.Sprintf("download.example.com:%v", port)
			u := mustParseURL(fmt.Sprintf("http://%v/by_id/%v/foo", host, arvadostest.FooCollection))
			req := &http.Request{
				Method:     "GET",
				Host:       u.Host,
				URL:        u,
				RequestURI: u.RequestURI(),
				Header:     http.Header{"Authorization": []string{"Bearer " + arvadostest.ActiveToken}},
			}
			req, resp := s.doReq(req)
			code, _ := resp.Code, resp.Body.String()

			if port == "8000" {
				c.Check(code, check.Equals, 401)
			} else {
				c.Check(code, check.Equals, 200)
			}
		}
	}
}

func (s *IntegrationSuite) do(method string, urlstring string, token string, hdr http.Header) (*http.Request, *httptest.ResponseRecorder) {
	u := mustParseURL(urlstring)
	if hdr == nil && token != "" {
		hdr = http.Header{"Authorization": {"Bearer " + token}}
	} else if hdr == nil {
		hdr = http.Header{}
	} else if token != "" {
		panic("must not pass both token and hdr")
	}
	return s.doReq(&http.Request{
		Method:     method,
		Host:       u.Host,
		URL:        u,
		RequestURI: u.RequestURI(),
		Header:     hdr,
	})
}

func (s *IntegrationSuite) doReq(req *http.Request) (*http.Request, *httptest.ResponseRecorder) {
	resp := httptest.NewRecorder()
	s.handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusSeeOther {
		return req, resp
	}
	cookies := (&http.Response{Header: resp.Header()}).Cookies()
	u, _ := req.URL.Parse(resp.Header().Get("Location"))
	req = &http.Request{
		Method:     "GET",
		Host:       u.Host,
		URL:        u,
		RequestURI: u.RequestURI(),
		Header:     http.Header{},
	}
	for _, c := range cookies {
		req.AddCookie(c)
	}
	return s.doReq(req)
}

func (s *IntegrationSuite) TestVhostRedirectQueryTokenToCookie(c *check.C) {
	s.testVhostRedirectTokenToCookie(c, "GET",
		arvadostest.FooCollection+".example.com/foo",
		"?api_token="+arvadostest.ActiveToken,
		nil,
		"",
		http.StatusOK,
		"foo",
	)
}

func (s *IntegrationSuite) TestSingleOriginSecretLink(c *check.C) {
	s.testVhostRedirectTokenToCookie(c, "GET",
		"example.com/c="+arvadostest.FooCollection+"/t="+arvadostest.ActiveToken+"/foo",
		"",
		nil,
		"",
		http.StatusOK,
		"foo",
	)
}

func (s *IntegrationSuite) TestCollectionSharingToken(c *check.C) {
	s.testVhostRedirectTokenToCookie(c, "GET",
		"example.com/c="+arvadostest.FooFileCollectionUUID+"/t="+arvadostest.FooFileCollectionSharingToken+"/foo",
		"",
		nil,
		"",
		http.StatusOK,
		"foo",
	)
	// Same valid sharing token, but requesting a different collection
	s.testVhostRedirectTokenToCookie(c, "GET",
		"example.com/c="+arvadostest.FooCollection+"/t="+arvadostest.FooFileCollectionSharingToken+"/foo",
		"",
		nil,
		"",
		http.StatusNotFound,
		regexp.QuoteMeta(notFoundMessage+"\n"),
	)
}

// Bad token in URL is 404 Not Found because it doesn't make sense to
// retry the same URL with different authorization.
func (s *IntegrationSuite) TestSingleOriginSecretLinkBadToken(c *check.C) {
	s.testVhostRedirectTokenToCookie(c, "GET",
		"example.com/c="+arvadostest.FooCollection+"/t=bogus/foo",
		"",
		nil,
		"",
		http.StatusNotFound,
		regexp.QuoteMeta(notFoundMessage+"\n"),
	)
}

// Bad token in a cookie (even if it got there via our own
// query-string-to-cookie redirect) is, in principle, retryable via
// wb2-login-and-redirect flow.
func (s *IntegrationSuite) TestVhostRedirectQueryTokenToBogusCookie(c *check.C) {
	// Inline
	resp := s.testVhostRedirectTokenToCookie(c, "GET",
		arvadostest.FooCollection+".example.com/foo",
		"?api_token=thisisabogustoken",
		http.Header{"Sec-Fetch-Mode": {"navigate"}},
		"",
		http.StatusSeeOther,
		"",
	)
	u, err := url.Parse(resp.Header().Get("Location"))
	c.Assert(err, check.IsNil)
	c.Logf("redirected to %s", u)
	c.Check(u.Host, check.Equals, s.handler.Cluster.Services.Workbench2.ExternalURL.Host)
	c.Check(u.Query().Get("redirectToPreview"), check.Equals, "/c="+arvadostest.FooCollection+"/foo")
	c.Check(u.Query().Get("redirectToDownload"), check.Equals, "")

	// Download/attachment indicated by ?disposition=attachment
	resp = s.testVhostRedirectTokenToCookie(c, "GET",
		arvadostest.FooCollection+".example.com/foo",
		"?api_token=thisisabogustoken&disposition=attachment",
		http.Header{"Sec-Fetch-Mode": {"navigate"}},
		"",
		http.StatusSeeOther,
		"",
	)
	u, err = url.Parse(resp.Header().Get("Location"))
	c.Assert(err, check.IsNil)
	c.Logf("redirected to %s", u)
	c.Check(u.Host, check.Equals, s.handler.Cluster.Services.Workbench2.ExternalURL.Host)
	c.Check(u.Query().Get("redirectToPreview"), check.Equals, "")
	c.Check(u.Query().Get("redirectToDownload"), check.Equals, "/c="+arvadostest.FooCollection+"/foo")

	// Download/attachment indicated by vhost
	resp = s.testVhostRedirectTokenToCookie(c, "GET",
		s.handler.Cluster.Services.WebDAVDownload.ExternalURL.Host+"/c="+arvadostest.FooCollection+"/foo",
		"?api_token=thisisabogustoken",
		http.Header{"Sec-Fetch-Mode": {"navigate"}},
		"",
		http.StatusSeeOther,
		"",
	)
	u, err = url.Parse(resp.Header().Get("Location"))
	c.Assert(err, check.IsNil)
	c.Logf("redirected to %s", u)
	c.Check(u.Host, check.Equals, s.handler.Cluster.Services.Workbench2.ExternalURL.Host)
	c.Check(u.Query().Get("redirectToPreview"), check.Equals, "")
	c.Check(u.Query().Get("redirectToDownload"), check.Equals, "/c="+arvadostest.FooCollection+"/foo")

	// Without "Sec-Fetch-Mode: navigate" header, just 401.
	s.testVhostRedirectTokenToCookie(c, "GET",
		s.handler.Cluster.Services.WebDAVDownload.ExternalURL.Host+"/c="+arvadostest.FooCollection+"/foo",
		"?api_token=thisisabogustoken",
		http.Header{"Sec-Fetch-Mode": {"cors"}},
		"",
		http.StatusUnauthorized,
		regexp.QuoteMeta(unauthorizedMessage+"\n"),
	)
	s.testVhostRedirectTokenToCookie(c, "GET",
		s.handler.Cluster.Services.WebDAVDownload.ExternalURL.Host+"/c="+arvadostest.FooCollection+"/foo",
		"?api_token=thisisabogustoken",
		nil,
		"",
		http.StatusUnauthorized,
		regexp.QuoteMeta(unauthorizedMessage+"\n"),
	)
}

func (s *IntegrationSuite) TestVhostRedirectWithNoCache(c *check.C) {
	resp := s.testVhostRedirectTokenToCookie(c, "GET",
		arvadostest.FooCollection+".example.com/foo",
		"?api_token=thisisabogustoken",
		http.Header{
			"Sec-Fetch-Mode": {"navigate"},
			"Cache-Control":  {"no-cache"},
		},
		"",
		http.StatusSeeOther,
		"",
	)
	u, err := url.Parse(resp.Header().Get("Location"))
	c.Assert(err, check.IsNil)
	c.Logf("redirected to %s", u)
	c.Check(u.Host, check.Equals, s.handler.Cluster.Services.Workbench2.ExternalURL.Host)
	c.Check(u.Query().Get("redirectToPreview"), check.Equals, "/c="+arvadostest.FooCollection+"/foo")
	c.Check(u.Query().Get("redirectToDownload"), check.Equals, "")
}

func (s *IntegrationSuite) TestNoTokenWorkbench2LoginFlow(c *check.C) {
	for _, trial := range []struct {
		anonToken    bool
		cacheControl string
	}{
		{},
		{cacheControl: "no-cache"},
		{anonToken: true},
		{anonToken: true, cacheControl: "no-cache"},
	} {
		c.Logf("trial: %+v", trial)

		if trial.anonToken {
			s.handler.Cluster.Users.AnonymousUserToken = arvadostest.AnonymousToken
		} else {
			s.handler.Cluster.Users.AnonymousUserToken = ""
		}
		req, err := http.NewRequest("GET", "http://"+arvadostest.FooCollection+".example.com/foo", nil)
		c.Assert(err, check.IsNil)
		req.Header.Set("Sec-Fetch-Mode", "navigate")
		if trial.cacheControl != "" {
			req.Header.Set("Cache-Control", trial.cacheControl)
		}
		resp := httptest.NewRecorder()
		s.handler.ServeHTTP(resp, req)
		c.Check(resp.Code, check.Equals, http.StatusSeeOther)
		u, err := url.Parse(resp.Header().Get("Location"))
		c.Assert(err, check.IsNil)
		c.Logf("redirected to %q", u)
		c.Check(u.Host, check.Equals, s.handler.Cluster.Services.Workbench2.ExternalURL.Host)
		c.Check(u.Query().Get("redirectToPreview"), check.Equals, "/c="+arvadostest.FooCollection+"/foo")
		c.Check(u.Query().Get("redirectToDownload"), check.Equals, "")
	}
}

func (s *IntegrationSuite) TestVhostRedirectQueryTokenSingleOriginError(c *check.C) {
	s.testVhostRedirectTokenToCookie(c, "GET",
		"example.com/c="+arvadostest.FooCollection+"/foo",
		"?api_token="+arvadostest.ActiveToken,
		nil,
		"",
		http.StatusBadRequest,
		regexp.QuoteMeta("cannot serve inline content at this URL (possible configuration error; see https://doc.arvados.org/install/install-keep-web.html#dns)\n"),
	)
}

// If client requests an attachment by putting ?disposition=attachment
// in the query string, and gets redirected, the redirect target
// should respond with an attachment.
func (s *IntegrationSuite) TestVhostRedirectQueryTokenRequestAttachment(c *check.C) {
	resp := s.testVhostRedirectTokenToCookie(c, "GET",
		arvadostest.FooCollection+".example.com/foo",
		"?disposition=attachment&api_token="+arvadostest.ActiveToken,
		nil,
		"",
		http.StatusOK,
		"foo",
	)
	c.Check(resp.Header().Get("Content-Disposition"), check.Matches, "attachment(;.*)?")
}

func (s *IntegrationSuite) TestVhostRedirectQueryTokenSiteFS(c *check.C) {
	s.handler.Cluster.Services.WebDAVDownload.ExternalURL.Host = "download.example.com"
	resp := s.testVhostRedirectTokenToCookie(c, "GET",
		"download.example.com/by_id/"+arvadostest.FooCollection+"/foo",
		"?api_token="+arvadostest.ActiveToken,
		nil,
		"",
		http.StatusOK,
		"foo",
	)
	c.Check(resp.Header().Get("Content-Disposition"), check.Matches, "attachment(;.*)?")
}

func (s *IntegrationSuite) TestPastCollectionVersionFileAccess(c *check.C) {
	s.handler.Cluster.Services.WebDAVDownload.ExternalURL.Host = "download.example.com"
	resp := s.testVhostRedirectTokenToCookie(c, "GET",
		"download.example.com/c="+arvadostest.WazVersion1Collection+"/waz",
		"?api_token="+arvadostest.ActiveToken,
		nil,
		"",
		http.StatusOK,
		"waz",
	)
	c.Check(resp.Header().Get("Content-Disposition"), check.Matches, "attachment(;.*)?")
	resp = s.testVhostRedirectTokenToCookie(c, "GET",
		"download.example.com/by_id/"+arvadostest.WazVersion1Collection+"/waz",
		"?api_token="+arvadostest.ActiveToken,
		nil,
		"",
		http.StatusOK,
		"waz",
	)
	c.Check(resp.Header().Get("Content-Disposition"), check.Matches, "attachment(;.*)?")
}

func (s *IntegrationSuite) TestVhostRedirectQueryTokenTrustAllContent(c *check.C) {
	s.handler.Cluster.Collections.TrustAllContent = true
	s.testVhostRedirectTokenToCookie(c, "GET",
		"example.com/c="+arvadostest.FooCollection+"/foo",
		"?api_token="+arvadostest.ActiveToken,
		nil,
		"",
		http.StatusOK,
		"foo",
	)
}

func (s *IntegrationSuite) TestVhostRedirectQueryTokenAttachmentOnlyHost(c *check.C) {
	s.handler.Cluster.Services.WebDAVDownload.ExternalURL.Host = "example.com:1234"

	s.testVhostRedirectTokenToCookie(c, "GET",
		"example.com/c="+arvadostest.FooCollection+"/foo",
		"?api_token="+arvadostest.ActiveToken,
		nil,
		"",
		http.StatusBadRequest,
		regexp.QuoteMeta("cannot serve inline content at this URL (possible configuration error; see https://doc.arvados.org/install/install-keep-web.html#dns)\n"),
	)

	resp := s.testVhostRedirectTokenToCookie(c, "GET",
		"example.com:1234/c="+arvadostest.FooCollection+"/foo",
		"?api_token="+arvadostest.ActiveToken,
		nil,
		"",
		http.StatusOK,
		"foo",
	)
	c.Check(resp.Header().Get("Content-Disposition"), check.Equals, "attachment")
}

func (s *IntegrationSuite) TestVhostRedirectMultipleTokens(c *check.C) {
	baseUrl := arvadostest.FooCollection + ".example.com/foo"
	query := url.Values{}

	// The intent of these tests is to check that requests are redirected
	// correctly in the presence of multiple API tokens. The exact response
	// codes and content are not closely considered: they're just how
	// keep-web responded when we made the smallest possible fix. Changing
	// those responses may be okay, but you should still test all these
	// different cases and the associated redirect logic.
	query["api_token"] = []string{arvadostest.ActiveToken, arvadostest.AnonymousToken}
	s.testVhostRedirectTokenToCookie(c, "GET", baseUrl, "?"+query.Encode(), nil, "", http.StatusOK, "foo")
	query["api_token"] = []string{arvadostest.ActiveToken, arvadostest.AnonymousToken, ""}
	s.testVhostRedirectTokenToCookie(c, "GET", baseUrl, "?"+query.Encode(), nil, "", http.StatusOK, "foo")
	query["api_token"] = []string{arvadostest.ActiveToken, "", arvadostest.AnonymousToken}
	s.testVhostRedirectTokenToCookie(c, "GET", baseUrl, "?"+query.Encode(), nil, "", http.StatusOK, "foo")
	query["api_token"] = []string{"", arvadostest.ActiveToken}
	s.testVhostRedirectTokenToCookie(c, "GET", baseUrl, "?"+query.Encode(), nil, "", http.StatusOK, "foo")

	expectContent := regexp.QuoteMeta(unauthorizedMessage + "\n")
	query["api_token"] = []string{arvadostest.AnonymousToken, "invalidtoo"}
	s.testVhostRedirectTokenToCookie(c, "GET", baseUrl, "?"+query.Encode(), nil, "", http.StatusUnauthorized, expectContent)
	query["api_token"] = []string{arvadostest.AnonymousToken, ""}
	s.testVhostRedirectTokenToCookie(c, "GET", baseUrl, "?"+query.Encode(), nil, "", http.StatusUnauthorized, expectContent)
	query["api_token"] = []string{"", arvadostest.AnonymousToken}
	s.testVhostRedirectTokenToCookie(c, "GET", baseUrl, "?"+query.Encode(), nil, "", http.StatusUnauthorized, expectContent)
}

func (s *IntegrationSuite) TestVhostRedirectPOSTFormTokenToCookie(c *check.C) {
	s.testVhostRedirectTokenToCookie(c, "POST",
		arvadostest.FooCollection+".example.com/foo",
		"",
		http.Header{"Content-Type": {"application/x-www-form-urlencoded"}},
		url.Values{"api_token": {arvadostest.ActiveToken}}.Encode(),
		http.StatusOK,
		"foo",
	)
}

func (s *IntegrationSuite) TestVhostRedirectPOSTFormTokenToCookie404(c *check.C) {
	s.testVhostRedirectTokenToCookie(c, "POST",
		arvadostest.FooCollection+".example.com/foo",
		"",
		http.Header{"Content-Type": {"application/x-www-form-urlencoded"}},
		url.Values{"api_token": {arvadostest.SpectatorToken}}.Encode(),
		http.StatusNotFound,
		regexp.QuoteMeta(notFoundMessage+"\n"),
	)
}

func (s *IntegrationSuite) TestAnonymousTokenOK(c *check.C) {
	s.handler.Cluster.Users.AnonymousUserToken = arvadostest.AnonymousToken
	s.testVhostRedirectTokenToCookie(c, "GET",
		"example.com/c="+arvadostest.HelloWorldCollection+"/Hello%20world.txt",
		"",
		nil,
		"",
		http.StatusOK,
		"Hello world\n",
	)
}

func (s *IntegrationSuite) TestAnonymousTokenError(c *check.C) {
	s.handler.Cluster.Users.AnonymousUserToken = "anonymousTokenConfiguredButInvalid"
	s.testVhostRedirectTokenToCookie(c, "GET",
		"example.com/c="+arvadostest.HelloWorldCollection+"/Hello%20world.txt",
		"",
		nil,
		"",
		http.StatusUnauthorized,
		"Authorization tokens are not accepted here: .*\n",
	)
}

func (s *IntegrationSuite) TestSpecialCharsInPath(c *check.C) {
	s.handler.Cluster.Services.WebDAVDownload.ExternalURL.Host = "download.example.com"

	client := arvados.NewClientFromEnv()
	client.AuthToken = arvadostest.ActiveToken
	fs, err := (&arvados.Collection{}).FileSystem(client, nil)
	c.Assert(err, check.IsNil)
	path := `https:\\"odd' path chars`
	f, err := fs.OpenFile(path, os.O_CREATE, 0777)
	c.Assert(err, check.IsNil)
	f.Close()
	mtxt, err := fs.MarshalManifest(".")
	c.Assert(err, check.IsNil)
	var coll arvados.Collection
	err = client.RequestAndDecode(&coll, "POST", "arvados/v1/collections", nil, map[string]interface{}{
		"collection": map[string]string{
			"manifest_text": mtxt,
		},
	})
	c.Assert(err, check.IsNil)

	u, _ := url.Parse("http://download.example.com/c=" + coll.UUID + "/")
	req := &http.Request{
		Method:     "GET",
		Host:       u.Host,
		URL:        u,
		RequestURI: u.RequestURI(),
		Header: http.Header{
			"Authorization": {"Bearer " + client.AuthToken},
		},
	}
	resp := httptest.NewRecorder()
	s.handler.ServeHTTP(resp, req)
	c.Check(resp.Code, check.Equals, http.StatusOK)
	doc, err := html.Parse(resp.Body)
	c.Assert(err, check.IsNil)
	pathHrefMap := getPathHrefMap(doc)
	c.Check(pathHrefMap, check.HasLen, 1) // the one leaf added to collection
	href, hasPath := pathHrefMap[path]
	c.Assert(hasPath, check.Equals, true) // the path is listed
	relUrl := mustParseURL(href)
	c.Check(relUrl.Path, check.Equals, "./"+path) // href can be decoded back to path
}

func (s *IntegrationSuite) TestForwardSlashSubstitution(c *check.C) {
	arv := arvados.NewClientFromEnv()
	s.handler.Cluster.Services.WebDAVDownload.ExternalURL.Host = "download.example.com"
	s.handler.Cluster.Collections.ForwardSlashNameSubstitution = "{SOLIDUS}"
	name := "foo/bar/baz"
	nameShown := strings.Replace(name, "/", "{SOLIDUS}", -1)

	client := arvados.NewClientFromEnv()
	client.AuthToken = arvadostest.ActiveToken
	fs, err := (&arvados.Collection{}).FileSystem(client, nil)
	c.Assert(err, check.IsNil)
	f, err := fs.OpenFile("filename", os.O_CREATE, 0777)
	c.Assert(err, check.IsNil)
	f.Close()
	mtxt, err := fs.MarshalManifest(".")
	c.Assert(err, check.IsNil)
	var coll arvados.Collection
	err = client.RequestAndDecode(&coll, "POST", "arvados/v1/collections", nil, map[string]interface{}{
		"collection": map[string]string{
			"manifest_text": mtxt,
			"name":          name,
			"owner_uuid":    arvadostest.AProjectUUID,
		},
	})
	c.Assert(err, check.IsNil)
	defer arv.RequestAndDecode(&coll, "DELETE", "arvados/v1/collections/"+coll.UUID, nil, nil)

	base := "http://download.example.com/by_id/" + coll.OwnerUUID + "/"
	for tryURL, expectedAnchorText := range map[string]string{
		base:                   nameShown + "/",
		base + nameShown + "/": "filename",
	} {
		u, _ := url.Parse(tryURL)
		req := &http.Request{
			Method:     "GET",
			Host:       u.Host,
			URL:        u,
			RequestURI: u.RequestURI(),
			Header: http.Header{
				"Authorization": {"Bearer " + client.AuthToken},
			},
		}
		resp := httptest.NewRecorder()
		s.handler.ServeHTTP(resp, req)
		c.Check(resp.Code, check.Equals, http.StatusOK)
		doc, err := html.Parse(resp.Body)
		c.Assert(err, check.IsNil) // valid HTML
		pathHrefMap := getPathHrefMap(doc)
		href, hasExpected := pathHrefMap[expectedAnchorText]
		c.Assert(hasExpected, check.Equals, true) // has expected anchor text
		c.Assert(href, check.Not(check.Equals), "")
		relUrl := mustParseURL(href)
		c.Check(relUrl.Path, check.Equals, "./"+expectedAnchorText) // decoded href maps back to the anchor text
	}
}

// XHRs can't follow redirect-with-cookie so they rely on method=POST
// and disposition=attachment (telling us it's acceptable to respond
// with content instead of a redirect) and an Origin header that gets
// added automatically by the browser (telling us it's desirable to do
// so).
func (s *IntegrationSuite) TestXHRNoRedirect(c *check.C) {
	u, _ := url.Parse("http://example.com/c=" + arvadostest.FooCollection + "/foo")
	req := &http.Request{
		Method:     "POST",
		Host:       u.Host,
		URL:        u,
		RequestURI: u.RequestURI(),
		Header: http.Header{
			"Origin":       {"https://origin.example"},
			"Content-Type": {"application/x-www-form-urlencoded"},
		},
		Body: ioutil.NopCloser(strings.NewReader(url.Values{
			"api_token":   {arvadostest.ActiveToken},
			"disposition": {"attachment"},
		}.Encode())),
	}
	resp := httptest.NewRecorder()
	s.handler.ServeHTTP(resp, req)
	c.Check(resp.Code, check.Equals, http.StatusOK)
	c.Check(resp.Body.String(), check.Equals, "foo")
	c.Check(resp.Header().Get("Access-Control-Allow-Origin"), check.Equals, "*")

	// GET + Origin header is representative of both AJAX GET
	// requests and inline images via <IMG crossorigin="anonymous"
	// src="...">.
	u.RawQuery = "api_token=" + url.QueryEscape(arvadostest.ActiveTokenV2)
	req = &http.Request{
		Method:     "GET",
		Host:       u.Host,
		URL:        u,
		RequestURI: u.RequestURI(),
		Header: http.Header{
			"Origin": {"https://origin.example"},
		},
	}
	resp = httptest.NewRecorder()
	s.handler.ServeHTTP(resp, req)
	c.Check(resp.Code, check.Equals, http.StatusOK)
	c.Check(resp.Body.String(), check.Equals, "foo")
	c.Check(resp.Header().Get("Access-Control-Allow-Origin"), check.Equals, "*")
}

func (s *IntegrationSuite) testVhostRedirectTokenToCookie(c *check.C, method, hostPath, queryString string, reqHeader http.Header, reqBody string, expectStatus int, matchRespBody string) *httptest.ResponseRecorder {
	if reqHeader == nil {
		reqHeader = http.Header{}
	}
	u, _ := url.Parse(`http://` + hostPath + queryString)
	c.Logf("requesting %s", u)
	req := &http.Request{
		Method:     method,
		Host:       u.Host,
		URL:        u,
		RequestURI: u.RequestURI(),
		Header:     reqHeader,
		Body:       ioutil.NopCloser(strings.NewReader(reqBody)),
	}

	resp := httptest.NewRecorder()
	defer func() {
		c.Check(resp.Code, check.Equals, expectStatus)
		c.Check(resp.Body.String(), check.Matches, matchRespBody)
	}()

	s.handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusSeeOther {
		attachment, _ := regexp.MatchString(`^attachment(;|$)`, resp.Header().Get("Content-Disposition"))
		// Since we're not redirecting, check that any api_token in the URL is
		// handled safely.
		// If there is no token in the URL, then we're good.
		// Otherwise, if the response code is an error, the body is expected to
		// be static content, and nothing that might maliciously introspect the
		// URL. It's considered safe and allowed.
		// Otherwise, if the response content has attachment disposition,
		// that's considered safe for all the reasons explained in the
		// safeAttachment comment in handler.go.
		c.Check(!u.Query().Has("api_token") || resp.Code >= 400 || attachment, check.Equals, true)
		return resp
	}

	loc, err := url.Parse(resp.Header().Get("Location"))
	c.Assert(err, check.IsNil)
	c.Check(loc.Scheme, check.Equals, u.Scheme)
	c.Check(loc.Host, check.Equals, u.Host)
	c.Check(loc.RawPath, check.Equals, u.RawPath)
	// If the response was a redirect, it should never include an API token.
	c.Check(loc.Query().Has("api_token"), check.Equals, false)
	c.Check(resp.Body.String(), check.Matches, `.*href="http://`+regexp.QuoteMeta(html.EscapeString(hostPath))+`(\?[^"]*)?".*`)
	cookies := (&http.Response{Header: resp.Header()}).Cookies()

	c.Logf("following redirect to %s", u)
	req = &http.Request{
		Method:     "GET",
		Host:       loc.Host,
		URL:        loc,
		RequestURI: loc.RequestURI(),
		Header:     reqHeader,
	}
	for _, c := range cookies {
		req.AddCookie(c)
	}

	resp = httptest.NewRecorder()
	s.handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusSeeOther {
		c.Check(resp.Header().Get("Location"), check.Equals, "")
	}
	return resp
}

func (s *IntegrationSuite) TestDirectoryListingWithAnonymousToken(c *check.C) {
	s.handler.Cluster.Users.AnonymousUserToken = arvadostest.AnonymousToken
	s.testDirectoryListing(c)
}

func (s *IntegrationSuite) TestDirectoryListingWithNoAnonymousToken(c *check.C) {
	s.handler.Cluster.Users.AnonymousUserToken = ""
	s.testDirectoryListing(c)
}

func (s *IntegrationSuite) testDirectoryListing(c *check.C) {
	// The "ownership cycle" test fixtures are reachable from the
	// "filter group without filters" group, causing webdav's
	// walkfs to recurse indefinitely. Avoid that by deleting one
	// of the bogus fixtures.
	arv := arvados.NewClientFromEnv()
	err := arv.RequestAndDecode(nil, "DELETE", "arvados/v1/groups/zzzzz-j7d0g-cx2al9cqkmsf1hs", nil, nil)
	if err != nil {
		c.Assert(err, check.FitsTypeOf, &arvados.TransactionError{})
		c.Check(err.(*arvados.TransactionError).StatusCode, check.Equals, 404)
	}

	s.handler.Cluster.Services.WebDAVDownload.ExternalURL.Host = "download.example.com"
	authHeader := http.Header{
		"Authorization": {"Bearer " + arvadostest.ActiveToken},
	}
	for _, trial := range []struct {
		uri      string
		header   http.Header
		expect   []string
		redirect string
		cutDirs  int
	}{
		{
			uri:     strings.Replace(arvadostest.FooAndBarFilesInDirPDH, "+", "-", -1) + ".example.com/",
			header:  authHeader,
			expect:  []string{"dir1/foo", "dir1/bar"},
			cutDirs: 0,
		},
		{
			uri:     strings.Replace(arvadostest.FooAndBarFilesInDirPDH, "+", "-", -1) + ".example.com/dir1/",
			header:  authHeader,
			expect:  []string{"foo", "bar"},
			cutDirs: 1,
		},
		{
			// URLs of this form ignore authHeader, and
			// FooAndBarFilesInDirUUID isn't public, so
			// this returns 401.
			uri:    "download.example.com/collections/" + arvadostest.FooAndBarFilesInDirUUID + "/",
			header: authHeader,
			expect: nil,
		},
		{
			uri:     "download.example.com/users/active/foo_file_in_dir/",
			header:  authHeader,
			expect:  []string{"dir1/"},
			cutDirs: 3,
		},
		{
			uri:     "download.example.com/users/active/foo_file_in_dir/dir1/",
			header:  authHeader,
			expect:  []string{"bar"},
			cutDirs: 4,
		},
		{
			uri:     "download.example.com/",
			header:  authHeader,
			expect:  []string{"users/"},
			cutDirs: 0,
		},
		{
			uri:      "download.example.com/users",
			header:   authHeader,
			redirect: "/users/",
			expect:   []string{"active/"},
			cutDirs:  1,
		},
		{
			uri:     "download.example.com/users/",
			header:  authHeader,
			expect:  []string{"active/"},
			cutDirs: 1,
		},
		{
			uri:      "download.example.com/users/active",
			header:   authHeader,
			redirect: "/users/active/",
			expect:   []string{"foo_file_in_dir/"},
			cutDirs:  2,
		},
		{
			uri:     "download.example.com/users/active/",
			header:  authHeader,
			expect:  []string{"foo_file_in_dir/"},
			cutDirs: 2,
		},
		{
			uri:     "collections.example.com/collections/download/" + arvadostest.FooAndBarFilesInDirUUID + "/" + arvadostest.ActiveToken + "/",
			header:  nil,
			expect:  []string{"dir1/foo", "dir1/bar"},
			cutDirs: 4,
		},
		{
			uri:     "collections.example.com/c=" + arvadostest.FooAndBarFilesInDirUUID + "/t=" + arvadostest.ActiveToken + "/",
			header:  nil,
			expect:  []string{"dir1/foo", "dir1/bar"},
			cutDirs: 2,
		},
		{
			uri:     "collections.example.com/c=" + arvadostest.FooAndBarFilesInDirUUID + "/t=" + arvadostest.ActiveToken,
			header:  nil,
			expect:  []string{"dir1/foo", "dir1/bar"},
			cutDirs: 2,
		},
		{
			uri:     "download.example.com/c=" + arvadostest.FooAndBarFilesInDirUUID,
			header:  authHeader,
			expect:  []string{"dir1/foo", "dir1/bar"},
			cutDirs: 1,
		},
		{
			uri:      "download.example.com/c=" + arvadostest.FooAndBarFilesInDirUUID + "/dir1",
			header:   authHeader,
			redirect: "/c=" + arvadostest.FooAndBarFilesInDirUUID + "/dir1/",
			expect:   []string{"foo", "bar"},
			cutDirs:  2,
		},
		{
			uri:     "download.example.com/c=" + arvadostest.FooAndBarFilesInDirUUID + "/_/dir1/",
			header:  authHeader,
			expect:  []string{"foo", "bar"},
			cutDirs: 3,
		},
		{
			uri:      arvadostest.FooAndBarFilesInDirUUID + ".example.com/dir1?api_token=" + arvadostest.ActiveToken,
			header:   authHeader,
			redirect: "/dir1/",
			expect:   []string{"foo", "bar"},
			cutDirs:  1,
		},
		{
			uri:    "collections.example.com/c=" + arvadostest.FooAndBarFilesInDirUUID + "/theperthcountyconspiracydoesnotexist/",
			header: authHeader,
			expect: nil,
		},
		{
			uri:     "download.example.com/c=" + arvadostest.WazVersion1Collection,
			header:  authHeader,
			expect:  []string{"waz"},
			cutDirs: 1,
		},
		{
			uri:     "download.example.com/by_id/" + arvadostest.WazVersion1Collection,
			header:  authHeader,
			expect:  []string{"waz"},
			cutDirs: 2,
		},
		{
			uri:     "download.example.com/users/active/This filter group/",
			header:  authHeader,
			expect:  []string{"A Subproject/"},
			cutDirs: 3,
		},
		{
			uri:     "download.example.com/users/active/This filter group/A Subproject",
			header:  authHeader,
			expect:  []string{"baz_file/"},
			cutDirs: 4,
		},
		{
			uri:     "download.example.com/by_id/" + arvadostest.AFilterGroupUUID,
			header:  authHeader,
			expect:  []string{"A Subproject/"},
			cutDirs: 2,
		},
		{
			uri:     "download.example.com/by_id/" + arvadostest.AFilterGroupUUID + "/A Subproject",
			header:  authHeader,
			expect:  []string{"baz_file/"},
			cutDirs: 3,
		},
	} {
		comment := check.Commentf("HTML: %q redir %q => %q", trial.uri, trial.redirect, trial.expect)
		resp := httptest.NewRecorder()
		u := mustParseURL("//" + trial.uri)
		req := &http.Request{
			Method:     "GET",
			Host:       u.Host,
			URL:        u,
			RequestURI: u.RequestURI(),
			Header:     copyHeader(trial.header),
		}
		s.handler.ServeHTTP(resp, req)
		var cookies []*http.Cookie
		for resp.Code == http.StatusSeeOther {
			u, _ := req.URL.Parse(resp.Header().Get("Location"))
			req = &http.Request{
				Method:     "GET",
				Host:       u.Host,
				URL:        u,
				RequestURI: u.RequestURI(),
				Header:     copyHeader(trial.header),
			}
			cookies = append(cookies, (&http.Response{Header: resp.Header()}).Cookies()...)
			for _, c := range cookies {
				req.AddCookie(c)
			}
			resp = httptest.NewRecorder()
			s.handler.ServeHTTP(resp, req)
		}
		if trial.redirect != "" {
			c.Check(req.URL.Path, check.Equals, trial.redirect, comment)
		}
		if trial.expect == nil {
			c.Check(resp.Code, check.Equals, http.StatusUnauthorized, comment)
		} else {
			c.Check(resp.Code, check.Equals, http.StatusOK, comment)
			listingPageDoc, err := html.Parse(resp.Body)
			c.Check(err, check.IsNil, comment) // valid HTML document
			pathHrefMap := getPathHrefMap(listingPageDoc)
			c.Assert(pathHrefMap, check.Not(check.HasLen), 0, comment)
			for _, e := range trial.expect {
				href, hasE := pathHrefMap[e]
				c.Check(hasE, check.Equals, true, comment) // expected path is listed
				relUrl := mustParseURL(href)
				c.Check(relUrl.Path, check.Equals, "./"+e, comment) // href can be decoded back to path
			}
			wgetCommand := getWgetExamplePre(listingPageDoc)
			wgetExpected := regexp.MustCompile(`^\$ wget .*--cut-dirs=(\d+) .*'(https?://[^']+)'$`)
			wgetMatchGroups := wgetExpected.FindStringSubmatch(wgetCommand)
			c.Assert(wgetMatchGroups, check.NotNil)                                     // wget command matches
			c.Check(wgetMatchGroups[1], check.Equals, fmt.Sprintf("%d", trial.cutDirs)) // correct level of cut dirs in wget command
			printedUrl := mustParseURL(wgetMatchGroups[2])
			c.Check(printedUrl.Host, check.Equals, req.URL.Host)
			c.Check(printedUrl.Path, check.Equals, req.URL.Path) // URL arg in wget command can be decoded to the right path
		}

		comment = check.Commentf("WebDAV: %q => %q", trial.uri, trial.expect)
		req = &http.Request{
			Method:     "OPTIONS",
			Host:       u.Host,
			URL:        u,
			RequestURI: u.RequestURI(),
			Header:     copyHeader(trial.header),
			Body:       ioutil.NopCloser(&bytes.Buffer{}),
		}
		resp = httptest.NewRecorder()
		s.handler.ServeHTTP(resp, req)
		if trial.expect == nil {
			c.Check(resp.Code, check.Equals, http.StatusUnauthorized, comment)
		} else {
			c.Check(resp.Code, check.Equals, http.StatusOK, comment)
		}

		req = &http.Request{
			Method:     "PROPFIND",
			Host:       u.Host,
			URL:        u,
			RequestURI: u.RequestURI(),
			Header:     copyHeader(trial.header),
			Body:       ioutil.NopCloser(&bytes.Buffer{}),
		}
		resp = httptest.NewRecorder()
		s.handler.ServeHTTP(resp, req)
		// This check avoids logging a big XML document in the
		// event webdav throws a 500 error after sending
		// headers for a 207.
		if !c.Check(strings.HasSuffix(resp.Body.String(), "Internal Server Error"), check.Equals, false) {
			continue
		}
		if trial.expect == nil {
			c.Check(resp.Code, check.Equals, http.StatusUnauthorized, comment)
		} else {
			c.Check(resp.Code, check.Equals, http.StatusMultiStatus, comment)
			for _, e := range trial.expect {
				if strings.HasSuffix(e, "/") {
					e = filepath.Join(u.Path, e) + "/"
				} else {
					e = filepath.Join(u.Path, e)
				}
				e = strings.Replace(e, " ", "%20", -1)
				c.Check(resp.Body.String(), check.Matches, `(?ms).*<D:href>`+e+`</D:href>.*`, comment)
			}
		}
	}
}

// Shallow-traverse the HTML document, gathering the nodes satisfying the
// predicate function in the output slice. If a node matches the predicate,
// none of its children will be visited.
func getNodes(document *html.Node, predicate func(*html.Node) bool) []*html.Node {
	var acc []*html.Node
	var traverse func(*html.Node, []*html.Node) []*html.Node
	traverse = func(root *html.Node, sofar []*html.Node) []*html.Node {
		if root == nil {
			return sofar
		}
		if predicate(root) {
			return append(sofar, root)
		}
		for cur := root.FirstChild; cur != nil; cur = cur.NextSibling {
			sofar = traverse(cur, sofar)
		}
		return sofar
	}
	return traverse(document, acc)
}

// Returns true if a node has the attribute targetAttr with the given value
func matchesAttributeValue(node *html.Node, targetAttr string, value string) bool {
	for _, attr := range node.Attr {
		if attr.Key == targetAttr && attr.Val == value {
			return true
		}
	}
	return false
}

// Concatenate the content of text-node children of node; only direct
// children are visited, and any non-text children are skipped.
func getNodeText(node *html.Node) string {
	var recv strings.Builder
	for c := node.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.TextNode {
			recv.WriteString(c.Data)
		}
	}
	return recv.String()
}

// Returns a map from the directory listing item string (a path) to the href
// value of its <a> tag (an encoded relative URL)
func getPathHrefMap(document *html.Node) map[string]string {
	isItemATag := func(node *html.Node) bool {
		return node.Type == html.ElementNode && node.Data == "a" && matchesAttributeValue(node, "class", "item")
	}
	aTags := getNodes(document, isItemATag)
	output := make(map[string]string)
	for _, elem := range aTags {
		textContent := getNodeText(elem)
		for _, attr := range elem.Attr {
			if attr.Key == "href" {
				output[textContent] = attr.Val
				break
			}
		}
	}
	return output
}

func getWgetExamplePre(document *html.Node) string {
	isWgetPre := func(node *html.Node) bool {
		return node.Type == html.ElementNode && matchesAttributeValue(node, "id", "wget-example")
	}
	elements := getNodes(document, isWgetPre)
	if len(elements) != 1 {
		return ""
	}
	return getNodeText(elements[0])
}

func (s *IntegrationSuite) TestDeleteLastFile(c *check.C) {
	arv := arvados.NewClientFromEnv()
	var newCollection arvados.Collection
	err := arv.RequestAndDecode(&newCollection, "POST", "arvados/v1/collections", nil, map[string]interface{}{
		"collection": map[string]string{
			"owner_uuid":    arvadostest.ActiveUserUUID,
			"manifest_text": ". acbd18db4cc2f85cedef654fccc4a4d8+3 0:3:foo.txt 0:3:bar.txt\n",
			"name":          "keep-web test collection",
		},
		"ensure_unique_name": true,
	})
	c.Assert(err, check.IsNil)
	defer arv.RequestAndDecode(&newCollection, "DELETE", "arvados/v1/collections/"+newCollection.UUID, nil, nil)

	var updated arvados.Collection
	for _, fnm := range []string{"foo.txt", "bar.txt"} {
		s.handler.Cluster.Services.WebDAVDownload.ExternalURL.Host = "example.com"
		u, _ := url.Parse("http://example.com/c=" + newCollection.UUID + "/" + fnm)
		req := &http.Request{
			Method:     "DELETE",
			Host:       u.Host,
			URL:        u,
			RequestURI: u.RequestURI(),
			Header: http.Header{
				"Authorization": {"Bearer " + arvadostest.ActiveToken},
			},
		}
		resp := httptest.NewRecorder()
		s.handler.ServeHTTP(resp, req)
		c.Check(resp.Code, check.Equals, http.StatusNoContent)

		updated = arvados.Collection{}
		err = arv.RequestAndDecode(&updated, "GET", "arvados/v1/collections/"+newCollection.UUID, nil, nil)
		c.Check(err, check.IsNil)
		c.Check(updated.ManifestText, check.Not(check.Matches), `(?ms).*\Q`+fnm+`\E.*`)
		c.Logf("updated manifest_text %q", updated.ManifestText)
	}
	c.Check(updated.ManifestText, check.Equals, "")
}

func (s *IntegrationSuite) TestFileContentType(c *check.C) {
	s.handler.Cluster.Services.WebDAVDownload.ExternalURL.Host = "download.example.com"

	client := arvados.NewClientFromEnv()
	client.AuthToken = arvadostest.ActiveToken
	arv, err := arvadosclient.New(client)
	c.Assert(err, check.Equals, nil)
	kc, err := keepclient.MakeKeepClient(arv)
	c.Assert(err, check.Equals, nil)

	fs, err := (&arvados.Collection{}).FileSystem(client, kc)
	c.Assert(err, check.IsNil)

	trials := []struct {
		filename    string
		content     string
		contentType string
	}{
		{"picture.txt", "BMX bikes are small this year\n", "text/plain; charset=utf-8"},
		{"picture.bmp", "BMX bikes are small this year\n", "image/(x-ms-)?bmp"},
		{"picture.jpg", "BMX bikes are small this year\n", "image/jpeg"},
		{"picture1", "BMX bikes are small this year\n", "image/bmp"},            // content sniff; "BM" is the magic signature for .bmp
		{"picture2", "Cars are small this year\n", "text/plain; charset=utf-8"}, // content sniff
	}
	for _, trial := range trials {
		f, err := fs.OpenFile(trial.filename, os.O_CREATE|os.O_WRONLY, 0777)
		c.Assert(err, check.IsNil)
		_, err = f.Write([]byte(trial.content))
		c.Assert(err, check.IsNil)
		c.Assert(f.Close(), check.IsNil)
	}
	mtxt, err := fs.MarshalManifest(".")
	c.Assert(err, check.IsNil)
	var coll arvados.Collection
	err = client.RequestAndDecode(&coll, "POST", "arvados/v1/collections", nil, map[string]interface{}{
		"collection": map[string]string{
			"manifest_text": mtxt,
		},
	})
	c.Assert(err, check.IsNil)

	for _, trial := range trials {
		u, _ := url.Parse("http://download.example.com/by_id/" + coll.UUID + "/" + trial.filename)
		req := &http.Request{
			Method:     "GET",
			Host:       u.Host,
			URL:        u,
			RequestURI: u.RequestURI(),
			Header: http.Header{
				"Authorization": {"Bearer " + client.AuthToken},
			},
		}
		resp := httptest.NewRecorder()
		s.handler.ServeHTTP(resp, req)
		c.Check(resp.Code, check.Equals, http.StatusOK)
		c.Check(resp.Header().Get("Content-Type"), check.Matches, trial.contentType)
		c.Check(resp.Body.String(), check.Equals, trial.content)
	}
}

func (s *IntegrationSuite) TestCacheSize(c *check.C) {
	req, err := http.NewRequest("GET", "http://"+arvadostest.FooCollection+".example.com/foo", nil)
	req.Header.Set("Authorization", "Bearer "+arvadostest.ActiveTokenV2)
	c.Assert(err, check.IsNil)
	resp := httptest.NewRecorder()
	s.handler.ServeHTTP(resp, req)
	c.Assert(resp.Code, check.Equals, http.StatusOK)
	c.Check(s.handler.Cache.sessions[arvadostest.ActiveTokenV2].client.DiskCacheSize.Percent(), check.Equals, int64(10))
}

// Writing to a collection shouldn't affect its entry in the
// PDH-to-manifest cache.
func (s *IntegrationSuite) TestCacheWriteCollectionSamePDH(c *check.C) {
	arv, err := arvadosclient.MakeArvadosClient()
	c.Assert(err, check.Equals, nil)
	arv.ApiToken = arvadostest.ActiveToken

	u := mustParseURL("http://x.example/testfile")
	req := &http.Request{
		Method:     "GET",
		Host:       u.Host,
		URL:        u,
		RequestURI: u.RequestURI(),
		Header:     http.Header{"Authorization": {"Bearer " + arv.ApiToken}},
	}

	checkWithID := func(id string, status int) {
		req.URL.Host = strings.Replace(id, "+", "-", -1) + ".example"
		req.Host = req.URL.Host
		resp := httptest.NewRecorder()
		s.handler.ServeHTTP(resp, req)
		c.Check(resp.Code, check.Equals, status)
	}

	var colls [2]arvados.Collection
	for i := range colls {
		err := arv.Create("collections",
			map[string]interface{}{
				"ensure_unique_name": true,
				"collection": map[string]interface{}{
					"name": "test collection",
				},
			}, &colls[i])
		c.Assert(err, check.Equals, nil)
	}

	// Populate cache with empty collection
	checkWithID(colls[0].PortableDataHash, http.StatusNotFound)

	// write a file to colls[0]
	reqPut := *req
	reqPut.Method = "PUT"
	reqPut.URL.Host = colls[0].UUID + ".example"
	reqPut.Host = req.URL.Host
	reqPut.Body = ioutil.NopCloser(bytes.NewBufferString("testdata"))
	resp := httptest.NewRecorder()
	s.handler.ServeHTTP(resp, &reqPut)
	c.Check(resp.Code, check.Equals, http.StatusCreated)

	// new file should not appear in colls[1]
	checkWithID(colls[1].PortableDataHash, http.StatusNotFound)
	checkWithID(colls[1].UUID, http.StatusNotFound)

	checkWithID(colls[0].UUID, http.StatusOK)
}

func copyHeader(h http.Header) http.Header {
	hc := http.Header{}
	for k, v := range h {
		hc[k] = append([]string(nil), v...)
	}
	return hc
}

func (s *IntegrationSuite) checkUploadDownloadRequest(c *check.C, req *http.Request,
	successCode int, direction string, perm bool, userUuid, collectionUuid, collectionPDH, filepath string) {

	client := arvados.NewClientFromEnv()
	client.AuthToken = arvadostest.AdminToken
	var logentries arvados.LogList
	limit1 := 1
	err := client.RequestAndDecode(&logentries, "GET", "arvados/v1/logs", nil,
		arvados.ResourceListParams{
			Limit: &limit1,
			Order: "created_at desc"})
	c.Check(err, check.IsNil)
	c.Check(logentries.Items, check.HasLen, 1)
	lastLogId := logentries.Items[0].ID
	c.Logf("lastLogId: %d", lastLogId)

	var logbuf bytes.Buffer
	logger := logrus.New()
	logger.Out = &logbuf
	resp := httptest.NewRecorder()
	req = req.WithContext(ctxlog.Context(context.Background(), logger))
	s.handler.ServeHTTP(resp, req)

	if perm {
		c.Check(resp.Result().StatusCode, check.Equals, successCode)
		c.Check(logbuf.String(), check.Matches, `(?ms).*msg="File `+direction+`".*`)
		c.Check(logbuf.String(), check.Not(check.Matches), `(?ms).*level=error.*`)

		deadline := time.Now().Add(time.Second)
		for {
			c.Assert(time.Now().After(deadline), check.Equals, false, check.Commentf("timed out waiting for log entry"))
			logentries = arvados.LogList{}
			err = client.RequestAndDecode(&logentries, "GET", "arvados/v1/logs", nil,
				arvados.ResourceListParams{
					Filters: []arvados.Filter{
						{Attr: "event_type", Operator: "=", Operand: "file_" + direction},
						{Attr: "object_uuid", Operator: "=", Operand: userUuid},
					},
					Limit: &limit1,
					Order: "created_at desc",
				})
			c.Assert(err, check.IsNil)
			if len(logentries.Items) > 0 &&
				logentries.Items[0].ID > lastLogId &&
				logentries.Items[0].ObjectUUID == userUuid &&
				logentries.Items[0].Properties["collection_uuid"] == collectionUuid &&
				(collectionPDH == "" || logentries.Items[0].Properties["portable_data_hash"] == collectionPDH) &&
				logentries.Items[0].Properties["collection_file_path"] == filepath {
				break
			}
			c.Logf("logentries.Items: %+v", logentries.Items)
			time.Sleep(50 * time.Millisecond)
		}
	} else {
		c.Check(resp.Result().StatusCode, check.Equals, http.StatusForbidden)
		c.Check(logbuf.String(), check.Equals, "")
	}
}

func (s *IntegrationSuite) TestDownloadLoggingPermission(c *check.C) {
	u := mustParseURL("http://" + arvadostest.FooCollection + ".keep-web.example/foo")

	s.handler.Cluster.Collections.TrustAllContent = true
	s.handler.Cluster.Collections.WebDAVLogDownloadInterval = arvados.Duration(0)

	for _, adminperm := range []bool{true, false} {
		for _, userperm := range []bool{true, false} {
			s.handler.Cluster.Collections.WebDAVPermission.Admin.Download = adminperm
			s.handler.Cluster.Collections.WebDAVPermission.User.Download = userperm

			// Test admin permission
			req := &http.Request{
				Method:     "GET",
				Host:       u.Host,
				URL:        u,
				RequestURI: u.RequestURI(),
				Header: http.Header{
					"Authorization": {"Bearer " + arvadostest.AdminToken},
				},
			}
			s.checkUploadDownloadRequest(c, req, http.StatusOK, "download", adminperm,
				arvadostest.AdminUserUUID, arvadostest.FooCollection, arvadostest.FooCollectionPDH, "foo")

			// Test user permission
			req = &http.Request{
				Method:     "GET",
				Host:       u.Host,
				URL:        u,
				RequestURI: u.RequestURI(),
				Header: http.Header{
					"Authorization": {"Bearer " + arvadostest.ActiveToken},
				},
			}
			s.checkUploadDownloadRequest(c, req, http.StatusOK, "download", userperm,
				arvadostest.ActiveUserUUID, arvadostest.FooCollection, arvadostest.FooCollectionPDH, "foo")
		}
	}

	s.handler.Cluster.Collections.WebDAVPermission.User.Download = true

	for _, tryurl := range []string{"http://" + arvadostest.MultilevelCollection1 + ".keep-web.example/dir1/subdir/file1",
		"http://keep-web/users/active/multilevel_collection_1/dir1/subdir/file1"} {

		u = mustParseURL(tryurl)
		req := &http.Request{
			Method:     "GET",
			Host:       u.Host,
			URL:        u,
			RequestURI: u.RequestURI(),
			Header: http.Header{
				"Authorization": {"Bearer " + arvadostest.ActiveToken},
			},
		}
		s.checkUploadDownloadRequest(c, req, http.StatusOK, "download", true,
			arvadostest.ActiveUserUUID, arvadostest.MultilevelCollection1, arvadostest.MultilevelCollection1PDH, "dir1/subdir/file1")
	}

	u = mustParseURL("http://" + strings.Replace(arvadostest.FooCollectionPDH, "+", "-", 1) + ".keep-web.example/foo")
	req := &http.Request{
		Method:     "GET",
		Host:       u.Host,
		URL:        u,
		RequestURI: u.RequestURI(),
		Header: http.Header{
			"Authorization": {"Bearer " + arvadostest.ActiveToken},
		},
	}
	s.checkUploadDownloadRequest(c, req, http.StatusOK, "download", true,
		arvadostest.ActiveUserUUID, "", arvadostest.FooCollectionPDH, "foo")
}

func (s *IntegrationSuite) TestUploadLoggingPermission(c *check.C) {
	for _, adminperm := range []bool{true, false} {
		for _, userperm := range []bool{true, false} {

			arv := arvados.NewClientFromEnv()
			arv.AuthToken = arvadostest.ActiveToken

			var coll arvados.Collection
			err := arv.RequestAndDecode(&coll,
				"POST",
				"/arvados/v1/collections",
				nil,
				map[string]interface{}{
					"ensure_unique_name": true,
					"collection": map[string]interface{}{
						"name": "test collection",
					},
				})
			c.Assert(err, check.Equals, nil)

			u := mustParseURL("http://" + coll.UUID + ".keep-web.example/bar")

			s.handler.Cluster.Collections.WebDAVPermission.Admin.Upload = adminperm
			s.handler.Cluster.Collections.WebDAVPermission.User.Upload = userperm

			// Test admin permission
			req := &http.Request{
				Method:     "PUT",
				Host:       u.Host,
				URL:        u,
				RequestURI: u.RequestURI(),
				Header: http.Header{
					"Authorization": {"Bearer " + arvadostest.AdminToken},
				},
				Body: io.NopCloser(bytes.NewReader([]byte("bar"))),
			}
			s.checkUploadDownloadRequest(c, req, http.StatusCreated, "upload", adminperm,
				arvadostest.AdminUserUUID, coll.UUID, "", "bar")

			// Test user permission
			req = &http.Request{
				Method:     "PUT",
				Host:       u.Host,
				URL:        u,
				RequestURI: u.RequestURI(),
				Header: http.Header{
					"Authorization": {"Bearer " + arvadostest.ActiveToken},
				},
				Body: io.NopCloser(bytes.NewReader([]byte("bar"))),
			}
			s.checkUploadDownloadRequest(c, req, http.StatusCreated, "upload", userperm,
				arvadostest.ActiveUserUUID, coll.UUID, "", "bar")
		}
	}
}

func (s *IntegrationSuite) serveAndLogRequests(c *check.C, reqs *map[*http.Request]int) *bytes.Buffer {
	logbuf, ctx := newLoggerAndContext()
	var wg sync.WaitGroup
	for req, expectStatus := range *reqs {
		req := req.WithContext(ctx)
		expectStatus := expectStatus
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp := httptest.NewRecorder()
			s.handler.ServeHTTP(resp, req)
			c.Check(resp.Result().StatusCode, check.Equals, expectStatus)
		}()
	}
	wg.Wait()
	return logbuf
}

func countLogMatches(c *check.C, logbuf *bytes.Buffer, pattern string, matchCount int) bool {
	search, err := regexp.Compile(pattern)
	if !c.Check(err, check.IsNil, check.Commentf("failed to compile regexp: %v", err)) {
		return false
	}
	matches := search.FindAll(logbuf.Bytes(), -1)
	return c.Check(matches, check.HasLen, matchCount,
		check.Commentf("%d matching log messages: %+v", len(matches), matches))
}

func (s *IntegrationSuite) TestLogThrottling(c *check.C) {
	s.handler.Cluster.Collections.WebDAVLogDownloadInterval = arvados.Duration(time.Hour)
	fooURL := "http://" + arvadostest.FooCollection + ".keep-web.example/foo"
	req := newRequest("GET", fooURL)
	req.Header.Set("Authorization", "Bearer "+arvadostest.ActiveToken)
	pattern := `\bmsg="File download".* collection_file_path=foo\b`

	// All these requests get byte zero and should be logged.
	reqs := make(map[*http.Request]int)
	reqs[req] = http.StatusOK
	for _, byterange := range []string{"0-2", "0-1", "0-", "-3"} {
		req := req.Clone(context.Background())
		req.Header.Set("Range", "bytes="+byterange)
		reqs[req] = http.StatusPartialContent
	}
	logbuf := s.serveAndLogRequests(c, &reqs)
	countLogMatches(c, logbuf, pattern, len(reqs))

	// None of these requests get byte zero so they should all be throttled
	// (now that we've made at least one request for byte zero).
	reqs = make(map[*http.Request]int)
	for _, byterange := range []string{"1-2", "1-", "2-", "-1", "-2"} {
		req := req.Clone(context.Background())
		req.Header.Set("Range", "bytes="+byterange)
		reqs[req] = http.StatusPartialContent
	}
	logbuf = s.serveAndLogRequests(c, &reqs)
	countLogMatches(c, logbuf, pattern, 0)
}

func (s *IntegrationSuite) TestLogThrottleInterval(c *check.C) {
	s.handler.Cluster.Collections.WebDAVLogDownloadInterval = arvados.Duration(time.Nanosecond)
	logbuf, ctx := newLoggerAndContext()
	req := newRequest("GET", "http://"+arvadostest.FooCollection+".keep-web.example/foo")
	req.Header.Set("Authorization", "Bearer "+arvadostest.ActiveToken)
	req = req.WithContext(ctx)

	re := regexp.MustCompile(`\bmsg="File download".* collection_file_path=foo\b`)
	for expected := 1; expected < 4; expected++ {
		time.Sleep(2 * time.Nanosecond)
		resp := httptest.NewRecorder()
		s.handler.ServeHTTP(resp, req)
		c.Assert(resp.Result().StatusCode, check.Equals, http.StatusOK)
		matches := re.FindAll(logbuf.Bytes(), -1)
		c.Assert(matches, check.HasLen, expected,
			check.Commentf("%d matching log messages: %+v", len(matches), matches))
	}
}

func (s *IntegrationSuite) TestLogThrottleDifferentTokens(c *check.C) {
	s.handler.Cluster.Collections.WebDAVLogDownloadInterval = arvados.Duration(time.Hour)
	req := newRequest("GET", "http://"+arvadostest.FooCollection+".keep-web.example/foo")
	reqs := make(map[*http.Request]int)
	for _, token := range []string{arvadostest.ActiveToken, arvadostest.AdminToken} {
		req := req.Clone(context.Background())
		req.Header.Set("Authorization", "Bearer "+token)
		reqs[req] = http.StatusOK
	}
	logbuf := s.serveAndLogRequests(c, &reqs)
	countLogMatches(c, logbuf, `\bmsg="File download".* collection_file_path=foo\b`, len(reqs))
}

func (s *IntegrationSuite) TestLogThrottleDifferentFiles(c *check.C) {
	s.handler.Cluster.Collections.WebDAVLogDownloadInterval = arvados.Duration(time.Hour)
	baseURL := "http://" + arvadostest.MultilevelCollection1 + ".keep-web.example/"
	reqs := make(map[*http.Request]int)
	for _, filename := range []string{"file1", "file2", "file3"} {
		req := newRequest("GET", baseURL+filename)
		req.Header.Set("Authorization", "Bearer "+arvadostest.ActiveToken)
		reqs[req] = http.StatusOK
	}
	logbuf := s.serveAndLogRequests(c, &reqs)
	countLogMatches(c, logbuf, `\bmsg="File download".* collection_uuid=`+arvadostest.MultilevelCollection1+`\b`, len(reqs))
}

func (s *IntegrationSuite) TestLogThrottleDifferentSources(c *check.C) {
	s.handler.Cluster.Collections.WebDAVLogDownloadInterval = arvados.Duration(time.Hour)
	req := newRequest("GET", "http://"+arvadostest.FooCollection+".keep-web.example/foo")
	req.Header.Set("Authorization", "Bearer "+arvadostest.ActiveToken)
	reqs := make(map[*http.Request]int)
	reqs[req] = http.StatusOK
	for _, xff := range []string{"10.22.33.44", "100::123"} {
		req := req.Clone(context.Background())
		req.Header.Set("X-Forwarded-For", xff)
		reqs[req] = http.StatusOK
	}
	logbuf := s.serveAndLogRequests(c, &reqs)
	countLogMatches(c, logbuf, `\bmsg="File download".* collection_file_path=foo\b`, len(reqs))
}

func (s *IntegrationSuite) TestConcurrentWrites(c *check.C) {
	s.handler.Cluster.Collections.WebDAVCache.TTL = arvados.Duration(time.Second * 2)
	client := arvados.NewClientFromEnv()
	client.AuthToken = arvadostest.ActiveTokenV2
	var handler http.Handler = s.handler
	// handler = httpserver.AddRequestIDs(httpserver.LogRequests(s.handler)) // ...to enable request logging in test output

	// Each file we upload will consist of some unique content
	// followed by 2 MiB of filler content.
	filler := "."
	for i := 0; i < 21; i++ {
		filler += filler
	}

	// Start small, and increase concurrency (2^2, 4^2, ...)
	// only until hitting failure. Avoids unnecessarily long
	// failure reports.
	for n := 2; n < 16 && !c.Failed(); n = n * 2 {
		c.Logf("%s: n=%d", c.TestName(), n)

		var coll arvados.Collection
		err := client.RequestAndDecode(&coll, "POST", "arvados/v1/collections", nil, nil)
		c.Assert(err, check.IsNil)
		defer client.RequestAndDecode(&coll, "DELETE", "arvados/v1/collections/"+coll.UUID, nil, nil)

		var wg sync.WaitGroup
		for i := 0; i < n && !c.Failed(); i++ {
			i := i
			wg.Add(1)
			go func() {
				defer wg.Done()
				u := mustParseURL(fmt.Sprintf("http://%s.collections.example.com/i=%d", coll.UUID, i))
				resp := httptest.NewRecorder()
				req, err := http.NewRequest("MKCOL", u.String(), nil)
				c.Assert(err, check.IsNil)
				req.Header.Set("Authorization", "Bearer "+client.AuthToken)
				handler.ServeHTTP(resp, req)
				c.Assert(resp.Code, check.Equals, http.StatusCreated)
				for j := 0; j < n && !c.Failed(); j++ {
					j := j
					wg.Add(1)
					go func() {
						defer wg.Done()
						content := fmt.Sprintf("i=%d/j=%d", i, j)
						u := mustParseURL("http://" + coll.UUID + ".collections.example.com/" + content)

						resp := httptest.NewRecorder()
						req, err := http.NewRequest("PUT", u.String(), strings.NewReader(content+filler))
						c.Assert(err, check.IsNil)
						req.Header.Set("Authorization", "Bearer "+client.AuthToken)
						handler.ServeHTTP(resp, req)
						c.Check(resp.Code, check.Equals, http.StatusCreated, check.Commentf("%s", content))

						time.Sleep(time.Second)
						resp = httptest.NewRecorder()
						req, err = http.NewRequest("GET", u.String(), nil)
						c.Assert(err, check.IsNil)
						req.Header.Set("Authorization", "Bearer "+client.AuthToken)
						handler.ServeHTTP(resp, req)
						c.Check(resp.Code, check.Equals, http.StatusOK, check.Commentf("%s", content))
						c.Check(strings.TrimSuffix(resp.Body.String(), filler), check.Equals, content)
					}()
				}
			}()
		}
		wg.Wait()
		for i := 0; i < n; i++ {
			u := mustParseURL(fmt.Sprintf("http://%s.collections.example.com/i=%d", coll.UUID, i))
			resp := httptest.NewRecorder()
			req, err := http.NewRequest("PROPFIND", u.String(), &bytes.Buffer{})
			c.Assert(err, check.IsNil)
			req.Header.Set("Authorization", "Bearer "+client.AuthToken)
			s.handler.ServeHTTP(resp, req)
			c.Assert(resp.Code, check.Equals, http.StatusMultiStatus)
		}
	}
}

func (s *IntegrationSuite) TestDepthHeader(c *check.C) {
	s.handler.Cluster.Collections.WebDAVCache.TTL = arvados.Duration(time.Second * 2)
	client := arvados.NewClientFromEnv()
	client.AuthToken = arvadostest.ActiveTokenV2

	var coll arvados.Collection
	err := client.RequestAndDecode(&coll, "POST", "arvados/v1/collections", nil, nil)
	c.Assert(err, check.IsNil)
	defer client.RequestAndDecode(&coll, "DELETE", "arvados/v1/collections/"+coll.UUID, nil, nil)
	base := "http://" + coll.UUID + ".collections.example.com/"

	for _, trial := range []struct {
		method      string
		path        string
		destination string
		depth       string
		expectCode  int // 0 means expect 2xx
	}{
		// setup...
		{method: "MKCOL", path: "dir"},
		{method: "PUT", path: "dir/file"},
		{method: "MKCOL", path: "dir/dir2"},
		// delete with no depth = OK
		{method: "DELETE", path: "dir/dir2", depth: ""},
		// delete with depth other than infinity = fail
		{method: "DELETE", path: "dir", depth: "0", expectCode: 400},
		{method: "DELETE", path: "dir", depth: "1", expectCode: 400},
		// delete with depth infinity = OK
		{method: "DELETE", path: "dir", depth: "infinity"},

		// setup...
		{method: "MKCOL", path: "dir"},
		{method: "PUT", path: "dir/file"},
		{method: "MKCOL", path: "dir/dir2"},
		// move with depth other than infinity = fail
		{method: "MOVE", path: "dir", destination: "moved", depth: "0", expectCode: 400},
		{method: "MOVE", path: "dir", destination: "moved", depth: "1", expectCode: 400},
		// move with depth infinity = OK
		{method: "MOVE", path: "dir", destination: "moved", depth: "infinity"},
		{method: "DELETE", path: "moved"},

		// setup...
		{method: "MKCOL", path: "dir"},
		{method: "PUT", path: "dir/file"},
		{method: "MKCOL", path: "dir/dir2"},
		// copy with depth 0 = create empty destination dir
		{method: "COPY", path: "dir/", destination: "copied-empty/", depth: "0"},
		{method: "DELETE", path: "copied-empty/file", expectCode: 404},
		{method: "DELETE", path: "copied-empty"},
		// copy with depth 0 = create empty destination dir
		// (destination dir has no trailing slash this time)
		{method: "COPY", path: "dir/", destination: "copied-empty-noslash", depth: "0"},
		{method: "DELETE", path: "copied-empty-noslash/file", expectCode: 404},
		{method: "DELETE", path: "copied-empty-noslash"},
		// copy with depth 0 = create empty destination dir
		// (source dir has no trailing slash this time)
		{method: "COPY", path: "dir", destination: "copied-empty-noslash", depth: "0"},
		{method: "DELETE", path: "copied-empty-noslash/file", expectCode: 404},
		{method: "DELETE", path: "copied-empty-noslash"},
		// copy with depth 1 = fail
		{method: "COPY", path: "dir", destination: "copied", depth: "1", expectCode: 400},
		// copy with depth infinity = copy entire subtree
		{method: "COPY", path: "dir/", destination: "copied", depth: "infinity"},
		{method: "DELETE", path: "copied/file"},
		{method: "DELETE", path: "copied"},
		// copy with depth infinity = copy entire subtree
		// (source dir has no trailing slash this time)
		{method: "COPY", path: "dir", destination: "copied", depth: "infinity"},
		{method: "DELETE", path: "copied/file"},
		{method: "DELETE", path: "copied"},
		// cleanup
		{method: "DELETE", path: "dir"},
	} {
		c.Logf("trial %+v", trial)
		resp := httptest.NewRecorder()
		req, err := http.NewRequest(trial.method, base+trial.path, strings.NewReader(""))
		c.Assert(err, check.IsNil)
		req.Header.Set("Authorization", "Bearer "+client.AuthToken)
		if trial.destination != "" {
			req.Header.Set("Destination", base+trial.destination)
		}
		if trial.depth != "" {
			req.Header.Set("Depth", trial.depth)
		}
		s.handler.ServeHTTP(resp, req)
		if trial.expectCode != 0 {
			c.Assert(resp.Code, check.Equals, trial.expectCode)
		} else {
			c.Assert(resp.Code >= 200, check.Equals, true, check.Commentf("got code %d", resp.Code))
			c.Assert(resp.Code < 300, check.Equals, true, check.Commentf("got code %d", resp.Code))
		}
		c.Logf("resp.Body: %q", resp.Body.String())
	}
}
