package main

import (
	_ "encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/arvados"

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
			io.WriteString(w, `{"items_available":2,"items":[
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
		} else if strings.Contains(r.Form.Get("filters"), `"modified_at","\u003e="`) {
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
	return s.serveStatic("/arvados/v1/keep_services",
		`{"items":[],"items_available":0}`)
}

func (s *stubServer) serveFourDiskKeepServices() *reqTracker {
	return s.serveStatic("/arvados/v1/keep_services", `{"items_available":5,"items":[
		{"uuid":"zzzzz-bi6l4-000000000000000","service_host":"keep0.zzzzz.arvadosapi.com","service_port":25107,"service_ssl_flag":false,"service_type":"disk"},
		{"uuid":"zzzzz-bi6l4-000000000000001","service_host":"keep1.zzzzz.arvadosapi.com","service_port":25107,"service_ssl_flag":false,"service_type":"disk"},
		{"uuid":"zzzzz-bi6l4-000000000000002","service_host":"keep2.zzzzz.arvadosapi.com","service_port":25107,"service_ssl_flag":false,"service_type":"disk"},
		{"uuid":"zzzzz-bi6l4-000000000000003","service_host":"keep3.zzzzz.arvadosapi.com","service_port":25107,"service_ssl_flag":false,"service_type":"disk"},
		{"uuid":"zzzzz-bi6l4-h0a0xwut9qa6g3a","service_host":"keep.zzzzz.arvadosapi.com","service_port":25333,"service_ssl_flag":true,"service_type":"proxy"}]}`)
}

func (s *stubServer) serveKeepstoreIndexFoo4Bar1() *reqTracker {
	rt := &reqTracker{}
	s.mux.HandleFunc("/index/", func(w http.ResponseWriter, r *http.Request) {
		count := rt.Add(r)
		if r.Host == "keep0.zzzzz.arvadosapi.com:25107" {
			io.WriteString(w, "37b51d194a7513e45b56f6524f2d51f2+3 12345678\n")
		}
		fmt.Fprintf(w, "acbd18db4cc2f85cedef654fccc4a4d8+3 %d\n\n", 12345678+count)
	})
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
	config Config
}

// make a log.Logger that writes to the current test's c.Log().
func (s *runSuite) logger(c *check.C) *log.Logger {
	r, w := io.Pipe()
	go func() {
		buf := make([]byte, 10000)
		for {
			n, err := r.Read(buf)
			if n > 0 {
				if buf[n-1] == '\n' {
					n--
				}
				c.Log(string(buf[:n]))
			}
			if err != nil {
				break
			}
		}
	}()
	return log.New(w, "", log.LstdFlags)
}

func (s *runSuite) SetUpTest(c *check.C) {
	s.config = Config{
		Client: arvados.Client{
			AuthToken: "xyzzy",
			APIHost:   "zzzzz.arvadosapi.com",
			Client:    s.stub.Start()},
		KeepServiceTypes: []string{"disk"}}
	s.stub.serveDiscoveryDoc()
	s.stub.logf = c.Logf
}

func (s *runSuite) TearDownTest(c *check.C) {
	s.stub.Close()
}

func (s *runSuite) TestRefuseZeroCollections(c *check.C) {
	opts := RunOptions{
		CommitPulls: true,
		CommitTrash: true,
		Logger:      s.logger(c),
	}
	s.stub.serveCurrentUserAdmin()
	s.stub.serveZeroCollections()
	s.stub.serveFourDiskKeepServices()
	s.stub.serveKeepstoreIndexFoo4Bar1()
	trashReqs := s.stub.serveKeepstoreTrash()
	pullReqs := s.stub.serveKeepstorePull()
	_, err := (&Balancer{}).Run(s.config, opts)
	c.Check(err, check.ErrorMatches, "received zero collections")
	c.Check(trashReqs.Count(), check.Equals, 4)
	c.Check(pullReqs.Count(), check.Equals, 0)
}

func (s *runSuite) TestServiceTypes(c *check.C) {
	opts := RunOptions{
		CommitPulls: true,
		CommitTrash: true,
		Logger:      s.logger(c),
	}
	s.config.KeepServiceTypes = []string{"unlisted-type"}
	s.stub.serveCurrentUserAdmin()
	s.stub.serveFooBarFileCollections()
	s.stub.serveFourDiskKeepServices()
	indexReqs := s.stub.serveKeepstoreIndexFoo4Bar1()
	trashReqs := s.stub.serveKeepstoreTrash()
	_, err := (&Balancer{}).Run(s.config, opts)
	c.Check(err, check.IsNil)
	c.Check(indexReqs.Count(), check.Equals, 0)
	c.Check(trashReqs.Count(), check.Equals, 0)
}

func (s *runSuite) TestRefuseNonAdmin(c *check.C) {
	opts := RunOptions{
		CommitPulls: true,
		CommitTrash: true,
		Logger:      s.logger(c),
	}
	s.stub.serveCurrentUserNotAdmin()
	s.stub.serveZeroCollections()
	s.stub.serveFourDiskKeepServices()
	trashReqs := s.stub.serveKeepstoreTrash()
	pullReqs := s.stub.serveKeepstorePull()
	_, err := (&Balancer{}).Run(s.config, opts)
	c.Check(err, check.ErrorMatches, "current user .* is not .* admin user")
	c.Check(trashReqs.Count(), check.Equals, 0)
	c.Check(pullReqs.Count(), check.Equals, 0)
}

func (s *runSuite) TestDetectSkippedCollections(c *check.C) {
	opts := RunOptions{
		CommitPulls: true,
		CommitTrash: true,
		Logger:      s.logger(c),
	}
	s.stub.serveCurrentUserAdmin()
	s.stub.serveCollectionsButSkipOne()
	s.stub.serveFourDiskKeepServices()
	s.stub.serveKeepstoreIndexFoo4Bar1()
	trashReqs := s.stub.serveKeepstoreTrash()
	pullReqs := s.stub.serveKeepstorePull()
	_, err := (&Balancer{}).Run(s.config, opts)
	c.Check(err, check.ErrorMatches, `Retrieved 2 collections with modtime <= .* but server now reports there are 3 collections.*`)
	c.Check(trashReqs.Count(), check.Equals, 4)
	c.Check(pullReqs.Count(), check.Equals, 0)
}

func (s *runSuite) TestDryRun(c *check.C) {
	opts := RunOptions{
		CommitPulls: false,
		CommitTrash: false,
		Logger:      s.logger(c),
	}
	s.stub.serveCurrentUserAdmin()
	s.stub.serveFooBarFileCollections()
	s.stub.serveFourDiskKeepServices()
	s.stub.serveKeepstoreIndexFoo4Bar1()
	trashReqs := s.stub.serveKeepstoreTrash()
	pullReqs := s.stub.serveKeepstorePull()
	var bal Balancer
	_, err := bal.Run(s.config, opts)
	c.Check(err, check.IsNil)
	c.Check(trashReqs.Count(), check.Equals, 0)
	c.Check(pullReqs.Count(), check.Equals, 0)
	stats := bal.getStatistics()
	c.Check(stats.pulls, check.Not(check.Equals), 0)
	c.Check(stats.underrep.replicas, check.Not(check.Equals), 0)
	c.Check(stats.overrep.replicas, check.Not(check.Equals), 0)
}

func (s *runSuite) TestCommit(c *check.C) {
	opts := RunOptions{
		CommitPulls: true,
		CommitTrash: true,
		Logger:      s.logger(c),
		Dumper:      s.logger(c),
	}
	s.stub.serveCurrentUserAdmin()
	s.stub.serveFooBarFileCollections()
	s.stub.serveFourDiskKeepServices()
	s.stub.serveKeepstoreIndexFoo4Bar1()
	trashReqs := s.stub.serveKeepstoreTrash()
	pullReqs := s.stub.serveKeepstorePull()
	var bal Balancer
	_, err := bal.Run(s.config, opts)
	c.Check(err, check.IsNil)
	c.Check(trashReqs.Count(), check.Equals, 8)
	c.Check(pullReqs.Count(), check.Equals, 4)
	stats := bal.getStatistics()
	// "foo" block is overreplicated by 2
	c.Check(stats.trashes, check.Equals, 2)
	// "bar" block is underreplicated by 1, and its only copy is
	// in a poor rendezvous position
	c.Check(stats.pulls, check.Equals, 2)
}

func (s *runSuite) TestRunForever(c *check.C) {
	opts := RunOptions{
		CommitPulls: true,
		CommitTrash: true,
		Logger:      s.logger(c),
		Dumper:      s.logger(c),
	}
	s.stub.serveCurrentUserAdmin()
	s.stub.serveFooBarFileCollections()
	s.stub.serveFourDiskKeepServices()
	s.stub.serveKeepstoreIndexFoo4Bar1()
	trashReqs := s.stub.serveKeepstoreTrash()
	pullReqs := s.stub.serveKeepstorePull()

	stop := make(chan interface{})
	s.config.RunPeriod = arvados.Duration(time.Millisecond)
	go RunForever(s.config, opts, stop)

	// Each run should send 4 pull lists + 4 trash lists. The
	// first run should also send 4 empty trash lists at
	// startup. We should complete all four runs in much less than
	// a second.
	for t0 := time.Now(); pullReqs.Count() < 16 && time.Since(t0) < 10*time.Second; {
		time.Sleep(time.Millisecond)
	}
	stop <- true
	c.Check(pullReqs.Count() >= 16, check.Equals, true)
	c.Check(trashReqs.Count(), check.Equals, pullReqs.Count()+4)
}
