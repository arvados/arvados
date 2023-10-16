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
	"errors"
	"flag"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"git.arvados.org/arvados.git/lib/cloud"
	"git.arvados.org/arvados.git/lib/dispatchcloud/test"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"git.arvados.org/arvados.git/sdk/go/config"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/ghodss/yaml"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	check "gopkg.in/check.v1"
)

var live = flag.String("live-ec2-cfg", "", "Test with real EC2 API, provide config file")

// Gocheck boilerplate
func Test(t *testing.T) {
	check.TestingT(t)
}

type sliceOrStringSuite struct{}

var _ = check.Suite(&sliceOrStringSuite{})

func (s *sliceOrStringSuite) TestUnmarshal(c *check.C) {
	var conf ec2InstanceSetConfig
	for _, trial := range []struct {
		input  string
		output sliceOrSingleString
	}{
		{``, nil},
		{`""`, nil},
		{`[]`, nil},
		{`"foo"`, sliceOrSingleString{"foo"}},
		{`["foo"]`, sliceOrSingleString{"foo"}},
		{`[foo]`, sliceOrSingleString{"foo"}},
		{`["foo", "bar"]`, sliceOrSingleString{"foo", "bar"}},
		{`[foo-bar, baz]`, sliceOrSingleString{"foo-bar", "baz"}},
	} {
		c.Logf("trial: %+v", trial)
		err := yaml.Unmarshal([]byte("SubnetID: "+trial.input+"\n"), &conf)
		if !c.Check(err, check.IsNil) {
			continue
		}
		c.Check(conf.SubnetID, check.DeepEquals, trial.output)
	}
}

type EC2InstanceSetSuite struct{}

var _ = check.Suite(&EC2InstanceSetSuite{})

type testConfig struct {
	ImageIDForTestSuite string
	DriverParameters    json.RawMessage
}

type ec2stub struct {
	c                     *check.C
	reftime               time.Time
	importKeyPairCalls    []*ec2.ImportKeyPairInput
	describeKeyPairsCalls []*ec2.DescribeKeyPairsInput
	runInstancesCalls     []*ec2.RunInstancesInput
	// {subnetID => error}: RunInstances returns error if subnetID
	// matches.
	subnetErrorOnRunInstances map[string]error
}

func (e *ec2stub) ImportKeyPair(input *ec2.ImportKeyPairInput) (*ec2.ImportKeyPairOutput, error) {
	e.importKeyPairCalls = append(e.importKeyPairCalls, input)
	return nil, nil
}

func (e *ec2stub) DescribeKeyPairs(input *ec2.DescribeKeyPairsInput) (*ec2.DescribeKeyPairsOutput, error) {
	e.describeKeyPairsCalls = append(e.describeKeyPairsCalls, input)
	return &ec2.DescribeKeyPairsOutput{}, nil
}

func (e *ec2stub) RunInstances(input *ec2.RunInstancesInput) (*ec2.Reservation, error) {
	e.runInstancesCalls = append(e.runInstancesCalls, input)
	if len(input.NetworkInterfaces) > 0 && input.NetworkInterfaces[0].SubnetId != nil {
		err := e.subnetErrorOnRunInstances[*input.NetworkInterfaces[0].SubnetId]
		if err != nil {
			return nil, err
		}
	}
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
				State:             &ec2.InstanceState{Name: aws.String("running"), Code: aws.Int64(16)},
			}, {
				InstanceId:        aws.String("i-124"),
				InstanceLifecycle: aws.String("spot"),
				InstanceType:      aws.String("t2.micro"),
				PrivateIpAddress:  aws.String("10.1.2.4"),
				State:             &ec2.InstanceState{Name: aws.String("running"), Code: aws.Int64(16)},
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

type ec2stubError struct {
	code    string
	message string
}

func (err *ec2stubError) Code() string    { return err.code }
func (err *ec2stubError) Message() string { return err.message }
func (err *ec2stubError) Error() string   { return fmt.Sprintf("%s: %s", err.code, err.message) }
func (err *ec2stubError) OrigErr() error  { return errors.New("stub OrigErr") }

// Ensure ec2stubError satisfies the aws.Error interface
var _ = awserr.Error(&ec2stubError{})

func GetInstanceSet(c *check.C, conf string) (*ec2InstanceSet, cloud.ImageID, arvados.Cluster, *prometheus.Registry) {
	reg := prometheus.NewRegistry()
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

		is, err := newEC2InstanceSet(exampleCfg.DriverParameters, "test123", nil, logrus.StandardLogger(), reg)
		c.Assert(err, check.IsNil)
		return is.(*ec2InstanceSet), cloud.ImageID(exampleCfg.ImageIDForTestSuite), cluster, reg
	} else {
		is, err := newEC2InstanceSet(json.RawMessage(conf), "test123", nil, ctxlog.TestLogger(c), reg)
		c.Assert(err, check.IsNil)
		is.(*ec2InstanceSet).client = &ec2stub{c: c, reftime: time.Now().UTC()}
		return is.(*ec2InstanceSet), cloud.ImageID("blob"), cluster, reg
	}
}

func (*EC2InstanceSetSuite) TestCreate(c *check.C) {
	ap, img, cluster, _ := GetInstanceSet(c, "{}")
	pk, _ := test.LoadTestKey(c, "../../dispatchcloud/test/sshkey_dispatch")

	inst, err := ap.Create(cluster.InstanceTypes["tiny"],
		img, map[string]string{
			"TestTagName": "test tag value",
		}, "umask 0600; echo -n test-file-data >/var/run/test-file", pk)
	c.Assert(err, check.IsNil)

	tags := inst.Tags()
	c.Check(tags["TestTagName"], check.Equals, "test tag value")
	c.Logf("inst.String()=%v Address()=%v Tags()=%v", inst.String(), inst.Address(), tags)

	if *live == "" {
		c.Check(ap.client.(*ec2stub).describeKeyPairsCalls, check.HasLen, 1)
		c.Check(ap.client.(*ec2stub).importKeyPairCalls, check.HasLen, 1)
	}
}

func (*EC2InstanceSetSuite) TestCreateWithExtraScratch(c *check.C) {
	ap, img, cluster, _ := GetInstanceSet(c, "{}")
	inst, err := ap.Create(cluster.InstanceTypes["tiny-with-extra-scratch"],
		img, map[string]string{
			"TestTagName": "test tag value",
		}, "umask 0600; echo -n test-file-data >/var/run/test-file", nil)

	c.Assert(err, check.IsNil)

	tags := inst.Tags()
	c.Check(tags["TestTagName"], check.Equals, "test tag value")
	c.Logf("inst.String()=%v Address()=%v Tags()=%v", inst.String(), inst.Address(), tags)

	if *live == "" {
		// Should not have called key pair APIs, because
		// publickey arg was nil
		c.Check(ap.client.(*ec2stub).describeKeyPairsCalls, check.HasLen, 0)
		c.Check(ap.client.(*ec2stub).importKeyPairCalls, check.HasLen, 0)
	}
}

func (*EC2InstanceSetSuite) TestCreatePreemptible(c *check.C) {
	ap, img, cluster, _ := GetInstanceSet(c, "{}")
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

func (*EC2InstanceSetSuite) TestCreateFailoverSecondSubnet(c *check.C) {
	if *live != "" {
		c.Skip("not applicable in live mode")
		return
	}

	ap, img, cluster, reg := GetInstanceSet(c, `{"SubnetID":["subnet-full","subnet-good"]}`)
	ap.client.(*ec2stub).subnetErrorOnRunInstances = map[string]error{
		"subnet-full": &ec2stubError{
			code:    "InsufficientFreeAddressesInSubnet",
			message: "subnet is full",
		},
	}
	inst, err := ap.Create(cluster.InstanceTypes["tiny"], img, nil, "", nil)
	c.Check(err, check.IsNil)
	c.Check(inst, check.NotNil)
	c.Check(ap.client.(*ec2stub).runInstancesCalls, check.HasLen, 2)
	metrics := arvadostest.GatherMetricsAsString(reg)
	c.Check(metrics, check.Matches, `(?ms).*`+
		`arvados_dispatchcloud_ec2_instance_starts_total{subnet_id="subnet-full",success="0"} 1\n`+
		`arvados_dispatchcloud_ec2_instance_starts_total{subnet_id="subnet-full",success="1"} 0\n`+
		`arvados_dispatchcloud_ec2_instance_starts_total{subnet_id="subnet-good",success="0"} 0\n`+
		`arvados_dispatchcloud_ec2_instance_starts_total{subnet_id="subnet-good",success="1"} 1\n`+
		`.*`)

	// Next RunInstances call should try the working subnet first
	inst, err = ap.Create(cluster.InstanceTypes["tiny"], img, nil, "", nil)
	c.Check(err, check.IsNil)
	c.Check(inst, check.NotNil)
	c.Check(ap.client.(*ec2stub).runInstancesCalls, check.HasLen, 3)
	metrics = arvadostest.GatherMetricsAsString(reg)
	c.Check(metrics, check.Matches, `(?ms).*`+
		`arvados_dispatchcloud_ec2_instance_starts_total{subnet_id="subnet-full",success="0"} 1\n`+
		`arvados_dispatchcloud_ec2_instance_starts_total{subnet_id="subnet-full",success="1"} 0\n`+
		`arvados_dispatchcloud_ec2_instance_starts_total{subnet_id="subnet-good",success="0"} 0\n`+
		`arvados_dispatchcloud_ec2_instance_starts_total{subnet_id="subnet-good",success="1"} 2\n`+
		`.*`)
}

func (*EC2InstanceSetSuite) TestCreateAllSubnetsFailing(c *check.C) {
	if *live != "" {
		c.Skip("not applicable in live mode")
		return
	}

	ap, img, cluster, reg := GetInstanceSet(c, `{"SubnetID":["subnet-full","subnet-broken"]}`)
	ap.client.(*ec2stub).subnetErrorOnRunInstances = map[string]error{
		"subnet-full": &ec2stubError{
			code:    "InsufficientFreeAddressesInSubnet",
			message: "subnet is full",
		},
		"subnet-broken": &ec2stubError{
			code:    "InvalidSubnetId.NotFound",
			message: "bogus subnet id",
		},
	}
	_, err := ap.Create(cluster.InstanceTypes["tiny"], img, nil, "", nil)
	c.Check(err, check.NotNil)
	c.Check(err, check.ErrorMatches, `.*InvalidSubnetId\.NotFound.*`)
	c.Check(ap.client.(*ec2stub).runInstancesCalls, check.HasLen, 2)
	metrics := arvadostest.GatherMetricsAsString(reg)
	c.Check(metrics, check.Matches, `(?ms).*`+
		`arvados_dispatchcloud_ec2_instance_starts_total{subnet_id="subnet-broken",success="0"} 1\n`+
		`arvados_dispatchcloud_ec2_instance_starts_total{subnet_id="subnet-broken",success="1"} 0\n`+
		`arvados_dispatchcloud_ec2_instance_starts_total{subnet_id="subnet-full",success="0"} 1\n`+
		`arvados_dispatchcloud_ec2_instance_starts_total{subnet_id="subnet-full",success="1"} 0\n`+
		`.*`)

	_, err = ap.Create(cluster.InstanceTypes["tiny"], img, nil, "", nil)
	c.Check(err, check.NotNil)
	c.Check(err, check.ErrorMatches, `.*InsufficientFreeAddressesInSubnet.*`)
	c.Check(ap.client.(*ec2stub).runInstancesCalls, check.HasLen, 4)
	metrics = arvadostest.GatherMetricsAsString(reg)
	c.Check(metrics, check.Matches, `(?ms).*`+
		`arvados_dispatchcloud_ec2_instance_starts_total{subnet_id="subnet-broken",success="0"} 2\n`+
		`arvados_dispatchcloud_ec2_instance_starts_total{subnet_id="subnet-broken",success="1"} 0\n`+
		`arvados_dispatchcloud_ec2_instance_starts_total{subnet_id="subnet-full",success="0"} 2\n`+
		`arvados_dispatchcloud_ec2_instance_starts_total{subnet_id="subnet-full",success="1"} 0\n`+
		`.*`)
}

func (*EC2InstanceSetSuite) TestTagInstances(c *check.C) {
	ap, _, _, _ := GetInstanceSet(c, "{}")
	l, err := ap.Instances(nil)
	c.Assert(err, check.IsNil)

	for _, i := range l {
		tg := i.Tags()
		tg["TestTag2"] = "123 test tag 2"
		c.Check(i.SetTags(tg), check.IsNil)
	}
}

func (*EC2InstanceSetSuite) TestListInstances(c *check.C) {
	ap, _, _, reg := GetInstanceSet(c, "{}")
	l, err := ap.Instances(nil)
	c.Assert(err, check.IsNil)

	for _, i := range l {
		tg := i.Tags()
		c.Logf("%v %v %v", i.String(), i.Address(), tg)
	}

	metrics := arvadostest.GatherMetricsAsString(reg)
	c.Check(metrics, check.Matches, `(?ms).*`+
		`arvados_dispatchcloud_ec2_instances{subnet_id="[^"]*"} \d+\n`+
		`.*`)
}

func (*EC2InstanceSetSuite) TestDestroyInstances(c *check.C) {
	ap, _, _, _ := GetInstanceSet(c, "{}")
	l, err := ap.Instances(nil)
	c.Assert(err, check.IsNil)

	for _, i := range l {
		c.Check(i.Destroy(), check.IsNil)
	}
}

func (*EC2InstanceSetSuite) TestInstancePriceHistory(c *check.C) {
	ap, img, cluster, _ := GetInstanceSet(c, "{}")
	pk, _ := test.LoadTestKey(c, "../../dispatchcloud/test/sshkey_dispatch")
	tags := cloud.InstanceTags{"arvados-ec2-driver": "test"}

	defer func() {
		instances, err := ap.Instances(tags)
		c.Assert(err, check.IsNil)
		for _, inst := range instances {
			c.Logf("cleanup: destroy instance %s", inst)
			c.Check(inst.Destroy(), check.IsNil)
		}
	}()

	ap.ec2config.SpotPriceUpdateInterval = arvados.Duration(time.Hour)
	ap.ec2config.EBSPrice = 0.1 // $/GiB/month
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
			ec2i := inst.(*ec2Instance).instance
			if *ec2i.InstanceLifecycle == "spot" && *ec2i.State.Code&16 != 0 {
				running++
			}
		}
		if running >= 2 {
			c.Logf("instances are running, and identifiable as spot instances")
			break
		}
		c.Logf("waiting for instances to reach running state so their availability zone becomes visible...")
		time.Sleep(10 * time.Second)
	}

	for _, inst := range instances {
		hist := inst.PriceHistory(arvados.InstanceType{})
		c.Logf("%s price history: %v", inst.ID(), hist)
		c.Check(len(hist) > 0, check.Equals, true)

		histWithScratch := inst.PriceHistory(arvados.InstanceType{AddedScratch: 640 << 30})
		c.Logf("%s price history with 640 GiB scratch: %v", inst.ID(), histWithScratch)

		for i, ip := range hist {
			c.Check(ip.Price, check.Not(check.Equals), 0.0)
			if i > 0 {
				c.Check(ip.StartTime.Before(hist[i-1].StartTime), check.Equals, true)
			}
			c.Check(ip.Price < histWithScratch[i].Price, check.Equals, true)
		}
	}
}

func (*EC2InstanceSetSuite) TestWrapError(c *check.C) {
	retryError := awserr.New("Throttling", "", nil)
	wrapped := wrapError(retryError, &atomic.Value{})
	_, ok := wrapped.(cloud.RateLimitError)
	c.Check(ok, check.Equals, true)

	quotaError := awserr.New("InstanceLimitExceeded", "", nil)
	wrapped = wrapError(quotaError, nil)
	_, ok = wrapped.(cloud.QuotaError)
	c.Check(ok, check.Equals, true)

	capacityError := awserr.New("InsufficientInstanceCapacity", "", nil)
	wrapped = wrapError(capacityError, nil)
	caperr, ok := wrapped.(cloud.CapacityError)
	c.Check(ok, check.Equals, true)
	c.Check(caperr.IsCapacityError(), check.Equals, true)
	c.Check(caperr.IsInstanceTypeSpecific(), check.Equals, true)
}
