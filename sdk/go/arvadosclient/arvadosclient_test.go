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
var _ = Suite(&UnitSuite{})

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

func (s *ServerRequiredSuite) TestGetInvalidUUID(c *C) {
	arv, err := MakeArvadosClient()

	getback := make(Dict)
	err = arv.Get("collections", "", nil, &getback)
	c.Assert(err, Equals, ErrInvalidArgument)
	c.Assert(len(getback), Equals, 0)

	err = arv.Get("collections", "zebra-moose-unicorn", nil, &getback)
	c.Assert(err, Equals, ErrInvalidArgument)
	c.Assert(len(getback), Equals, 0)

	err = arv.Get("collections", "acbd18db4cc2f85cedef654fccc4a4d8", nil, &getback)
	c.Assert(err, Equals, ErrInvalidArgument)
	c.Assert(len(getback), Equals, 0)
}

func (s *ServerRequiredSuite) TestGetValidUUID(c *C) {
	arv, err := MakeArvadosClient()

	getback := make(Dict)
	err = arv.Get("collections", "zzzzz-4zz18-abcdeabcdeabcde", nil, &getback)
	c.Assert(err, FitsTypeOf, APIServerError{})
	c.Assert(err.(APIServerError).HttpStatusCode, Equals, http.StatusNotFound)
	c.Assert(len(getback), Equals, 0)

	err = arv.Get("collections", "acbd18db4cc2f85cedef654fccc4a4d8+3", nil, &getback)
	c.Assert(err, FitsTypeOf, APIServerError{})
	c.Assert(err.(APIServerError).HttpStatusCode, Equals, http.StatusNotFound)
	c.Assert(len(getback), Equals, 0)
}

func (s *ServerRequiredSuite) TestInvalidResourceType(c *C) {
	arv, err := MakeArvadosClient()

	getback := make(Dict)
	err = arv.Get("unicorns", "zzzzz-zebra-unicorn7unicorn", nil, &getback)
	c.Assert(err, FitsTypeOf, APIServerError{})
	c.Assert(err.(APIServerError).HttpStatusCode, Equals, http.StatusNotFound)
	c.Assert(len(getback), Equals, 0)

	err = arv.Update("unicorns", "zzzzz-zebra-unicorn7unicorn", nil, &getback)
	c.Assert(err, FitsTypeOf, APIServerError{})
	c.Assert(err.(APIServerError).HttpStatusCode, Equals, http.StatusNotFound)
	c.Assert(len(getback), Equals, 0)

	err = arv.List("unicorns", nil, &getback)
	c.Assert(err, FitsTypeOf, APIServerError{})
	c.Assert(err.(APIServerError).HttpStatusCode, Equals, http.StatusNotFound)
	c.Assert(len(getback), Equals, 0)
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

type UnitSuite struct{}

func (s *UnitSuite) TestUUIDMatch(c *C) {
	c.Assert(UUIDMatch("zzzzz-tpzed-000000000000000"), Equals, true)
	c.Assert(UUIDMatch("zzzzz-zebra-000000000000000"), Equals, true)
	c.Assert(UUIDMatch("00000-00000-zzzzzzzzzzzzzzz"), Equals, true)
	c.Assert(UUIDMatch("ZEBRA-HORSE-AFRICANELEPHANT"), Equals, false)
	c.Assert(UUIDMatch(" zzzzz-tpzed-000000000000000"), Equals, false)
	c.Assert(UUIDMatch("d41d8cd98f00b204e9800998ecf8427e"), Equals, false)
	c.Assert(UUIDMatch("d41d8cd98f00b204e9800998ecf8427e+0"), Equals, false)
	c.Assert(UUIDMatch(""), Equals, false)
}

func (s *UnitSuite) TestPDHMatch(c *C) {
	c.Assert(PDHMatch("zzzzz-tpzed-000000000000000"), Equals, false)
	c.Assert(PDHMatch("d41d8cd98f00b204e9800998ecf8427e"), Equals, false)
	c.Assert(PDHMatch("d41d8cd98f00b204e9800998ecf8427e+0"), Equals, true)
	c.Assert(PDHMatch("d41d8cd98f00b204e9800998ecf8427e+12345"), Equals, true)
	c.Assert(PDHMatch("d41d8cd98f00b204e9800998ecf8427e 12345"), Equals, false)
	c.Assert(PDHMatch("D41D8CD98F00B204E9800998ECF8427E+12345"), Equals, false)
	c.Assert(PDHMatch("d41d8cd98f00b204e9800998ecf8427e+12345 "), Equals, false)
	c.Assert(PDHMatch("d41d8cd98f00b204e9800998ecf8427e+abcdef"), Equals, false)
	c.Assert(PDHMatch("da39a3ee5e6b4b0d3255bfef95601890afd80709"), Equals, false)
	c.Assert(PDHMatch("da39a3ee5e6b4b0d3255bfef95601890afd80709+0"), Equals, false)
	c.Assert(PDHMatch("d41d8cd98f00b204e9800998ecf8427+12345"), Equals, false)
	c.Assert(PDHMatch("d41d8cd98f00b204e9800998ecf8427e+12345\n"), Equals, false)
	c.Assert(PDHMatch("+12345"), Equals, false)
	c.Assert(PDHMatch(""), Equals, false)
}
