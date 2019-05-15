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
    arvados-controller:
     listen: ":9004"
   `+hostname+`:
    arvados-api-server:
     listen: ":8000"
   dispatch-host:
    arvados-dispatch-cloud:
     listen: ":9006"
`, `
Clusters:
 z1111:
  Services:
   RailsAPI:
    InternalURLs:
     "http://localhost:8000": {}
   Controller:
    InternalURLs:
     "http://localhost:9004": {}
   DispatchCloud:
    InternalURLs:
     "http://dispatch-host:9006": {}
  NodeProfiles:
   "*":
    arvados-controller:
     listen: ":9004"
   `+hostname+`:
    arvados-api-server:
     listen: ":8000"
   dispatch-host:
    arvados-dispatch-cloud:
     listen: ":9006"
`)
}
