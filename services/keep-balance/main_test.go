package main

import (
	"encoding/json"
	"time"

	check "gopkg.in/check.v1"
)

var _ = check.Suite(&mainSuite{})

type mainSuite struct{}

func (s *mainSuite) TestExampleJSON(c *check.C) {
	var config Config
	c.Check(json.Unmarshal(exampleConfigFile, &config), check.IsNil)
	c.Check(config.KeepServiceTypes, check.DeepEquals, []string{"disk"})
	c.Check(config.Client.AuthToken, check.Equals, "xyzzy")
	c.Check(time.Duration(config.RunPeriod), check.Equals, 600*time.Second)
}

func (s *mainSuite) TestConfigJSONWithKeepServiceList(c *check.C) {
	var config Config
	c.Check(json.Unmarshal([]byte(`
		{
		    "Client": {
			"APIHost": "zzzzz.arvadosapi.com:443",
			"AuthToken": "xyzzy",
			"Insecure": false
		    },
		    "KeepServiceList": {
			"items": [
			    {"uuid":"zzzzz-bi64l-abcdefghijklmno", "service_type":"disk", "service_host":"a.zzzzz.arvadosapi.com", "service_port":12345},
			    {"uuid":"zzzzz-bi64l-bcdefghijklmnop", "service_type":"blob", "service_host":"b.zzzzz.arvadosapi.com", "service_port":12345}
			]
		    },
		    "RunPeriod": "600s"
		}`), &config), check.IsNil)
	c.Assert(len(config.KeepServiceList.Items), check.Equals, 2)
	c.Check(config.KeepServiceList.Items[0].UUID, check.Equals, "zzzzz-bi64l-abcdefghijklmno")
	c.Check(config.KeepServiceList.Items[0].ServicePort, check.Equals, 12345)
	c.Check(config.Client.AuthToken, check.Equals, "xyzzy")
}
