// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0
//
//
// How to manually run individual tests against the real cloud:
//
// $ go test -v git.curoverse.com/arvados.git/lib/cloud/ec2 -live-ec2-cfg ec2config.yml -check.f=TestCreate
//
// Tests should be run individually and in the order they are listed in the file:
//
// Example azconfig.yml:
//
// ImageIDForTestSuite: ami-xxxxxxxxxxxxxxxxx
// DriverParameters:
//       AccessKeyID: XXXXXXXXXXXXXX
//       SecretAccessKey: xxxxxxxxxxxxxxxxxxxx
//       Region: us-east-1
//       SecurityGroupId: sg-xxxxxxxx
//       SubnetId: subnet-xxxxxxxx
//       AdminUsername: crunch
//       KeyPairName: arvados-dispatcher-keypair

package ec2

import (
	"encoding/json"
	"flag"
	"log"
	"testing"

	"git.curoverse.com/arvados.git/lib/cloud"
	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/config"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
	check "gopkg.in/check.v1"
)

var live = flag.String("live-ec2-cfg", "", "Test with real EC2 API, provide config file")

// Gocheck boilerplate
func Test(t *testing.T) {
	check.TestingT(t)
}

type EC2InstanceSetSuite struct{}

var _ = check.Suite(&EC2InstanceSetSuite{})

type testConfig struct {
	ImageIDForTestSuite string
	DriverParameters    json.RawMessage
}

func GetInstanceSet() (cloud.InstanceSet, cloud.ImageID, arvados.Cluster, error) {
	cluster := arvados.Cluster{
		InstanceTypes: arvados.InstanceTypeMap(map[string]arvados.InstanceType{
			"tiny": arvados.InstanceType{
				Name:         "tiny",
				ProviderType: "m1.small",
				VCPUs:        1,
				RAM:          4000000000,
				Scratch:      10000000000,
				Price:        .02,
				Preemptible:  false,
			},
		})}
	if *live != "" {
		var exampleCfg testConfig
		err := config.LoadFile(&exampleCfg, *live)
		if err != nil {
			return nil, cloud.ImageID(""), cluster, err
		}

		ap, err := newEC2InstanceSet(exampleCfg.DriverParameters, "test123", logrus.StandardLogger())
		return ap, cloud.ImageID(exampleCfg.ImageIDForTestSuite), cluster, err
	}
	ap := ec2InstanceSet{
		ec2config:    ec2InstanceSetConfig{},
		dispatcherID: "test123",
		logger:       logrus.StandardLogger(),
	}
	return &ap, cloud.ImageID("blob"), cluster, nil
}

var testKey = []byte(`ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDLQS1ExT2+WjA0d/hntEAyAtgeN1W2ik2QX8c2zO6HjlPHWXL92r07W0WMuDib40Pcevpi1BXeBWXA9ZB5KKMJB+ukaAu22KklnQuUmNvk6ZXnPKSkGxuCYvPQb08WhHf3p1VxiKfP3iauedBDM4x9/bkJohlBBQiFXzNUcQ+a6rKiMzmJN2gbL8ncyUzc+XQ5q4JndTwTGtOlzDiGOc9O4z5Dd76wtAVJneOuuNpwfFRVHThpJM6VThpCZOnl8APaceWXKeuwOuCae3COZMz++xQfxOfZ9Z8aIwo+TlQhsRaNfZ4Vjrop6ej8dtfZtgUFKfbXEOYaHrGrWGotFDTD example@example`)

func (*EC2InstanceSetSuite) TestCreate(c *check.C) {
	ap, img, cluster, err := GetInstanceSet()
	if err != nil {
		c.Fatal("Error making provider", err)
	}

	pk, _, _, _, err := ssh.ParseAuthorizedKey(testKey)
	c.Assert(err, check.IsNil)

	inst, err := ap.Create(cluster.InstanceTypes["tiny"],
		img, map[string]string{
			"TestTagName": "test tag value",
		}, "umask 0600; echo -n test-file-data >/var/run/test-file", pk)

	c.Assert(err, check.IsNil)

	tags := inst.Tags()
	c.Check(tags["TestTagName"], check.Equals, "test tag value")
	c.Logf("inst.String()=%v Address()=%v Tags()=%v", inst.String(), inst.Address(), tags)

}

func (*EC2InstanceSetSuite) TestListInstances(c *check.C) {
	ap, _, _, err := GetInstanceSet()
	if err != nil {
		c.Fatal("Error making provider", err)
	}

	l, err := ap.Instances(nil)

	c.Assert(err, check.IsNil)

	for _, i := range l {
		tg := i.Tags()
		log.Printf("%v %v %v", i.String(), i.Address(), tg)
	}
}

func (*EC2InstanceSetSuite) TestDestroyInstances(c *check.C) {
	ap, _, _, err := GetInstanceSet()
	if err != nil {
		c.Fatal("Error making provider", err)
	}

	l, err := ap.Instances(nil)
	c.Assert(err, check.IsNil)

	for _, i := range l {
		c.Check(i.Destroy(), check.IsNil)
	}
}
