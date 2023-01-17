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
// Example ec2config.yml:
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
	"sync/atomic"
	"testing"
	"time"

	"git.arvados.org/arvados.git/lib/cloud"
	"git.arvados.org/arvados.git/lib/dispatchcloud/test"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/config"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
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
	c       *check.C
	reftime time.Time
}

func (e *ec2stub) ImportKeyPair(input *ec2.ImportKeyPairInput) (*ec2.ImportKeyPairOutput, error) {
	return nil, nil
}

func (e *ec2stub) DescribeKeyPairs(input *ec2.DescribeKeyPairsInput) (*ec2.DescribeKeyPairsOutput, error) {
	return &ec2.DescribeKeyPairsOutput{}, nil
}

func (e *ec2stub) RunInstances(input *ec2.RunInstancesInput) (*ec2.Reservation, error) {
	return &ec2.Reservation{Instances: []*ec2.Instance{{
		InstanceId:   aws.String("i-123"),
		InstanceType: aws.String("t2.micro"),
		Tags:         input.TagSpecifications[0].Tags,
	}}}, nil
}

func (e *ec2stub) DescribeInstances(input *ec2.DescribeInstancesInput) (*ec2.DescribeInstancesOutput, error) {
	return &ec2.DescribeInstancesOutput{
		Reservations: []*ec2.Reservation{{
			Instances: []*ec2.Instance{{
				InstanceId:        aws.String("i-123"),
				InstanceLifecycle: aws.String("spot"),
				InstanceType:      aws.String("t2.micro"),
				PrivateIpAddress:  aws.String("10.1.2.3"),
				State:             &ec2.InstanceState{Name: aws.String("running")},
			}, {
				InstanceId:        aws.String("i-124"),
				InstanceLifecycle: aws.String("spot"),
				InstanceType:      aws.String("t2.micro"),
				PrivateIpAddress:  aws.String("10.1.2.4"),
				State:             &ec2.InstanceState{Name: aws.String("running")},
			}},
		}},
	}, nil
}

func (e *ec2stub) DescribeInstanceStatusPages(input *ec2.DescribeInstanceStatusInput, fn func(*ec2.DescribeInstanceStatusOutput, bool) bool) error {
	fn(&ec2.DescribeInstanceStatusOutput{
		InstanceStatuses: []*ec2.InstanceStatus{{
			InstanceId:       aws.String("i-123"),
			AvailabilityZone: aws.String("aa-east-1a"),
		}, {
			InstanceId:       aws.String("i-124"),
			AvailabilityZone: aws.String("aa-east-1a"),
		}},
	}, true)
	return nil
}

func (e *ec2stub) DescribeSpotPriceHistoryPages(input *ec2.DescribeSpotPriceHistoryInput, fn func(*ec2.DescribeSpotPriceHistoryOutput, bool) bool) error {
	if !fn(&ec2.DescribeSpotPriceHistoryOutput{
		SpotPriceHistory: []*ec2.SpotPrice{
			&ec2.SpotPrice{
				InstanceType:     aws.String("t2.micro"),
				AvailabilityZone: aws.String("aa-east-1a"),
				SpotPrice:        aws.String("0.005"),
				Timestamp:        aws.Time(e.reftime.Add(-9 * time.Minute)),
			},
			&ec2.SpotPrice{
				InstanceType:     aws.String("t2.micro"),
				AvailabilityZone: aws.String("aa-east-1a"),
				SpotPrice:        aws.String("0.015"),
				Timestamp:        aws.Time(e.reftime.Add(-5 * time.Minute)),
			},
		},
	}, false) {
		return nil
	}
	fn(&ec2.DescribeSpotPriceHistoryOutput{
		SpotPriceHistory: []*ec2.SpotPrice{
			&ec2.SpotPrice{
				InstanceType:     aws.String("t2.micro"),
				AvailabilityZone: aws.String("aa-east-1a"),
				SpotPrice:        aws.String("0.01"),
				Timestamp:        aws.Time(e.reftime.Add(-2 * time.Minute)),
			},
		},
	}, true)
	return nil
}

func (e *ec2stub) CreateTags(input *ec2.CreateTagsInput) (*ec2.CreateTagsOutput, error) {
	return nil, nil
}

func (e *ec2stub) TerminateInstances(input *ec2.TerminateInstancesInput) (*ec2.TerminateInstancesOutput, error) {
	return nil, nil
}

func GetInstanceSet(c *check.C) (cloud.InstanceSet, cloud.ImageID, arvados.Cluster) {
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
				Name:         "tiny-with-extra-scratch",
				ProviderType: "t2.micro",
				VCPUs:        1,
				RAM:          4000000000,
				Price:        .02,
				Preemptible:  false,
				AddedScratch: 20000000000,
			},
			"tiny-preemptible": {
				Name:         "tiny-preemptible",
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
		c.Assert(err, check.IsNil)

		ap, err := newEC2InstanceSet(exampleCfg.DriverParameters, "test123", nil, logrus.StandardLogger())
		c.Assert(err, check.IsNil)
		return ap, cloud.ImageID(exampleCfg.ImageIDForTestSuite), cluster
	}
	ap := ec2InstanceSet{
		ec2config:     ec2InstanceSetConfig{},
		instanceSetID: "test123",
		logger:        logrus.StandardLogger(),
		client:        &ec2stub{c: c, reftime: time.Now().UTC()},
		keys:          make(map[string]string),
	}
	return &ap, cloud.ImageID("blob"), cluster
}

func (*EC2InstanceSetSuite) TestCreate(c *check.C) {
	ap, img, cluster := GetInstanceSet(c)
	pk, _ := test.LoadTestKey(c, "../../dispatchcloud/test/sshkey_dispatch")

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
	ap, img, cluster := GetInstanceSet(c)
	pk, _ := test.LoadTestKey(c, "../../dispatchcloud/test/sshkey_dispatch")

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
	ap, img, cluster := GetInstanceSet(c)
	pk, _ := test.LoadTestKey(c, "../../dispatchcloud/test/sshkey_dispatch")

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
	ap, _, _ := GetInstanceSet(c)
	l, err := ap.Instances(nil)
	c.Assert(err, check.IsNil)

	for _, i := range l {
		tg := i.Tags()
		tg["TestTag2"] = "123 test tag 2"
		c.Check(i.SetTags(tg), check.IsNil)
	}
}

func (*EC2InstanceSetSuite) TestListInstances(c *check.C) {
	ap, _, _ := GetInstanceSet(c)
	l, err := ap.Instances(nil)
	c.Assert(err, check.IsNil)

	for _, i := range l {
		tg := i.Tags()
		c.Logf("%v %v %v", i.String(), i.Address(), tg)
	}
}

func (*EC2InstanceSetSuite) TestDestroyInstances(c *check.C) {
	ap, _, _ := GetInstanceSet(c)
	l, err := ap.Instances(nil)
	c.Assert(err, check.IsNil)

	for _, i := range l {
		c.Check(i.Destroy(), check.IsNil)
	}
}

func (*EC2InstanceSetSuite) TestInstancePriceHistory(c *check.C) {
	ap, img, cluster := GetInstanceSet(c)
	pk, _ := test.LoadTestKey(c, "../../dispatchcloud/test/sshkey_dispatch")
	tags := cloud.InstanceTags{"arvados-ec2-driver": "test"}
	inst1, err := ap.Create(cluster.InstanceTypes["tiny-preemptible"], img, tags, "true", pk)
	c.Assert(err, check.IsNil)
	defer inst1.Destroy()
	inst2, err := ap.Create(cluster.InstanceTypes["tiny-preemptible"], img, tags, "true", pk)
	c.Assert(err, check.IsNil)
	defer inst2.Destroy()

	// in live mode, we need to wait for the instances to reach
	// running state before we can discover their availability
	// zones and look up the appropriate prices.
	var instances []cloud.Instance
	for deadline := time.Now().Add(5 * time.Minute); ; {
		if deadline.Before(time.Now()) {
			c.Fatal("timed out")
		}
		instances, err = ap.Instances(tags)
		running := 0
		for _, inst := range instances {
			if inst.Address() != "" {
				running++
			}
		}
		if running >= 2 {
			break
		}
		time.Sleep(10 * time.Second)
	}

	for _, inst := range instances {
		hist := inst.PriceHistory()
		c.Logf("%s price history: %v", inst.ID(), hist)
		c.Check(len(hist) > 0, check.Equals, true)
		for i, ip := range hist {
			c.Check(ip.Price, check.Not(check.Equals), 0.0)
			if i > 0 {
				c.Check(ip.StartTime.Before(hist[i-1].StartTime), check.Equals, true)
			}
		}
	}
}

func (*EC2InstanceSetSuite) TestWrapError(c *check.C) {
	retryError := awserr.New("Throttling", "", nil)
	wrapped := wrapError(retryError, &atomic.Value{})
	_, ok := wrapped.(cloud.RateLimitError)
	c.Check(ok, check.Equals, true)

	quotaError := awserr.New("InsufficientInstanceCapacity", "", nil)
	wrapped = wrapError(quotaError, nil)
	_, ok = wrapped.(cloud.QuotaError)
	c.Check(ok, check.Equals, true)
}
