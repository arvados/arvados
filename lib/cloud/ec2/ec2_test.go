// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0
//
//
// How to manually run individual tests against the real cloud:
//
// $ go test -v git.arvados.org/arvados.git/lib/cloud/ec2 -live-ec2-cfg ec2config.yml -check.f=TestCreate
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
//       SecurityGroupIDs: [sg-xxxxxxxx]
//       SubnetID: subnet-xxxxxxxx
//       AdminUsername: crunch

package ec2

import (
	"encoding/json"
	"flag"
	"testing"

	"git.arvados.org/arvados.git/lib/cloud"
	"git.arvados.org/arvados.git/lib/dispatchcloud/test"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/config"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/sirupsen/logrus"
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

type ec2stub struct {
}

func (e *ec2stub) ImportKeyPair(input *ec2.ImportKeyPairInput) (*ec2.ImportKeyPairOutput, error) {
	return nil, nil
}

func (e *ec2stub) DescribeKeyPairs(input *ec2.DescribeKeyPairsInput) (*ec2.DescribeKeyPairsOutput, error) {
	return &ec2.DescribeKeyPairsOutput{}, nil
}

func (e *ec2stub) RunInstances(input *ec2.RunInstancesInput) (*ec2.Reservation, error) {
	return &ec2.Reservation{Instances: []*ec2.Instance{{
		InstanceId: aws.String("i-123"),
		Tags:       input.TagSpecifications[0].Tags,
	}}}, nil
}

func (e *ec2stub) DescribeInstances(input *ec2.DescribeInstancesInput) (*ec2.DescribeInstancesOutput, error) {
	return &ec2.DescribeInstancesOutput{}, nil
}

func (e *ec2stub) CreateTags(input *ec2.CreateTagsInput) (*ec2.CreateTagsOutput, error) {
	return nil, nil
}

func (e *ec2stub) TerminateInstances(input *ec2.TerminateInstancesInput) (*ec2.TerminateInstancesOutput, error) {
	return nil, nil
}

func GetInstanceSet() (cloud.InstanceSet, cloud.ImageID, arvados.Cluster, error) {
	cluster := arvados.Cluster{
		InstanceTypes: arvados.InstanceTypeMap(map[string]arvados.InstanceType{
			"tiny": {
				Name:         "tiny",
				ProviderType: "t2.micro",
				VCPUs:        1,
				RAM:          4000000000,
				Scratch:      10000000000,
				Price:        .02,
				Preemptible:  false,
			},
			"tiny-with-extra-scratch": {
				Name:         "tiny",
				ProviderType: "t2.micro",
				VCPUs:        1,
				RAM:          4000000000,
				Price:        .02,
				Preemptible:  false,
				AddedScratch: 20000000000,
			},
			"tiny-preemptible": {
				Name:         "tiny",
				ProviderType: "t2.micro",
				VCPUs:        1,
				RAM:          4000000000,
				Scratch:      10000000000,
				Price:        .02,
				Preemptible:  true,
			},
		})}
	if *live != "" {
		var exampleCfg testConfig
		err := config.LoadFile(&exampleCfg, *live)
		if err != nil {
			return nil, cloud.ImageID(""), cluster, err
		}

		ap, err := newEC2InstanceSet(exampleCfg.DriverParameters, "test123", nil, logrus.StandardLogger())
		return ap, cloud.ImageID(exampleCfg.ImageIDForTestSuite), cluster, err
	}
	ap := ec2InstanceSet{
		ec2config:     ec2InstanceSetConfig{},
		instanceSetID: "test123",
		logger:        logrus.StandardLogger(),
		client:        &ec2stub{},
		keys:          make(map[string]string),
	}
	return &ap, cloud.ImageID("blob"), cluster, nil
}

func (*EC2InstanceSetSuite) TestCreate(c *check.C) {
	ap, img, cluster, err := GetInstanceSet()
	if err != nil {
		c.Fatal("Error making provider", err)
	}

	pk, _ := test.LoadTestKey(c, "../../dispatchcloud/test/sshkey_dispatch")
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

func (*EC2InstanceSetSuite) TestCreateWithExtraScratch(c *check.C) {
	ap, img, cluster, err := GetInstanceSet()
	if err != nil {
		c.Fatal("Error making provider", err)
	}

	pk, _ := test.LoadTestKey(c, "../../dispatchcloud/test/sshkey_dispatch")
	c.Assert(err, check.IsNil)

	inst, err := ap.Create(cluster.InstanceTypes["tiny-with-extra-scratch"],
		img, map[string]string{
			"TestTagName": "test tag value",
		}, "umask 0600; echo -n test-file-data >/var/run/test-file", pk)

	c.Assert(err, check.IsNil)

	tags := inst.Tags()
	c.Check(tags["TestTagName"], check.Equals, "test tag value")
	c.Logf("inst.String()=%v Address()=%v Tags()=%v", inst.String(), inst.Address(), tags)

}

func (*EC2InstanceSetSuite) TestCreatePreemptible(c *check.C) {
	ap, img, cluster, err := GetInstanceSet()
	if err != nil {
		c.Fatal("Error making provider", err)
	}

	pk, _ := test.LoadTestKey(c, "../../dispatchcloud/test/sshkey_dispatch")
	c.Assert(err, check.IsNil)

	inst, err := ap.Create(cluster.InstanceTypes["tiny-preemptible"],
		img, map[string]string{
			"TestTagName": "test tag value",
		}, "umask 0600; echo -n test-file-data >/var/run/test-file", pk)

	c.Assert(err, check.IsNil)

	tags := inst.Tags()
	c.Check(tags["TestTagName"], check.Equals, "test tag value")
	c.Logf("inst.String()=%v Address()=%v Tags()=%v", inst.String(), inst.Address(), tags)

}

func (*EC2InstanceSetSuite) TestTagInstances(c *check.C) {
	ap, _, _, err := GetInstanceSet()
	if err != nil {
		c.Fatal("Error making provider", err)
	}

	l, err := ap.Instances(nil)
	c.Assert(err, check.IsNil)

	for _, i := range l {
		tg := i.Tags()
		tg["TestTag2"] = "123 test tag 2"
		c.Check(i.SetTags(tg), check.IsNil)
	}
}

func (*EC2InstanceSetSuite) TestListInstances(c *check.C) {
	ap, _, _, err := GetInstanceSet()
	if err != nil {
		c.Fatal("Error making provider: ", err)
	}

	l, err := ap.Instances(nil)

	c.Assert(err, check.IsNil)

	for _, i := range l {
		tg := i.Tags()
		c.Logf("%v %v %v", i.String(), i.Address(), tg)
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

var TestRateLimitErrorInterface cloud.RateLimitError = rateLimitError{}
