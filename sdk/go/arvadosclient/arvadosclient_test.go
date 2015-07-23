package arvadosclient

import (
	"git.curoverse.com/arvados.git/sdk/go/arvadostest"
	. "gopkg.in/check.v1"
	"net/http"
	"os"
	"testing"
)

// Gocheck boilerplate
func Test(t *testing.T) {
	TestingT(t)
}

var _ = Suite(&ServerRequiredSuite{})

// Tests that require the Keep server running
type ServerRequiredSuite struct{}

func (s *ServerRequiredSuite) SetUpSuite(c *C) {
	arvadostest.StartAPI()
	arvadostest.StartKeep()
}

func (s *ServerRequiredSuite) SetUpTest(c *C) {
	arvadostest.ResetEnv()
}

func (s *ServerRequiredSuite) TestMakeArvadosClientSecure(c *C) {
	os.Setenv("ARVADOS_API_HOST_INSECURE", "")
	kc, err := MakeArvadosClient()
	c.Assert(err, Equals, nil)
	c.Check(kc.ApiServer, Equals, os.Getenv("ARVADOS_API_HOST"))
	c.Check(kc.ApiToken, Equals, os.Getenv("ARVADOS_API_TOKEN"))
	c.Check(kc.ApiInsecure, Equals, false)
}

func (s *ServerRequiredSuite) TestMakeArvadosClientInsecure(c *C) {
	os.Setenv("ARVADOS_API_HOST_INSECURE", "true")
	kc, err := MakeArvadosClient()
	c.Assert(err, Equals, nil)
	c.Check(kc.ApiInsecure, Equals, true)
	c.Check(kc.ApiServer, Equals, os.Getenv("ARVADOS_API_HOST"))
	c.Check(kc.ApiToken, Equals, os.Getenv("ARVADOS_API_TOKEN"))
	c.Check(kc.Client.Transport.(*http.Transport).TLSClientConfig.InsecureSkipVerify, Equals, true)
}

func (s *ServerRequiredSuite) TestCreatePipelineTemplate(c *C) {
	arv, err := MakeArvadosClient()

	getback := make(Dict)
	err = arv.Create("pipeline_templates",
		Dict{"pipeline_template": Dict{
			"name": "tmp",
			"components": Dict{
				"c1": map[string]string{"script": "script1"},
				"c2": map[string]string{"script": "script2"}}}},
		&getback)
	c.Assert(err, Equals, nil)
	c.Assert(getback["name"], Equals, "tmp")
	c.Assert(getback["components"].(map[string]interface{})["c2"].(map[string]interface{})["script"], Equals, "script2")

	uuid := getback["uuid"].(string)

	getback = make(Dict)
	err = arv.Get("pipeline_templates", uuid, nil, &getback)
	c.Assert(err, Equals, nil)
	c.Assert(getback["name"], Equals, "tmp")
	c.Assert(getback["components"].(map[string]interface{})["c1"].(map[string]interface{})["script"], Equals, "script1")

	getback = make(Dict)
	err = arv.Update("pipeline_templates", uuid,
		Dict{
			"pipeline_template": Dict{"name": "tmp2"}},
		&getback)
	c.Assert(err, Equals, nil)
	c.Assert(getback["name"], Equals, "tmp2")

	c.Assert(getback["uuid"].(string), Equals, uuid)
	getback = make(Dict)
	err = arv.Delete("pipeline_templates", uuid, nil, &getback)
	c.Assert(err, Equals, nil)
	c.Assert(getback["name"], Equals, "tmp2")
}

func (s *ServerRequiredSuite) TestErrorResponse(c *C) {
	arv, _ := MakeArvadosClient()

	getback := make(Dict)

	{
		err := arv.Create("logs",
			Dict{"log": Dict{"bogus_attr": "foo"}},
			&getback)
		c.Assert(err, ErrorMatches, "arvados API server error: .*")
		c.Assert(err, ErrorMatches, ".*unknown attribute: bogus_attr.*")
		c.Assert(err, FitsTypeOf, APIServerError{})
		c.Assert(err.(APIServerError).HttpStatusCode, Equals, 422)
	}

	{
		err := arv.Create("bogus",
			Dict{"bogus": Dict{}},
			&getback)
		c.Assert(err, ErrorMatches, "arvados API server error: .*")
		c.Assert(err, ErrorMatches, ".*Path not found.*")
		c.Assert(err, FitsTypeOf, APIServerError{})
		c.Assert(err.(APIServerError).HttpStatusCode, Equals, 404)
	}
}

func (s *ServerRequiredSuite) TestAPIDiscovery_Get_defaultCollectionReplication(c *C) {
	arv, err := MakeArvadosClient()
	value, err := arv.Discovery("defaultCollectionReplication")
	c.Assert(err, IsNil)
	c.Assert(value, NotNil)
}

func (s *ServerRequiredSuite) TestAPIDiscovery_Get_noSuchParameter(c *C) {
	arv, err := MakeArvadosClient()
	value, err := arv.Discovery("noSuchParameter")
	c.Assert(err, NotNil)
	c.Assert(value, IsNil)
}
