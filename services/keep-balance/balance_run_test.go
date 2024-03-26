// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepbalance

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"syscall"
	"time"

	"git.arvados.org/arvados.git/lib/config"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"github.com/jmoiron/sqlx"
	"github.com/prometheus/client_golang/prometheus"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&runSuite{})

type reqTracker struct {
	reqs []http.Request
	sync.Mutex
}

func (rt *reqTracker) Count() int {
	rt.Lock()
	defer rt.Unlock()
	return len(rt.reqs)
}

func (rt *reqTracker) Add(req *http.Request) int {
	rt.Lock()
	defer rt.Unlock()
	rt.reqs = append(rt.reqs, *req)
	return len(rt.reqs)
}

var stubServices = []arvados.KeepService{
	{
		UUID:           "zzzzz-bi6l4-000000000000000",
		ServiceHost:    "keep0.zzzzz.arvadosapi.com",
		ServicePort:    25107,
		ServiceSSLFlag: false,
		ServiceType:    "disk",
	},
	{
		UUID:           "zzzzz-bi6l4-000000000000001",
		ServiceHost:    "keep1.zzzzz.arvadosapi.com",
		ServicePort:    25107,
		ServiceSSLFlag: false,
		ServiceType:    "disk",
	},
	{
		UUID:           "zzzzz-bi6l4-000000000000002",
		ServiceHost:    "keep2.zzzzz.arvadosapi.com",
		ServicePort:    25107,
		ServiceSSLFlag: false,
		ServiceType:    "disk",
	},
	{
		UUID:           "zzzzz-bi6l4-000000000000003",
		ServiceHost:    "keep3.zzzzz.arvadosapi.com",
		ServicePort:    25107,
		ServiceSSLFlag: false,
		ServiceType:    "disk",
	},
	{
		UUID:           "zzzzz-bi6l4-h0a0xwut9qa6g3a",
		ServiceHost:    "keep.zzzzz.arvadosapi.com",
		ServicePort:    25333,
		ServiceSSLFlag: true,
		ServiceType:    "proxy",
	},
}

var stubMounts = map[string][]arvados.KeepMount{
	"keep0.zzzzz.arvadosapi.com:25107": {{
		UUID:           "zzzzz-ivpuk-000000000000000",
		DeviceID:       "keep0-vol0",
		StorageClasses: map[string]bool{"default": true},
		AllowWrite:     true,
		AllowTrash:     true,
	}},
	"keep1.zzzzz.arvadosapi.com:25107": {{
		UUID:           "zzzzz-ivpuk-100000000000000",
		DeviceID:       "keep1-vol0",
		StorageClasses: map[string]bool{"default": true},
		AllowWrite:     true,
		AllowTrash:     true,
	}},
	"keep2.zzzzz.arvadosapi.com:25107": {{
		UUID:           "zzzzz-ivpuk-200000000000000",
		DeviceID:       "keep2-vol0",
		StorageClasses: map[string]bool{"default": true},
		AllowWrite:     true,
		AllowTrash:     true,
	}},
	"keep3.zzzzz.arvadosapi.com:25107": {{
		UUID:           "zzzzz-ivpuk-300000000000000",
		DeviceID:       "keep3-vol0",
		StorageClasses: map[string]bool{"default": true},
		AllowWrite:     true,
		AllowTrash:     true,
	}},
}

// stubServer is an HTTP transport that intercepts and processes all
// requests using its own handlers.
type stubServer struct {
	mux      *http.ServeMux
	srv      *httptest.Server
	mutex    sync.Mutex
	Requests reqTracker
	logf     func(string, ...interface{})
}

// Start initializes the stub server and returns an *http.Client that
// uses the stub server to handle all requests.
//
// A stubServer that has been started should eventually be shut down
// with Close().
func (s *stubServer) Start() *http.Client {
	// Set up a config.Client that forwards all requests to s.mux
	// via s.srv. Test cases will attach handlers to s.mux to get
	// the desired responses.
	s.mux = http.NewServeMux()
	s.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.mutex.Lock()
		s.Requests.Add(r)
		s.mutex.Unlock()
		w.Header().Set("Content-Type", "application/json")
		s.mux.ServeHTTP(w, r)
	}))
	return &http.Client{Transport: s}
}

func (s *stubServer) RoundTrip(req *http.Request) (*http.Response, error) {
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)
	return &http.Response{
		StatusCode: w.Code,
		Status:     fmt.Sprintf("%d %s", w.Code, http.StatusText(w.Code)),
		Header:     w.HeaderMap,
		Body:       ioutil.NopCloser(w.Body)}, nil
}

// Close releases resources used by the server.
func (s *stubServer) Close() {
	s.srv.Close()
}

func (s *stubServer) serveStatic(path, data string) *reqTracker {
	rt := &reqTracker{}
	s.mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		rt.Add(r)
		if r.Body != nil {
			ioutil.ReadAll(r.Body)
			r.Body.Close()
		}
		io.WriteString(w, data)
	})
	return rt
}

func (s *stubServer) serveCurrentUserAdmin() *reqTracker {
	return s.serveStatic("/arvados/v1/users/current",
		`{"uuid":"zzzzz-tpzed-000000000000000","is_admin":true,"is_active":true}`)
}

func (s *stubServer) serveCurrentUserNotAdmin() *reqTracker {
	return s.serveStatic("/arvados/v1/users/current",
		`{"uuid":"zzzzz-tpzed-000000000000000","is_admin":false,"is_active":true}`)
}

func (s *stubServer) serveDiscoveryDoc() *reqTracker {
	return s.serveStatic("/discovery/v1/apis/arvados/v1/rest",
		`{"defaultCollectionReplication":2}`)
}

func (s *stubServer) serveZeroCollections() *reqTracker {
	return s.serveStatic("/arvados/v1/collections",
		`{"items":[],"items_available":0}`)
}

func (s *stubServer) serveFooBarFileCollections() *reqTracker {
	rt := &reqTracker{}
	s.mux.HandleFunc("/arvados/v1/collections", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		rt.Add(r)
		if strings.Contains(r.Form.Get("filters"), `modified_at`) {
			io.WriteString(w, `{"items_available":0,"items":[]}`)
		} else {
			io.WriteString(w, `{"items_available":3,"items":[
				{"uuid":"zzzzz-4zz18-aaaaaaaaaaaaaaa","portable_data_hash":"fa7aeb5140e2848d39b416daeef4ffc5+45","manifest_text":". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:bar\n","modified_at":"2014-02-03T17:22:54Z"},
				{"uuid":"zzzzz-4zz18-ehbhgtheo8909or","portable_data_hash":"fa7aeb5140e2848d39b416daeef4ffc5+45","manifest_text":". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:bar\n","modified_at":"2014-02-03T17:22:54Z"},
				{"uuid":"zzzzz-4zz18-znfnqtbbv4spc3w","portable_data_hash":"1f4b0bc7583c2a7f9102c395f4ffc5e3+45","manifest_text":". acbd18db4cc2f85cedef654fccc4a4d8+3 0:3:foo\n","modified_at":"2014-02-03T17:22:54Z"}]}`)
		}
	})
	return rt
}

func (s *stubServer) serveCollectionsButSkipOne() *reqTracker {
	rt := &reqTracker{}
	s.mux.HandleFunc("/arvados/v1/collections", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		rt.Add(r)
		if strings.Contains(r.Form.Get("filters"), `"modified_at","\u003c="`) {
			io.WriteString(w, `{"items_available":3,"items":[]}`)
		} else if strings.Contains(r.Form.Get("filters"), `"modified_at","\u003e`) {
			io.WriteString(w, `{"items_available":0,"items":[]}`)
		} else if strings.Contains(r.Form.Get("filters"), `"modified_at","="`) && strings.Contains(r.Form.Get("filters"), `"uuid","\u003e"`) {
			io.WriteString(w, `{"items_available":0,"items":[]}`)
		} else if strings.Contains(r.Form.Get("filters"), `"modified_at","=",null`) {
			io.WriteString(w, `{"items_available":0,"items":[]}`)
		} else {
			io.WriteString(w, `{"items_available":2,"items":[
				{"uuid":"zzzzz-4zz18-ehbhgtheo8909or","portable_data_hash":"fa7aeb5140e2848d39b416daeef4ffc5+45","manifest_text":". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:bar\n","modified_at":"2014-02-03T17:22:54Z"},
				{"uuid":"zzzzz-4zz18-znfnqtbbv4spc3w","portable_data_hash":"1f4b0bc7583c2a7f9102c395f4ffc5e3+45","manifest_text":". acbd18db4cc2f85cedef654fccc4a4d8+3 0:3:foo\n","modified_at":"2014-02-03T17:22:54Z"}]}`)
		}
	})
	return rt
}

func (s *stubServer) serveZeroKeepServices() *reqTracker {
	return s.serveJSON("/arvados/v1/keep_services", arvados.KeepServiceList{})
}

func (s *stubServer) serveKeepServices(svcs []arvados.KeepService) *reqTracker {
	return s.serveJSON("/arvados/v1/keep_services", arvados.KeepServiceList{
		ItemsAvailable: len(svcs),
		Items:          svcs,
	})
}

func (s *stubServer) serveJSON(path string, resp interface{}) *reqTracker {
	rt := &reqTracker{}
	s.mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		rt.Add(r)
		json.NewEncoder(w).Encode(resp)
	})
	return rt
}

func (s *stubServer) serveKeepstoreMounts() *reqTracker {
	rt := &reqTracker{}
	s.mux.HandleFunc("/mounts", func(w http.ResponseWriter, r *http.Request) {
		rt.Add(r)
		json.NewEncoder(w).Encode(stubMounts[r.Host])
	})
	return rt
}

func (s *stubServer) serveKeepstoreIndexFoo4Bar1() *reqTracker {
	fooLine := func(mt int) string { return fmt.Sprintf("acbd18db4cc2f85cedef654fccc4a4d8+3 %d\n", 12345678+mt) }
	barLine := "37b51d194a7513e45b56f6524f2d51f2+3 12345678\n"
	rt := &reqTracker{}
	s.mux.HandleFunc("/index/", func(w http.ResponseWriter, r *http.Request) {
		count := rt.Add(r)
		if r.Host == "keep0.zzzzz.arvadosapi.com:25107" && strings.HasPrefix(barLine, r.URL.Path[7:]) {
			io.WriteString(w, barLine)
		}
		if strings.HasPrefix(fooLine(count), r.URL.Path[7:]) {
			io.WriteString(w, fooLine(count))
		}
		io.WriteString(w, "\n")
	})
	for _, mounts := range stubMounts {
		for i, mnt := range mounts {
			i := i
			s.mux.HandleFunc(fmt.Sprintf("/mounts/%s/blocks", mnt.UUID), func(w http.ResponseWriter, r *http.Request) {
				count := rt.Add(r)
				r.ParseForm()
				if i == 0 && r.Host == "keep0.zzzzz.arvadosapi.com:25107" && strings.HasPrefix(barLine, r.Form.Get("prefix")) {
					io.WriteString(w, barLine)
				}
				if i == 0 && strings.HasPrefix(fooLine(count), r.Form.Get("prefix")) {
					io.WriteString(w, fooLine(count))
				}
				io.WriteString(w, "\n")
			})
		}
	}
	return rt
}

func (s *stubServer) serveKeepstoreIndexFoo1() *reqTracker {
	fooLine := "acbd18db4cc2f85cedef654fccc4a4d8+3 12345678\n"
	rt := &reqTracker{}
	s.mux.HandleFunc("/index/", func(w http.ResponseWriter, r *http.Request) {
		rt.Add(r)
		if r.Host == "keep0.zzzzz.arvadosapi.com:25107" && strings.HasPrefix(fooLine, r.URL.Path[7:]) {
			io.WriteString(w, fooLine)
		}
		io.WriteString(w, "\n")
	})
	for _, mounts := range stubMounts {
		for i, mnt := range mounts {
			i := i
			s.mux.HandleFunc(fmt.Sprintf("/mounts/%s/blocks", mnt.UUID), func(w http.ResponseWriter, r *http.Request) {
				rt.Add(r)
				if i == 0 && strings.HasPrefix(fooLine, r.Form.Get("prefix")) {
					io.WriteString(w, fooLine)
				}
				io.WriteString(w, "\n")
			})
		}
	}
	return rt
}

func (s *stubServer) serveKeepstoreIndexIgnoringPrefix() *reqTracker {
	fooLine := "acbd18db4cc2f85cedef654fccc4a4d8+3 12345678\n"
	rt := &reqTracker{}
	s.mux.HandleFunc("/index/", func(w http.ResponseWriter, r *http.Request) {
		rt.Add(r)
		io.WriteString(w, fooLine)
		io.WriteString(w, "\n")
	})
	for _, mounts := range stubMounts {
		for _, mnt := range mounts {
			s.mux.HandleFunc(fmt.Sprintf("/mounts/%s/blocks", mnt.UUID), func(w http.ResponseWriter, r *http.Request) {
				rt.Add(r)
				io.WriteString(w, fooLine)
				io.WriteString(w, "\n")
			})
		}
	}
	return rt
}

func (s *stubServer) serveKeepstoreTrash() *reqTracker {
	return s.serveStatic("/trash", `{}`)
}

func (s *stubServer) serveKeepstorePull() *reqTracker {
	return s.serveStatic("/pull", `{}`)
}

type runSuite struct {
	stub   stubServer
	config *arvados.Cluster
	db     *sqlx.DB
	client *arvados.Client
}

func (s *runSuite) newServer(options *RunOptions) *Server {
	srv := &Server{
		Cluster:    s.config,
		ArvClient:  s.client,
		RunOptions: *options,
		Metrics:    newMetrics(prometheus.NewRegistry()),
		Logger:     options.Logger,
		Dumper:     options.Dumper,
		DB:         s.db,
	}
	return srv
}

func (s *runSuite) SetUpTest(c *check.C) {
	cfg, err := config.NewLoader(nil, ctxlog.TestLogger(c)).Load()
	c.Assert(err, check.Equals, nil)
	s.config, err = cfg.GetCluster("")
	c.Assert(err, check.Equals, nil)
	s.db, err = sqlx.Open("postgres", s.config.PostgreSQL.Connection.String())
	c.Assert(err, check.IsNil)

	s.config.Collections.BalancePeriod = arvados.Duration(time.Second)
	arvadostest.SetServiceURL(&s.config.Services.Keepbalance, "http://localhost:/")

	s.client = &arvados.Client{
		AuthToken: "xyzzy",
		APIHost:   "zzzzz.arvadosapi.com",
		Client:    s.stub.Start()}

	s.stub.serveDiscoveryDoc()
	s.stub.logf = c.Logf
}

func (s *runSuite) TearDownTest(c *check.C) {
	s.stub.Close()
}

func (s *runSuite) TestRefuseZeroCollections(c *check.C) {
	defer arvados.NewClientFromEnv().RequestAndDecode(nil, "POST", "database/reset", nil, nil)
	_, err := s.db.Exec(`delete from collections`)
	c.Assert(err, check.IsNil)
	opts := RunOptions{
		Logger: ctxlog.TestLogger(c),
	}
	s.stub.serveCurrentUserAdmin()
	s.stub.serveZeroCollections()
	s.stub.serveKeepServices(stubServices)
	s.stub.serveKeepstoreMounts()
	s.stub.serveKeepstoreIndexFoo4Bar1()
	trashReqs := s.stub.serveKeepstoreTrash()
	pullReqs := s.stub.serveKeepstorePull()
	srv := s.newServer(&opts)
	_, err = srv.runOnce(context.Background())
	c.Check(err, check.ErrorMatches, "received zero collections")
	c.Check(trashReqs.Count(), check.Equals, 4)
	c.Check(pullReqs.Count(), check.Equals, 0)
}

func (s *runSuite) TestRefuseBadIndex(c *check.C) {
	opts := RunOptions{
		ChunkPrefix: "abc",
		Logger:      ctxlog.TestLogger(c),
	}
	s.stub.serveCurrentUserAdmin()
	s.stub.serveFooBarFileCollections()
	s.stub.serveKeepServices(stubServices)
	s.stub.serveKeepstoreMounts()
	s.stub.serveKeepstoreIndexIgnoringPrefix()
	trashReqs := s.stub.serveKeepstoreTrash()
	pullReqs := s.stub.serveKeepstorePull()
	srv := s.newServer(&opts)
	bal, err := srv.runOnce(context.Background())
	c.Check(err, check.ErrorMatches, ".*Index response included block .* despite asking for prefix \"abc\"")
	c.Check(trashReqs.Count(), check.Equals, 4)
	c.Check(pullReqs.Count(), check.Equals, 0)
	c.Check(bal.stats.trashes, check.Equals, 0)
	c.Check(bal.stats.pulls, check.Equals, 0)
}

func (s *runSuite) TestRefuseNonAdmin(c *check.C) {
	opts := RunOptions{
		Logger: ctxlog.TestLogger(c),
	}
	s.stub.serveCurrentUserNotAdmin()
	s.stub.serveZeroCollections()
	s.stub.serveKeepServices(stubServices)
	s.stub.serveKeepstoreMounts()
	trashReqs := s.stub.serveKeepstoreTrash()
	pullReqs := s.stub.serveKeepstorePull()
	srv := s.newServer(&opts)
	_, err := srv.runOnce(context.Background())
	c.Check(err, check.ErrorMatches, "current user .* is not .* admin user")
	c.Check(trashReqs.Count(), check.Equals, 0)
	c.Check(pullReqs.Count(), check.Equals, 0)
}

func (s *runSuite) TestInvalidChunkPrefix(c *check.C) {
	for _, trial := range []struct {
		prefix string
		errRe  string
	}{
		{"123ABC", "invalid char \"A\" in chunk prefix.*"},
		{"123xyz", "invalid char \"x\" in chunk prefix.*"},
		{"123456789012345678901234567890123", "invalid chunk prefix .* longer than a block hash"},
	} {
		s.SetUpTest(c)
		c.Logf("trying invalid prefix %q", trial.prefix)
		opts := RunOptions{
			ChunkPrefix: trial.prefix,
			Logger:      ctxlog.TestLogger(c),
		}
		s.stub.serveCurrentUserAdmin()
		s.stub.serveFooBarFileCollections()
		s.stub.serveKeepServices(stubServices)
		s.stub.serveKeepstoreMounts()
		trashReqs := s.stub.serveKeepstoreTrash()
		pullReqs := s.stub.serveKeepstorePull()
		srv := s.newServer(&opts)
		_, err := srv.runOnce(context.Background())
		c.Check(err, check.ErrorMatches, trial.errRe)
		c.Check(trashReqs.Count(), check.Equals, 0)
		c.Check(pullReqs.Count(), check.Equals, 0)
	}
}

func (s *runSuite) TestRefuseSameDeviceDifferentVolumes(c *check.C) {
	opts := RunOptions{
		Logger: ctxlog.TestLogger(c),
	}
	s.stub.serveCurrentUserAdmin()
	s.stub.serveZeroCollections()
	s.stub.serveKeepServices(stubServices)
	s.stub.mux.HandleFunc("/mounts", func(w http.ResponseWriter, r *http.Request) {
		hostid := r.Host[:5] // "keep0.zzzzz.arvadosapi.com:25107" => "keep0"
		json.NewEncoder(w).Encode([]arvados.KeepMount{{
			UUID:           "zzzzz-ivpuk-0000000000" + hostid,
			DeviceID:       "keep0-vol0",
			StorageClasses: map[string]bool{"default": true},
		}})
	})
	trashReqs := s.stub.serveKeepstoreTrash()
	pullReqs := s.stub.serveKeepstorePull()
	srv := s.newServer(&opts)
	_, err := srv.runOnce(context.Background())
	c.Check(err, check.ErrorMatches, "cannot continue with config errors.*")
	c.Check(trashReqs.Count(), check.Equals, 0)
	c.Check(pullReqs.Count(), check.Equals, 0)
}

func (s *runSuite) TestWriteLostBlocks(c *check.C) {
	lostf, err := ioutil.TempFile("", "keep-balance-lost-blocks-test-")
	c.Assert(err, check.IsNil)
	s.config.Collections.BlobMissingReport = lostf.Name()
	defer os.Remove(lostf.Name())
	opts := RunOptions{
		Logger: ctxlog.TestLogger(c),
	}
	s.stub.serveCurrentUserAdmin()
	s.stub.serveFooBarFileCollections()
	s.stub.serveKeepServices(stubServices)
	s.stub.serveKeepstoreMounts()
	s.stub.serveKeepstoreIndexFoo1()
	s.stub.serveKeepstoreTrash()
	s.stub.serveKeepstorePull()
	srv := s.newServer(&opts)
	c.Assert(err, check.IsNil)
	_, err = srv.runOnce(context.Background())
	c.Check(err, check.IsNil)
	lost, err := ioutil.ReadFile(lostf.Name())
	c.Assert(err, check.IsNil)
	c.Check(string(lost), check.Matches, `(?ms).*37b51d194a7513e45b56f6524f2d51f2.* fa7aeb5140e2848d39b416daeef4ffc5\+45.*`)
}

func (s *runSuite) TestDryRun(c *check.C) {
	s.config.Collections.BalanceTrashLimit = 0
	s.config.Collections.BalancePullLimit = 0
	opts := RunOptions{
		Logger: ctxlog.TestLogger(c),
	}
	s.stub.serveCurrentUserAdmin()
	collReqs := s.stub.serveFooBarFileCollections()
	s.stub.serveKeepServices(stubServices)
	s.stub.serveKeepstoreMounts()
	s.stub.serveKeepstoreIndexFoo4Bar1()
	trashReqs := s.stub.serveKeepstoreTrash()
	pullReqs := s.stub.serveKeepstorePull()
	srv := s.newServer(&opts)
	bal, err := srv.runOnce(context.Background())
	c.Check(err, check.IsNil)
	for _, req := range collReqs.reqs {
		c.Check(req.Form.Get("include_trash"), check.Equals, "true")
		c.Check(req.Form.Get("include_old_versions"), check.Equals, "true")
	}
	c.Check(trashReqs.Count(), check.Equals, 0)
	c.Check(pullReqs.Count(), check.Equals, 0)
	c.Check(bal.stats.pulls, check.Equals, 0)
	c.Check(bal.stats.pullsDeferred, check.Not(check.Equals), 0)
	c.Check(bal.stats.trashes, check.Equals, 0)
	c.Check(bal.stats.trashesDeferred, check.Not(check.Equals), 0)
	c.Check(bal.stats.underrep.replicas, check.Not(check.Equals), 0)
	c.Check(bal.stats.overrep.replicas, check.Not(check.Equals), 0)

	metrics := arvadostest.GatherMetricsAsString(srv.Metrics.reg)
	c.Check(metrics, check.Matches, `(?ms).*\narvados_keep_trash_entries_deferred_count [1-9].*`)
	c.Check(metrics, check.Matches, `(?ms).*\narvados_keep_pull_entries_deferred_count [1-9].*`)
}

func (s *runSuite) TestCommit(c *check.C) {
	s.config.Collections.BlobMissingReport = c.MkDir() + "/keep-balance-lost-blocks-test-"
	s.config.ManagementToken = "xyzzy"
	opts := RunOptions{
		Logger: ctxlog.TestLogger(c),
		Dumper: ctxlog.TestLogger(c),
	}
	s.stub.serveCurrentUserAdmin()
	s.stub.serveFooBarFileCollections()
	s.stub.serveKeepServices(stubServices)
	s.stub.serveKeepstoreMounts()
	s.stub.serveKeepstoreIndexFoo4Bar1()
	trashReqs := s.stub.serveKeepstoreTrash()
	pullReqs := s.stub.serveKeepstorePull()
	srv := s.newServer(&opts)
	bal, err := srv.runOnce(context.Background())
	c.Check(err, check.IsNil)
	c.Check(trashReqs.Count(), check.Equals, 8)
	c.Check(pullReqs.Count(), check.Equals, 4)
	// "foo" block is overreplicated by 2
	c.Check(bal.stats.trashes, check.Equals, 2)
	// "bar" block is underreplicated by 1, and its only copy is
	// in a poor rendezvous position
	c.Check(bal.stats.pulls, check.Equals, 2)

	lost, err := ioutil.ReadFile(s.config.Collections.BlobMissingReport)
	c.Assert(err, check.IsNil)
	c.Check(string(lost), check.Not(check.Matches), `(?ms).*acbd18db4cc2f85cedef654fccc4a4d8.*`)

	metrics := arvadostest.GatherMetricsAsString(srv.Metrics.reg)
	c.Check(metrics, check.Matches, `(?ms).*\narvados_keep_total_bytes 15\n.*`)
	c.Check(metrics, check.Matches, `(?ms).*\narvados_keepbalance_changeset_compute_seconds_sum [0-9\.]+\n.*`)
	c.Check(metrics, check.Matches, `(?ms).*\narvados_keepbalance_changeset_compute_seconds_count 1\n.*`)
	c.Check(metrics, check.Matches, `(?ms).*\narvados_keep_dedup_byte_ratio [1-9].*`)
	c.Check(metrics, check.Matches, `(?ms).*\narvados_keep_dedup_block_ratio [1-9].*`)

	for _, cat := range []string{
		"dedup_byte_ratio", "dedup_block_ratio", "collection_bytes",
		"referenced_bytes", "referenced_blocks", "reference_count",
		"pull_entries_sent_count",
		"trash_entries_sent_count",
	} {
		c.Check(metrics, check.Matches, `(?ms).*\narvados_keep_`+cat+` [1-9].*`)
	}

	for _, cat := range []string{
		"pull_entries_deferred_count",
		"trash_entries_deferred_count",
	} {
		c.Check(metrics, check.Matches, `(?ms).*\narvados_keep_`+cat+` 0\n.*`)
	}

	c.Check(metrics, check.Matches, `(?ms).*\narvados_keep_replicated_block_count{replicas="0"} [1-9].*`)
	c.Check(metrics, check.Matches, `(?ms).*\narvados_keep_replicated_block_count{replicas="1"} [1-9].*`)
	c.Check(metrics, check.Matches, `(?ms).*\narvados_keep_replicated_block_count{replicas="9"} 0\n.*`)

	for _, sub := range []string{"replicas", "blocks", "bytes"} {
		for _, cat := range []string{"needed", "unneeded", "unachievable", "pulling"} {
			c.Check(metrics, check.Matches, `(?ms).*\narvados_keep_usage_`+sub+`{status="`+cat+`",storage_class="default"} [1-9].*`)
		}
		for _, cat := range []string{"total", "garbage", "transient", "overreplicated", "underreplicated", "unachievable", "balanced", "desired", "lost"} {
			c.Check(metrics, check.Matches, `(?ms).*\narvados_keep_`+cat+`_`+sub+` [0-9].*`)
		}
	}
	c.Logf("%s", metrics)
}

func (s *runSuite) TestChunkPrefix(c *check.C) {
	s.config.Collections.BlobMissingReport = c.MkDir() + "/keep-balance-lost-blocks-test-"
	opts := RunOptions{
		ChunkPrefix: "ac", // catch "foo" but not "bar"
		Logger:      ctxlog.TestLogger(c),
		Dumper:      ctxlog.TestLogger(c),
	}
	s.stub.serveCurrentUserAdmin()
	s.stub.serveFooBarFileCollections()
	s.stub.serveKeepServices(stubServices)
	s.stub.serveKeepstoreMounts()
	s.stub.serveKeepstoreIndexFoo4Bar1()
	trashReqs := s.stub.serveKeepstoreTrash()
	pullReqs := s.stub.serveKeepstorePull()
	srv := s.newServer(&opts)
	bal, err := srv.runOnce(context.Background())
	c.Check(err, check.IsNil)
	c.Check(trashReqs.Count(), check.Equals, 8)
	c.Check(pullReqs.Count(), check.Equals, 4)
	// "foo" block is overreplicated by 2
	c.Check(bal.stats.trashes, check.Equals, 2)
	// "bar" block is underreplicated but does not match prefix
	c.Check(bal.stats.pulls, check.Equals, 0)

	lost, err := ioutil.ReadFile(s.config.Collections.BlobMissingReport)
	c.Assert(err, check.IsNil)
	c.Check(string(lost), check.Equals, "")
}

func (s *runSuite) TestRunForever_TriggeredByTimer(c *check.C) {
	s.config.ManagementToken = "xyzzy"
	opts := RunOptions{
		Logger: ctxlog.TestLogger(c),
		Dumper: ctxlog.TestLogger(c),
	}
	s.stub.serveCurrentUserAdmin()
	s.stub.serveFooBarFileCollections()
	s.stub.serveKeepServices(stubServices)
	s.stub.serveKeepstoreMounts()
	s.stub.serveKeepstoreIndexFoo4Bar1()
	trashReqs := s.stub.serveKeepstoreTrash()
	pullReqs := s.stub.serveKeepstorePull()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.config.Collections.BalancePeriod = arvados.Duration(10 * time.Millisecond)
	srv := s.newServer(&opts)

	done := make(chan bool)
	go func() {
		srv.runForever(ctx)
		close(done)
	}()

	// Each run should send 4 pull lists + 4 trash lists. The
	// first run should also send 4 empty trash lists at
	// startup. We should complete at least four runs in much less
	// than 10s.
	for t0 := time.Now(); time.Since(t0) < 10*time.Second; {
		pulls := pullReqs.Count()
		if pulls >= 16 && trashReqs.Count() == pulls+4 {
			break
		}
		time.Sleep(time.Millisecond)
	}
	cancel()
	<-done
	c.Check(pullReqs.Count() >= 16, check.Equals, true)
	c.Check(trashReqs.Count() >= 20, check.Equals, true)

	// We should have completed 4 runs before calling cancel().
	// But the next run might also have started before we called
	// cancel(), in which case the extra run will be included in
	// the changeset_compute_seconds_count metric.
	completed := pullReqs.Count() / 4
	metrics := arvadostest.GatherMetricsAsString(srv.Metrics.reg)
	c.Check(metrics, check.Matches, fmt.Sprintf(`(?ms).*\narvados_keepbalance_changeset_compute_seconds_count (%d|%d)\n.*`, completed, completed+1))
}

func (s *runSuite) TestRunForever_TriggeredBySignal(c *check.C) {
	s.config.ManagementToken = "xyzzy"
	opts := RunOptions{
		Logger: ctxlog.TestLogger(c),
		Dumper: ctxlog.TestLogger(c),
	}
	s.stub.serveCurrentUserAdmin()
	s.stub.serveFooBarFileCollections()
	s.stub.serveKeepServices(stubServices)
	s.stub.serveKeepstoreMounts()
	s.stub.serveKeepstoreIndexFoo4Bar1()
	trashReqs := s.stub.serveKeepstoreTrash()
	pullReqs := s.stub.serveKeepstorePull()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.config.Collections.BalancePeriod = arvados.Duration(time.Minute)
	srv := s.newServer(&opts)

	done := make(chan bool)
	go func() {
		srv.runForever(ctx)
		close(done)
	}()

	procself, err := os.FindProcess(os.Getpid())
	c.Assert(err, check.IsNil)

	// Each run should send 4 pull lists + 4 trash lists. The
	// first run should also send 4 empty trash lists at
	// startup. We should be able to complete four runs in much
	// less than 10s.
	completedRuns := 0
	for t0 := time.Now(); time.Since(t0) < 10*time.Second; {
		pulls := pullReqs.Count()
		if pulls >= 16 && trashReqs.Count() == pulls+4 {
			break
		}
		// Once the 1st run has started automatically, we
		// start sending a single SIGUSR1 at the end of each
		// run, to ensure we get exactly 4 runs in total.
		if pulls > 0 && pulls%4 == 0 && pulls <= 12 && pulls/4 > completedRuns {
			completedRuns = pulls / 4
			c.Logf("completed run %d, sending SIGUSR1 to trigger next run", completedRuns)
			procself.Signal(syscall.SIGUSR1)
		}
		time.Sleep(time.Millisecond)
	}
	cancel()
	<-done
	c.Check(pullReqs.Count(), check.Equals, 16)
	c.Check(trashReqs.Count(), check.Equals, 20)

	metrics := arvadostest.GatherMetricsAsString(srv.Metrics.reg)
	c.Check(metrics, check.Matches, `(?ms).*\narvados_keepbalance_changeset_compute_seconds_count 4\n.*`)
}
