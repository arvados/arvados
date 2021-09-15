// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"

	"git.arvados.org/arvados.git/lib/config"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadosclient"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"git.arvados.org/arvados.git/sdk/go/keepclient"
	check "gopkg.in/check.v1"
)

var testAPIHost = os.Getenv("ARVADOS_API_HOST")

var _ = check.Suite(&IntegrationSuite{})

// IntegrationSuite tests need an API server and a keep-web server
type IntegrationSuite struct {
	testServer *server
	ArvConfig  *arvados.Config
}

func (s *IntegrationSuite) TestNoToken(c *check.C) {
	for _, token := range []string{
		"",
		"bogustoken",
	} {
		hdr, body, _ := s.runCurl(c, token, "collections.example.com", "/collections/"+arvadostest.FooCollection+"/foo")
		c.Check(hdr, check.Matches, `(?s)HTTP/1.1 404 Not Found\r\n.*`)
		c.Check(body, check.Equals, notFoundMessage+"\n")

		if token != "" {
			hdr, body, _ = s.runCurl(c, token, "collections.example.com", "/collections/download/"+arvadostest.FooCollection+"/"+token+"/foo")
			c.Check(hdr, check.Matches, `(?s)HTTP/1.1 404 Not Found\r\n.*`)
			c.Check(body, check.Equals, notFoundMessage+"\n")
		}

		hdr, body, _ = s.runCurl(c, token, "collections.example.com", "/bad-route")
		c.Check(hdr, check.Matches, `(?s)HTTP/1.1 404 Not Found\r\n.*`)
		c.Check(body, check.Equals, notFoundMessage+"\n")
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
			c.Check(body, check.Equals, notFoundMessage+"\n")
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
	s.testServer.Config.cluster.Users.AnonymousUserToken = arvadostest.AnonymousToken
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
	testHost, testPort, _ := net.SplitHostPort(s.testServer.Addr)
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

func (s *IntegrationSuite) TestMetrics(c *check.C) {
	s.testServer.Config.cluster.Services.WebDAVDownload.ExternalURL.Host = s.testServer.Addr
	origin := "http://" + s.testServer.Addr
	req, _ := http.NewRequest("GET", origin+"/notfound", nil)
	_, err := http.DefaultClient.Do(req)
	c.Assert(err, check.IsNil)
	req, _ = http.NewRequest("GET", origin+"/by_id/", nil)
	req.Header.Set("Authorization", "Bearer "+arvadostest.ActiveToken)
	resp, err := http.DefaultClient.Do(req)
	c.Assert(err, check.IsNil)
	c.Check(resp.StatusCode, check.Equals, http.StatusOK)
	for i := 0; i < 2; i++ {
		req, _ = http.NewRequest("GET", origin+"/foo", nil)
		req.Host = arvadostest.FooCollection + ".example.com"
		req.Header.Set("Authorization", "Bearer "+arvadostest.ActiveToken)
		resp, err = http.DefaultClient.Do(req)
		c.Assert(err, check.IsNil)
		c.Check(resp.StatusCode, check.Equals, http.StatusOK)
		buf, _ := ioutil.ReadAll(resp.Body)
		c.Check(buf, check.DeepEquals, []byte("foo"))
		resp.Body.Close()
	}

	s.testServer.Config.Cache.updateGauges()

	req, _ = http.NewRequest("GET", origin+"/metrics.json", nil)
	resp, err = http.DefaultClient.Do(req)
	c.Assert(err, check.IsNil)
	c.Check(resp.StatusCode, check.Equals, http.StatusUnauthorized)

	req, _ = http.NewRequest("GET", origin+"/metrics.json", nil)
	req.Header.Set("Authorization", "Bearer badtoken")
	resp, err = http.DefaultClient.Do(req)
	c.Assert(err, check.IsNil)
	c.Check(resp.StatusCode, check.Equals, http.StatusForbidden)

	req, _ = http.NewRequest("GET", origin+"/metrics.json", nil)
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
	c.Check(counters["arvados_keepweb_collectioncache_requests//"].Value, check.Equals, int64(2))
	c.Check(counters["arvados_keepweb_collectioncache_api_calls//"].Value, check.Equals, int64(2))
	c.Check(counters["arvados_keepweb_collectioncache_hits//"].Value, check.Equals, int64(1))
	c.Check(counters["arvados_keepweb_collectioncache_pdh_hits//"].Value, check.Equals, int64(1))
	c.Check(gauges["arvados_keepweb_collectioncache_cached_manifests//"].Value, check.Equals, float64(1))
	// FooCollection's cached manifest size is 45 ("1f4b0....+45") plus one 51-byte blob signature
	c.Check(gauges["arvados_keepweb_sessions_cached_collection_bytes//"].Value, check.Equals, float64(45+51))

	// If the Host header indicates a collection, /metrics.json
	// refers to a file in the collection -- the metrics handler
	// must not intercept that route.
	req, _ = http.NewRequest("GET", origin+"/metrics.json", nil)
	req.Host = strings.Replace(arvadostest.FooCollectionPDH, "+", "-", -1) + ".example.com"
	req.Header.Set("Authorization", "Bearer "+arvadostest.ActiveToken)
	resp, err = http.DefaultClient.Do(req)
	c.Assert(err, check.IsNil)
	c.Check(resp.StatusCode, check.Equals, http.StatusNotFound)
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
	ldr := config.NewLoader(bytes.NewBufferString("Clusters: {zzzzz: {}}"), ctxlog.TestLogger(c))
	ldr.Path = "-"
	arvCfg, err := ldr.Load()
	c.Check(err, check.IsNil)
	cfg := newConfig(arvCfg)
	c.Assert(err, check.IsNil)
	cfg.Client = arvados.Client{
		APIHost:  testAPIHost,
		Insecure: true,
	}
	listen := "127.0.0.1:0"
	cfg.cluster.Services.WebDAV.InternalURLs[arvados.URL{Host: listen}] = arvados.ServiceInstance{}
	cfg.cluster.Services.WebDAVDownload.InternalURLs[arvados.URL{Host: listen}] = arvados.ServiceInstance{}
	cfg.cluster.ManagementToken = arvadostest.ManagementToken
	cfg.cluster.SystemRootToken = arvadostest.SystemRootToken
	cfg.cluster.Users.AnonymousUserToken = arvadostest.AnonymousToken
	s.ArvConfig = arvCfg
	s.testServer = &server{Config: cfg}
	err = s.testServer.Start(ctxlog.TestLogger(c))
	c.Assert(err, check.Equals, nil)
}

func (s *IntegrationSuite) TearDownTest(c *check.C) {
	var err error
	if s.testServer != nil {
		err = s.testServer.Close()
	}
	c.Check(err, check.Equals, nil)
}

// Gocheck boilerplate
func Test(t *testing.T) {
	check.TestingT(t)
}
