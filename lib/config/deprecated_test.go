// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package config

import (
	"os"

	check "gopkg.in/check.v1"
)

func (s *LoadSuite) TestDeprecatedNodeProfilesToServices(c *check.C) {
	hostname, err := os.Hostname()
	c.Assert(err, check.IsNil)
	s.checkEquivalent(c, `
Clusters:
 z1111:
  NodeProfiles:
   "*":
    arvados-dispatch-cloud:
     listen: ":9006"
    arvados-controller:
     listen: ":9004"
   `+hostname+`:
    arvados-api-server:
     listen: ":8000"
`, `
Clusters:
 z1111:
  Services:
   RailsAPI:
    InternalURLs:
     "http://`+hostname+`:8000": {}
   Controller:
    InternalURLs:
     "http://`+hostname+`:9004": {}
   DispatchCloud:
    InternalURLs:
     "http://`+hostname+`:9006": {}
  NodeProfiles:
   "*":
    arvados-dispatch-cloud:
     listen: ":9006"
    arvados-controller:
     listen: ":9004"
   `+hostname+`:
    arvados-api-server:
     listen: ":8000"
`)
}
