// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepweb

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
	"time"

	"git.arvados.org/arvados.git/lib/config"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadosclient"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"git.arvados.org/arvados.git/sdk/go/httpserver"
	"git.arvados.org/arvados.git/sdk/go/keepclient"
	"github.com/prometheus/client_golang/prometheus"
	check "gopkg.in/check.v1"
)

var testAPIHost = os.Getenv("ARVADOS_API_HOST")

var _ = check.Suite(&IntegrationSuite{})

// IntegrationSuite tests need an API server and a keep-web server
type IntegrationSuite struct {
	testServer *httptest.Server
	handler    *handler
}

func (s *IntegrationSuite) TestNoToken(c *check.C) {
	for _, token := range []string{
		"",
		"bogustoken",
	} {
		hdr, body, _ := s.runCurl(c, token, "collections.example.com", "/collections/"+arvadostest.FooCollection+"/foo")
		c.Check(hdr, check.Matches, `(?s)HTTP/1.1 404 Not Found\r\n.*`)
		c.Check(strings.TrimSpace(body), check.Equals, notFoundMessage)

		if token != "" {
			hdr, body, _ = s.runCurl(c, token, "collections.example.com", "/collections/download/"+arvadostest.FooCollection+"/"+token+"/foo")
			c.Check(hdr, check.Matches, `(?s)HTTP/1.1 404 Not Found\r\n.*`)
			c.Check(strings.TrimSpace(body), check.Equals, notFoundMessage)
		}

		hdr, body, _ = s.runCurl(c, token, "collections.example.com", "/bad-route")
		c.Check(hdr, check.Matches, `(?s)HTTP/1.1 404 Not Found\r\n.*`)
		c.Check(strings.TrimSpace(body), check.Equals, notFoundMessage)
	}
}

// TODO: Move most cases to functional tests -- at least use Go's own
// http client instead of forking curl. Just leave enough of an
// integration test to assure that the documented way of invoking curl
// really works against the server.
func (s *IntegrationSuite) Test404(c *check.C) {
	for _, uri := range []string{
		// Routing errors (always 404 regardless of what's stored in Keep)
		"/foo",
		"/download",
		"/collections",
		"/collections/",
		// Implicit/generated index is not implemented yet;
		// until then, return 404.
		"/collections/" + arvadostest.FooCollection,
		"/collections/" + arvadostest.FooCollection + "/",
		"/collections/" + arvadostest.FooBarDirCollection + "/dir1",
		"/collections/" + arvadostest.FooBarDirCollection + "/dir1/",
		// Non-existent file in collection
		"/collections/" + arvadostest.FooCollection + "/theperthcountyconspiracy",
		"/collections/download/" + arvadostest.FooCollection + "/" + arvadostest.ActiveToken + "/theperthcountyconspiracy",
		// Non-existent collection
		"/collections/" + arvadostest.NonexistentCollection,
		"/collections/" + arvadostest.NonexistentCollection + "/",
		"/collections/" + arvadostest.NonexistentCollection + "/theperthcountyconspiracy",
		"/collections/download/" + arvadostest.NonexistentCollection + "/" + arvadostest.ActiveToken + "/theperthcountyconspiracy",
	} {
		hdr, body, _ := s.runCurl(c, arvadostest.ActiveToken, "collections.example.com", uri)
		c.Check(hdr, check.Matches, "(?s)HTTP/1.1 404 Not Found\r\n.*")
		if len(body) > 0 {
			c.Check(strings.TrimSpace(body), check.Equals, notFoundMessage)
		}
	}
}

func (s *IntegrationSuite) Test1GBFile(c *check.C) {
	if testing.Short() {
		c.Skip("skipping 1GB integration test in short mode")
	}
	s.test100BlockFile(c, 10000000)
}

func (s *IntegrationSuite) Test100BlockFile(c *check.C) {
	if testing.Short() {
		// 3 MB
		s.test100BlockFile(c, 30000)
	} else {
		// 300 MB
		s.test100BlockFile(c, 3000000)
	}
}

func (s *IntegrationSuite) test100BlockFile(c *check.C, blocksize int) {
	testdata := make([]byte, blocksize)
	for i := 0; i < blocksize; i++ {
		testdata[i] = byte(' ')
	}
	arv, err := arvadosclient.MakeArvadosClient()
	c.Assert(err, check.Equals, nil)
	arv.ApiToken = arvadostest.ActiveToken
	kc, err := keepclient.MakeKeepClient(arv)
	c.Assert(err, check.Equals, nil)
	loc, _, err := kc.PutB(testdata[:])
	c.Assert(err, check.Equals, nil)
	mtext := "."
	for i := 0; i < 100; i++ {
		mtext = mtext + " " + loc
	}
	mtext = mtext + fmt.Sprintf(" 0:%d00:testdata.bin\n", blocksize)
	coll := map[string]interface{}{}
	err = arv.Create("collections",
		map[string]interface{}{
			"collection": map[string]interface{}{
				"name":          fmt.Sprintf("testdata blocksize=%d", blocksize),
				"manifest_text": mtext,
			},
		}, &coll)
	c.Assert(err, check.Equals, nil)
	uuid := coll["uuid"].(string)

	hdr, body, size := s.runCurl(c, arv.ApiToken, uuid+".collections.example.com", "/testdata.bin")
	c.Check(hdr, check.Matches, `(?s)HTTP/1.1 200 OK\r\n.*`)
	c.Check(hdr, check.Matches, `(?si).*Content-length: `+fmt.Sprintf("%d00", blocksize)+`\r\n.*`)
	c.Check([]byte(body)[:1234], check.DeepEquals, testdata[:1234])
	c.Check(size, check.Equals, int64(blocksize)*100)
}

type curlCase struct {
	auth    string
	host    string
	path    string
	dataMD5 string
}

func (s *IntegrationSuite) Test200(c *check.C) {
	s.handler.Cluster.Users.AnonymousUserToken = arvadostest.AnonymousToken
	for _, spec := range []curlCase{
		// My collection
		{
			auth:    arvadostest.ActiveToken,
			host:    arvadostest.FooCollection + "--collections.example.com",
			path:    "/foo",
			dataMD5: "acbd18db4cc2f85cedef654fccc4a4d8",
		},
		{
			auth:    arvadostest.ActiveToken,
			host:    arvadostest.FooCollection + ".collections.example.com",
			path:    "/foo",
			dataMD5: "acbd18db4cc2f85cedef654fccc4a4d8",
		},
		{
			host:    strings.Replace(arvadostest.FooCollectionPDH, "+", "-", 1) + ".collections.example.com",
			path:    "/t=" + arvadostest.ActiveToken + "/foo",
			dataMD5: "acbd18db4cc2f85cedef654fccc4a4d8",
		},
		{
			path:    "/c=" + arvadostest.FooCollectionPDH + "/t=" + arvadostest.ActiveToken + "/foo",
			dataMD5: "acbd18db4cc2f85cedef654fccc4a4d8",
		},
		{
			path:    "/c=" + strings.Replace(arvadostest.FooCollectionPDH, "+", "-", 1) + "/t=" + arvadostest.ActiveToken + "/_/foo",
			dataMD5: "acbd18db4cc2f85cedef654fccc4a4d8",
		},
		{
			path:    "/collections/download/" + arvadostest.FooCollection + "/" + arvadostest.ActiveToken + "/foo",
			dataMD5: "acbd18db4cc2f85cedef654fccc4a4d8",
		},
		{
			auth:    "tokensobogus",
			path:    "/collections/download/" + arvadostest.FooCollection + "/" + arvadostest.ActiveToken + "/foo",
			dataMD5: "acbd18db4cc2f85cedef654fccc4a4d8",
		},
		{
			auth:    arvadostest.ActiveToken,
			path:    "/collections/download/" + arvadostest.FooCollection + "/" + arvadostest.ActiveToken + "/foo",
			dataMD5: "acbd18db4cc2f85cedef654fccc4a4d8",
		},
		{
			auth:    arvadostest.AnonymousToken,
			path:    "/collections/download/" + arvadostest.FooCollection + "/" + arvadostest.ActiveToken + "/foo",
			dataMD5: "acbd18db4cc2f85cedef654fccc4a4d8",
		},

		// Anonymously accessible data
		{
			path:    "/c=" + arvadostest.HelloWorldCollection + "/Hello%20world.txt",
			dataMD5: "f0ef7081e1539ac00ef5b761b4fb01b3",
		},
		{
			host:    arvadostest.HelloWorldCollection + ".collections.example.com",
			path:    "/Hello%20world.txt",
			dataMD5: "f0ef7081e1539ac00ef5b761b4fb01b3",
		},
		{
			host:    arvadostest.HelloWorldCollection + ".collections.example.com",
			path:    "/_/Hello%20world.txt",
			dataMD5: "f0ef7081e1539ac00ef5b761b4fb01b3",
		},
		{
			path:    "/collections/" + arvadostest.HelloWorldCollection + "/Hello%20world.txt",
			dataMD5: "f0ef7081e1539ac00ef5b761b4fb01b3",
		},
		{
			auth:    arvadostest.ActiveToken,
			path:    "/collections/" + arvadostest.HelloWorldCollection + "/Hello%20world.txt",
			dataMD5: "f0ef7081e1539ac00ef5b761b4fb01b3",
		},
		{
			auth:    arvadostest.SpectatorToken,
			path:    "/collections/" + arvadostest.HelloWorldCollection + "/Hello%20world.txt",
			dataMD5: "f0ef7081e1539ac00ef5b761b4fb01b3",
		},
		{
			auth:    arvadostest.SpectatorToken,
			host:    arvadostest.HelloWorldCollection + "--collections.example.com",
			path:    "/Hello%20world.txt",
			dataMD5: "f0ef7081e1539ac00ef5b761b4fb01b3",
		},
		{
			auth:    arvadostest.SpectatorToken,
			path:    "/collections/download/" + arvadostest.HelloWorldCollection + "/" + arvadostest.SpectatorToken + "/Hello%20world.txt",
			dataMD5: "f0ef7081e1539ac00ef5b761b4fb01b3",
		},
	} {
		host := spec.host
		if host == "" {
			host = "collections.example.com"
		}
		hdr, body, _ := s.runCurl(c, spec.auth, host, spec.path)
		c.Check(hdr, check.Matches, `(?s)HTTP/1.1 200 OK\r\n.*`)
		if strings.HasSuffix(spec.path, ".txt") {
			c.Check(hdr, check.Matches, `(?s).*\r\nContent-Type: text/plain.*`)
			// TODO: Check some types that aren't
			// automatically detected by Go's http server
			// by sniffing the content.
		}
		c.Check(fmt.Sprintf("%x", md5.Sum([]byte(body))), check.Equals, spec.dataMD5)
	}
}

// Return header block and body.
func (s *IntegrationSuite) runCurl(c *check.C, auth, host, uri string, args ...string) (hdr, bodyPart string, bodySize int64) {
	curlArgs := []string{"--silent", "--show-error", "--include"}
	testHost, testPort, _ := net.SplitHostPort(s.testServer.URL[7:])
	curlArgs = append(curlArgs, "--resolve", host+":"+testPort+":"+testHost)
	if strings.Contains(auth, " ") {
		// caller supplied entire Authorization header value
		curlArgs = append(curlArgs, "-H", "Authorization: "+auth)
	} else if auth != "" {
		// caller supplied Arvados token
		curlArgs = append(curlArgs, "-H", "Authorization: Bearer "+auth)
	}
	curlArgs = append(curlArgs, args...)
	curlArgs = append(curlArgs, "http://"+host+":"+testPort+uri)
	c.Log(fmt.Sprintf("curlArgs == %#v", curlArgs))
	cmd := exec.Command("curl", curlArgs...)
	stdout, err := cmd.StdoutPipe()
	c.Assert(err, check.IsNil)
	cmd.Stderr = os.Stderr
	err = cmd.Start()
	c.Assert(err, check.IsNil)
	buf := make([]byte, 2<<27)
	n, err := io.ReadFull(stdout, buf)
	// Discard (but measure size of) anything past 128 MiB.
	var discarded int64
	if err == io.ErrUnexpectedEOF {
		buf = buf[:n]
	} else {
		c.Assert(err, check.IsNil)
		discarded, err = io.Copy(ioutil.Discard, stdout)
		c.Assert(err, check.IsNil)
	}
	err = cmd.Wait()
	// Without "-f", curl exits 0 as long as it gets a valid HTTP
	// response from the server, even if the response status
	// indicates that the request failed. In our test suite, we
	// always expect a valid HTTP response, and we parse the
	// headers ourselves. If curl exits non-zero, our testing
	// environment is broken.
	c.Assert(err, check.Equals, nil)
	hdrsAndBody := strings.SplitN(string(buf), "\r\n\r\n", 2)
	c.Assert(len(hdrsAndBody), check.Equals, 2)
	hdr = hdrsAndBody[0]
	bodyPart = hdrsAndBody[1]
	bodySize = int64(len(bodyPart)) + discarded
	return
}

// Run a full-featured server, including the metrics/health routes
// that are added by service.Command.
func (s *IntegrationSuite) runServer(c *check.C) (cluster arvados.Cluster, srvaddr string, logbuf *bytes.Buffer) {
	logbuf = &bytes.Buffer{}
	cluster = *s.handler.Cluster
	cluster.Services.WebDAV.InternalURLs = map[arvados.URL]arvados.ServiceInstance{{Scheme: "http", Host: "0.0.0.0:0"}: {}}
	cluster.Services.WebDAVDownload.InternalURLs = map[arvados.URL]arvados.ServiceInstance{{Scheme: "http", Host: "0.0.0.0:0"}: {}}

	var configjson bytes.Buffer
	json.NewEncoder(&configjson).Encode(arvados.Config{Clusters: map[string]arvados.Cluster{"zzzzz": cluster}})
	go Command.RunCommand("keep-web", []string{"-config=-"}, &configjson, os.Stderr, io.MultiWriter(os.Stderr, logbuf))
	for deadline := time.Now().Add(time.Second); deadline.After(time.Now()); time.Sleep(time.Second / 100) {
		if m := regexp.MustCompile(`"Listen":"(.*?)"`).FindStringSubmatch(logbuf.String()); m != nil {
			srvaddr = "http://" + m[1]
			break
		}
	}
	if srvaddr == "" {
		c.Fatal("timed out")
	}
	return
}

// Ensure uploads can take longer than API.RequestTimeout.
//
// Currently, this works only by accident: service.Command cancels the
// request context as usual (there is no exemption), but
// webdav.Handler doesn't notice if the request context is cancelled
// while waiting to send or receive file data.
func (s *IntegrationSuite) TestRequestTimeoutExemption(c *check.C) {
	s.handler.Cluster.API.RequestTimeout = arvados.Duration(time.Second / 2)
	_, srvaddr, _ := s.runServer(c)

	var coll arvados.Collection
	arv, err := arvadosclient.MakeArvadosClient()
	c.Assert(err, check.IsNil)
	arv.ApiToken = arvadostest.ActiveTokenV2
	err = arv.Create("collections", map[string]interface{}{"ensure_unique_name": true}, &coll)
	c.Assert(err, check.IsNil)

	pr, pw := io.Pipe()
	go func() {
		time.Sleep(time.Second)
		pw.Write(make([]byte, 10000000))
		pw.Close()
	}()
	req, _ := http.NewRequest("PUT", srvaddr+"/testfile", pr)
	req.Host = coll.UUID + ".example"
	req.Header.Set("Authorization", "Bearer "+arvadostest.ActiveTokenV2)
	resp, err := http.DefaultClient.Do(req)
	c.Assert(err, check.IsNil)
	c.Check(resp.StatusCode, check.Equals, http.StatusCreated)

	req, _ = http.NewRequest("GET", srvaddr+"/testfile", nil)
	req.Host = coll.UUID + ".example"
	req.Header.Set("Authorization", "Bearer "+arvadostest.ActiveTokenV2)
	resp, err = http.DefaultClient.Do(req)
	c.Assert(err, check.IsNil)
	c.Check(resp.StatusCode, check.Equals, http.StatusOK)
	time.Sleep(time.Second)
	body, err := ioutil.ReadAll(resp.Body)
	c.Check(err, check.IsNil)
	c.Check(len(body), check.Equals, 10000000)
}

func (s *IntegrationSuite) TestHealthCheckPing(c *check.C) {
	cluster, srvaddr, _ := s.runServer(c)
	req, _ := http.NewRequest("GET", srvaddr+"/_health/ping", nil)
	req.Header.Set("Authorization", "Bearer "+cluster.ManagementToken)
	resp, err := http.DefaultClient.Do(req)
	c.Assert(err, check.IsNil)
	c.Check(resp.StatusCode, check.Equals, http.StatusOK)
	body, _ := ioutil.ReadAll(resp.Body)
	c.Check(string(body), check.Matches, `{"health":"OK"}\n`)
}

func (s *IntegrationSuite) TestMetrics(c *check.C) {
	cluster, srvaddr, _ := s.runServer(c)

	req, _ := http.NewRequest("GET", srvaddr+"/notfound", nil)
	req.Host = cluster.Services.WebDAVDownload.ExternalURL.Host
	_, err := http.DefaultClient.Do(req)
	c.Assert(err, check.IsNil)
	req, _ = http.NewRequest("GET", srvaddr+"/by_id/", nil)
	req.Host = cluster.Services.WebDAVDownload.ExternalURL.Host
	req.Header.Set("Authorization", "Bearer "+arvadostest.ActiveToken)
	resp, err := http.DefaultClient.Do(req)
	c.Assert(err, check.IsNil)
	c.Assert(resp.StatusCode, check.Equals, http.StatusOK)
	for i := 0; i < 2; i++ {
		req, _ = http.NewRequest("GET", srvaddr+"/foo", nil)
		req.Host = arvadostest.FooCollection + ".example.com"
		req.Header.Set("Authorization", "Bearer "+arvadostest.ActiveToken)
		resp, err = http.DefaultClient.Do(req)
		c.Assert(err, check.IsNil)
		c.Check(resp.StatusCode, check.Equals, http.StatusOK)
		buf, _ := ioutil.ReadAll(resp.Body)
		c.Check(buf, check.DeepEquals, []byte("foo"))
		resp.Body.Close()
	}

	time.Sleep(metricsUpdateInterval * 2)

	req, _ = http.NewRequest("GET", srvaddr+"/metrics.json", nil)
	req.Host = cluster.Services.WebDAVDownload.ExternalURL.Host
	resp, err = http.DefaultClient.Do(req)
	c.Assert(err, check.IsNil)
	c.Check(resp.StatusCode, check.Equals, http.StatusUnauthorized)

	req, _ = http.NewRequest("GET", srvaddr+"/metrics.json", nil)
	req.Host = cluster.Services.WebDAVDownload.ExternalURL.Host
	req.Header.Set("Authorization", "Bearer badtoken")
	resp, err = http.DefaultClient.Do(req)
	c.Assert(err, check.IsNil)
	c.Check(resp.StatusCode, check.Equals, http.StatusForbidden)

	req, _ = http.NewRequest("GET", srvaddr+"/metrics.json", nil)
	req.Host = cluster.Services.WebDAVDownload.ExternalURL.Host
	req.Header.Set("Authorization", "Bearer "+arvadostest.ManagementToken)
	resp, err = http.DefaultClient.Do(req)
	c.Assert(err, check.IsNil)
	c.Check(resp.StatusCode, check.Equals, http.StatusOK)
	type summary struct {
		SampleCount string
		SampleSum   float64
	}
	type counter struct {
		Value int64
	}
	type gauge struct {
		Value float64
	}
	var ents []struct {
		Name   string
		Help   string
		Type   string
		Metric []struct {
			Label []struct {
				Name  string
				Value string
			}
			Counter counter
			Gauge   gauge
			Summary summary
		}
	}
	json.NewDecoder(resp.Body).Decode(&ents)
	summaries := map[string]summary{}
	gauges := map[string]gauge{}
	counters := map[string]counter{}
	for _, e := range ents {
		for _, m := range e.Metric {
			labels := map[string]string{}
			for _, lbl := range m.Label {
				labels[lbl.Name] = lbl.Value
			}
			summaries[e.Name+"/"+labels["method"]+"/"+labels["code"]] = m.Summary
			counters[e.Name+"/"+labels["method"]+"/"+labels["code"]] = m.Counter
			gauges[e.Name+"/"+labels["method"]+"/"+labels["code"]] = m.Gauge
		}
	}
	c.Check(summaries["request_duration_seconds/get/200"].SampleSum, check.Not(check.Equals), 0)
	c.Check(summaries["request_duration_seconds/get/200"].SampleCount, check.Equals, "3")
	c.Check(summaries["request_duration_seconds/get/404"].SampleCount, check.Equals, "1")
	c.Check(summaries["time_to_status_seconds/get/404"].SampleCount, check.Equals, "1")
	c.Check(gauges["arvados_keepweb_sessions_cached_session_bytes//"].Value, check.Equals, float64(469))

	// If the Host header indicates a collection, /metrics.json
	// refers to a file in the collection -- the metrics handler
	// must not intercept that route. Ditto health check paths.
	for _, path := range []string{"/metrics.json", "/_health/ping"} {
		c.Logf("path: %q", path)
		req, _ = http.NewRequest("GET", srvaddr+path, nil)
		req.Host = strings.Replace(arvadostest.FooCollectionPDH, "+", "-", -1) + ".example.com"
		req.Header.Set("Authorization", "Bearer "+arvadostest.ActiveToken)
		resp, err = http.DefaultClient.Do(req)
		c.Assert(err, check.IsNil)
		c.Check(resp.StatusCode, check.Equals, http.StatusNotFound)
	}
}

func (s *IntegrationSuite) SetUpSuite(c *check.C) {
	arvadostest.ResetDB(c)
	arvadostest.StartKeep(2, true)

	arv, err := arvadosclient.MakeArvadosClient()
	c.Assert(err, check.Equals, nil)
	arv.ApiToken = arvadostest.ActiveToken
	kc, err := keepclient.MakeKeepClient(arv)
	c.Assert(err, check.Equals, nil)
	kc.PutB([]byte("Hello world\n"))
	kc.PutB([]byte("foo"))
	kc.PutB([]byte("foobar"))
	kc.PutB([]byte("waz"))
}

func (s *IntegrationSuite) TearDownSuite(c *check.C) {
	arvadostest.StopKeep(2)
}

func (s *IntegrationSuite) SetUpTest(c *check.C) {
	arvadostest.ResetEnv()
	logger := ctxlog.TestLogger(c)
	ldr := config.NewLoader(&bytes.Buffer{}, logger)
	cfg, err := ldr.Load()
	c.Assert(err, check.IsNil)
	cluster, err := cfg.GetCluster("")
	c.Assert(err, check.IsNil)

	ctx := ctxlog.Context(context.Background(), logger)

	s.handler = newHandlerOrErrorHandler(ctx, cluster, cluster.SystemRootToken, prometheus.NewRegistry()).(*handler)
	s.testServer = httptest.NewUnstartedServer(
		httpserver.AddRequestIDs(
			httpserver.LogRequests(
				s.handler)))
	s.testServer.Config.BaseContext = func(net.Listener) context.Context { return ctx }
	s.testServer.Start()

	cluster.Services.WebDAV.InternalURLs = map[arvados.URL]arvados.ServiceInstance{{Host: s.testServer.URL[7:]}: {}}
	cluster.Services.WebDAVDownload.InternalURLs = map[arvados.URL]arvados.ServiceInstance{{Host: s.testServer.URL[7:]}: {}}
}

func (s *IntegrationSuite) TearDownTest(c *check.C) {
	if s.testServer != nil {
		s.testServer.Close()
	}
}

// Gocheck boilerplate
func Test(t *testing.T) {
	check.TestingT(t)
}
