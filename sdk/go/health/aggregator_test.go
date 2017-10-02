package health

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/arvadostest"
	"gopkg.in/check.v1"
)

type AggregatorSuite struct {
	handler *Aggregator
	req     *http.Request
	resp    *httptest.ResponseRecorder
}

// Gocheck boilerplate
var _ = check.Suite(&AggregatorSuite{})

func (s *AggregatorSuite) TestInterface(c *check.C) {
	var _ http.Handler = &Aggregator{}
}

func (s *AggregatorSuite) SetUpTest(c *check.C) {
	s.handler = &Aggregator{Config: &arvados.Config{
		Clusters: map[string]arvados.Cluster{
			"zzzzz": {
				ManagementToken: arvadostest.ManagementToken,
				SystemNodes:     map[string]arvados.SystemNode{},
			},
		},
	}}
	s.req = httptest.NewRequest("GET", "/_health/all", nil)
	s.req.Header.Set("Authorization", "Bearer "+arvadostest.ManagementToken)
	s.resp = httptest.NewRecorder()
}

func (s *AggregatorSuite) TestNoAuth(c *check.C) {
	s.req.Header.Del("Authorization")
	s.handler.ServeHTTP(s.resp, s.req)
	s.checkError(c)
	c.Check(s.resp.Code, check.Equals, http.StatusUnauthorized)
}

func (s *AggregatorSuite) TestBadAuth(c *check.C) {
	s.req.Header.Set("Authorization", "xyzzy")
	s.handler.ServeHTTP(s.resp, s.req)
	s.checkError(c)
	c.Check(s.resp.Code, check.Equals, http.StatusUnauthorized)
}

func (s *AggregatorSuite) TestEmptyConfig(c *check.C) {
	s.handler.ServeHTTP(s.resp, s.req)
	s.checkOK(c)
}

type unhealthyHandler struct{}

func (*unhealthyHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	resp.Write([]byte(`{"health":"ERROR"}`))
}

func (s *AggregatorSuite) TestUnhealthy(c *check.C) {
	srv := httptest.NewServer(&unhealthyHandler{})
	defer srv.Close()

	var port string
	if parts := strings.Split(srv.URL, ":"); len(parts) < 3 {
		panic(srv.URL)
	} else {
		port = parts[len(parts)-1]
	}
	s.handler.Config.Clusters["zzzzz"].SystemNodes["localhost"] = arvados.SystemNode{
		Keepstore: arvados.Keepstore{Listen: ":" + port},
	}
	s.handler.ServeHTTP(s.resp, s.req)
	s.checkUnhealthy(c)
}

func (s *AggregatorSuite) checkError(c *check.C) {
	c.Check(s.resp.Code, check.Not(check.Equals), http.StatusOK)
	var body map[string]interface{}
	err := json.NewDecoder(s.resp.Body).Decode(&body)
	c.Check(err, check.IsNil)
	c.Check(body["health"], check.Not(check.Equals), "OK")
}

func (s *AggregatorSuite) checkUnhealthy(c *check.C) {
	c.Check(s.resp.Code, check.Equals, http.StatusOK)
	var body map[string]interface{}
	err := json.NewDecoder(s.resp.Body).Decode(&body)
	c.Check(err, check.IsNil)
	c.Check(body["health"], check.Equals, "ERROR")
}

func (s *AggregatorSuite) checkOK(c *check.C) {
	c.Check(s.resp.Code, check.Equals, http.StatusOK)
	var body map[string]interface{}
	err := json.NewDecoder(s.resp.Body).Decode(&body)
	c.Check(err, check.IsNil)
	c.Check(body["health"], check.Equals, "OK")
}
