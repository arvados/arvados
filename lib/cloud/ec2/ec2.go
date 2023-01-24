// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package ec2

import (
	"crypto/md5"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"git.arvados.org/arvados.git/lib/cloud"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/ec2rolecreds"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

// Driver is the ec2 implementation of the cloud.Driver interface.
var Driver = cloud.DriverFunc(newEC2InstanceSet)

const (
	throttleDelayMin = time.Second
	throttleDelayMax = time.Minute
)

type ec2InstanceSetConfig struct {
	AccessKeyID             string
	SecretAccessKey         string
	Region                  string
	SecurityGroupIDs        arvados.StringSet
	SubnetID                string
	AdminUsername           string
	EBSVolumeType           string
	EBSPrice                float64
	IAMInstanceProfile      string
	SpotPriceUpdateInterval arvados.Duration
}

type ec2Interface interface {
	DescribeKeyPairs(input *ec2.DescribeKeyPairsInput) (*ec2.DescribeKeyPairsOutput, error)
	ImportKeyPair(input *ec2.ImportKeyPairInput) (*ec2.ImportKeyPairOutput, error)
	RunInstances(input *ec2.RunInstancesInput) (*ec2.Reservation, error)
	DescribeInstances(input *ec2.DescribeInstancesInput) (*ec2.DescribeInstancesOutput, error)
	DescribeInstanceStatusPages(input *ec2.DescribeInstanceStatusInput, fn func(*ec2.DescribeInstanceStatusOutput, bool) bool) error
	DescribeSpotPriceHistoryPages(input *ec2.DescribeSpotPriceHistoryInput, fn func(*ec2.DescribeSpotPriceHistoryOutput, bool) bool) error
	CreateTags(input *ec2.CreateTagsInput) (*ec2.CreateTagsOutput, error)
	TerminateInstances(input *ec2.TerminateInstancesInput) (*ec2.TerminateInstancesOutput, error)
}

type ec2InstanceSet struct {
	ec2config              ec2InstanceSetConfig
	instanceSetID          cloud.InstanceSetID
	logger                 logrus.FieldLogger
	client                 ec2Interface
	keysMtx                sync.Mutex
	keys                   map[string]string
	throttleDelayCreate    atomic.Value
	throttleDelayInstances atomic.Value

	prices        map[priceKey][]cloud.InstancePrice
	pricesLock    sync.Mutex
	pricesUpdated map[priceKey]time.Time
}

func newEC2InstanceSet(config json.RawMessage, instanceSetID cloud.InstanceSetID, _ cloud.SharedResourceTags, logger logrus.FieldLogger) (prv cloud.InstanceSet, err error) {
	instanceSet := &ec2InstanceSet{
		instanceSetID: instanceSetID,
		logger:        logger,
	}
	err = json.Unmarshal(config, &instanceSet.ec2config)
	if err != nil {
		return nil, err
	}

	sess, err := session.NewSession()
	if err != nil {
		return nil, err
	}
	// First try any static credentials, fall back to an IAM instance profile/role
	creds := credentials.NewChainCredentials(
		[]credentials.Provider{
			&credentials.StaticProvider{Value: credentials.Value{AccessKeyID: instanceSet.ec2config.AccessKeyID, SecretAccessKey: instanceSet.ec2config.SecretAccessKey}},
			&ec2rolecreds.EC2RoleProvider{Client: ec2metadata.New(sess)},
		})

	awsConfig := aws.NewConfig().WithCredentials(creds).WithRegion(instanceSet.ec2config.Region)
	instanceSet.client = ec2.New(session.Must(session.NewSession(awsConfig)))
	instanceSet.keys = make(map[string]string)
	if instanceSet.ec2config.EBSVolumeType == "" {
		instanceSet.ec2config.EBSVolumeType = "gp2"
	}
	return instanceSet, nil
}

func awsKeyFingerprint(pk ssh.PublicKey) (md5fp string, sha1fp string, err error) {
	// AWS key fingerprints don't use the usual key fingerprint
	// you get from ssh-keygen or ssh.FingerprintLegacyMD5()
	// (you can get that from md5.Sum(pk.Marshal())
	//
	// AWS uses the md5 or sha1 of the PKIX DER encoding of the
	// public key, so calculate those fingerprints here.
	var rsaPub struct {
		Name string
		E    *big.Int
		N    *big.Int
	}
	if err := ssh.Unmarshal(pk.Marshal(), &rsaPub); err != nil {
		return "", "", fmt.Errorf("agent: Unmarshal failed to parse public key: %v", err)
	}
	rsaPk := rsa.PublicKey{
		E: int(rsaPub.E.Int64()),
		N: rsaPub.N,
	}
	pkix, _ := x509.MarshalPKIXPublicKey(&rsaPk)
	md5pkix := md5.Sum([]byte(pkix))
	sha1pkix := sha1.Sum([]byte(pkix))
	md5fp = ""
	sha1fp = ""
	for i := 0; i < len(md5pkix); i++ {
		md5fp += fmt.Sprintf(":%02x", md5pkix[i])
	}
	for i := 0; i < len(sha1pkix); i++ {
		sha1fp += fmt.Sprintf(":%02x", sha1pkix[i])
	}
	return md5fp[1:], sha1fp[1:], nil
}

func (instanceSet *ec2InstanceSet) Create(
	instanceType arvados.InstanceType,
	imageID cloud.ImageID,
	newTags cloud.InstanceTags,
	initCommand cloud.InitCommand,
	publicKey ssh.PublicKey) (cloud.Instance, error) {

	md5keyFingerprint, sha1keyFingerprint, err := awsKeyFingerprint(publicKey)
	if err != nil {
		return nil, fmt.Errorf("Could not make key fingerprint: %v", err)
	}
	instanceSet.keysMtx.Lock()
	var keyname string
	var ok bool
	if keyname, ok = instanceSet.keys[md5keyFingerprint]; !ok {
		keyout, err := instanceSet.client.DescribeKeyPairs(&ec2.DescribeKeyPairsInput{
			Filters: []*ec2.Filter{{
				Name:   aws.String("fingerprint"),
				Values: []*string{&md5keyFingerprint, &sha1keyFingerprint},
			}},
		})
		if err != nil {
			return nil, fmt.Errorf("Could not search for keypair: %v", err)
		}

		if len(keyout.KeyPairs) > 0 {
			keyname = *(keyout.KeyPairs[0].KeyName)
		} else {
			keyname = "arvados-dispatch-keypair-" + md5keyFingerprint
			_, err := instanceSet.client.ImportKeyPair(&ec2.ImportKeyPairInput{
				KeyName:           &keyname,
				PublicKeyMaterial: ssh.MarshalAuthorizedKey(publicKey),
			})
			if err != nil {
				return nil, fmt.Errorf("Could not import keypair: %v", err)
			}
		}
		instanceSet.keys[md5keyFingerprint] = keyname
	}
	instanceSet.keysMtx.Unlock()

	ec2tags := []*ec2.Tag{}
	for k, v := range newTags {
		ec2tags = append(ec2tags, &ec2.Tag{
			Key:   aws.String(k),
			Value: aws.String(v),
		})
	}

	var groups []string
	for sg := range instanceSet.ec2config.SecurityGroupIDs {
		groups = append(groups, sg)
	}

	rii := ec2.RunInstancesInput{
		ImageId:      aws.String(string(imageID)),
		InstanceType: &instanceType.ProviderType,
		MaxCount:     aws.Int64(1),
		MinCount:     aws.Int64(1),
		KeyName:      &keyname,

		NetworkInterfaces: []*ec2.InstanceNetworkInterfaceSpecification{
			{
				AssociatePublicIpAddress: aws.Bool(false),
				DeleteOnTermination:      aws.Bool(true),
				DeviceIndex:              aws.Int64(0),
				Groups:                   aws.StringSlice(groups),
				SubnetId:                 &instanceSet.ec2config.SubnetID,
			}},
		DisableApiTermination:             aws.Bool(false),
		InstanceInitiatedShutdownBehavior: aws.String("terminate"),
		TagSpecifications: []*ec2.TagSpecification{
			{
				ResourceType: aws.String("instance"),
				Tags:         ec2tags,
			}},
		UserData: aws.String(base64.StdEncoding.EncodeToString([]byte("#!/bin/sh\n" + initCommand + "\n"))),
	}

	if instanceType.AddedScratch > 0 {
		rii.BlockDeviceMappings = []*ec2.BlockDeviceMapping{{
			DeviceName: aws.String("/dev/xvdt"),
			Ebs: &ec2.EbsBlockDevice{
				DeleteOnTermination: aws.Bool(true),
				VolumeSize:          aws.Int64((int64(instanceType.AddedScratch) + (1<<30 - 1)) >> 30),
				VolumeType:          &instanceSet.ec2config.EBSVolumeType,
			}}}
	}

	if instanceType.Preemptible {
		rii.InstanceMarketOptions = &ec2.InstanceMarketOptionsRequest{
			MarketType: aws.String("spot"),
			SpotOptions: &ec2.SpotMarketOptions{
				InstanceInterruptionBehavior: aws.String("terminate"),
				MaxPrice:                     aws.String(fmt.Sprintf("%v", instanceType.Price)),
			}}
	}

	if instanceSet.ec2config.IAMInstanceProfile != "" {
		rii.IamInstanceProfile = &ec2.IamInstanceProfileSpecification{
			Name: aws.String(instanceSet.ec2config.IAMInstanceProfile),
		}
	}

	rsv, err := instanceSet.client.RunInstances(&rii)
	err = wrapError(err, &instanceSet.throttleDelayCreate)
	if err != nil {
		return nil, err
	}
	return &ec2Instance{
		provider: instanceSet,
		instance: rsv.Instances[0],
	}, nil
}

func (instanceSet *ec2InstanceSet) Instances(tags cloud.InstanceTags) (instances []cloud.Instance, err error) {
	var filters []*ec2.Filter
	for k, v := range tags {
		filters = append(filters, &ec2.Filter{
			Name:   aws.String("tag:" + k),
			Values: []*string{aws.String(v)},
		})
	}
	needAZs := false
	dii := &ec2.DescribeInstancesInput{Filters: filters}
	for {
		dio, err := instanceSet.client.DescribeInstances(dii)
		err = wrapError(err, &instanceSet.throttleDelayInstances)
		if err != nil {
			return nil, err
		}

		for _, rsv := range dio.Reservations {
			for _, inst := range rsv.Instances {
				if *inst.State.Name != "shutting-down" && *inst.State.Name != "terminated" {
					instances = append(instances, &ec2Instance{
						provider: instanceSet,
						instance: inst,
					})
					if aws.StringValue(inst.InstanceLifecycle) == "spot" {
						needAZs = true
					}
				}
			}
		}
		if dio.NextToken == nil {
			break
		}
		dii.NextToken = dio.NextToken
	}
	if needAZs && instanceSet.ec2config.SpotPriceUpdateInterval > 0 {
		az := map[string]string{}
		err := instanceSet.client.DescribeInstanceStatusPages(&ec2.DescribeInstanceStatusInput{
			IncludeAllInstances: aws.Bool(true),
		}, func(page *ec2.DescribeInstanceStatusOutput, lastPage bool) bool {
			for _, ent := range page.InstanceStatuses {
				az[*ent.InstanceId] = *ent.AvailabilityZone
			}
			return true
		})
		if err != nil {
			instanceSet.logger.Warnf("error getting instance statuses: %s", err)
		}
		for _, inst := range instances {
			inst := inst.(*ec2Instance)
			inst.availabilityZone = az[*inst.instance.InstanceId]
		}
		instanceSet.updateSpotPrices(instances)
	}
	return instances, err
}

type priceKey struct {
	instanceType     string
	spot             bool
	availabilityZone string
}

// Refresh recent spot instance pricing data for the given instances,
// unless we already have recent pricing data for all relevant types.
func (instanceSet *ec2InstanceSet) updateSpotPrices(instances []cloud.Instance) {
	if len(instances) == 0 {
		return
	}

	instanceSet.pricesLock.Lock()
	defer instanceSet.pricesLock.Unlock()
	if instanceSet.prices == nil {
		instanceSet.prices = map[priceKey][]cloud.InstancePrice{}
		instanceSet.pricesUpdated = map[priceKey]time.Time{}
	}

	updateTime := time.Now()
	staleTime := updateTime.Add(-instanceSet.ec2config.SpotPriceUpdateInterval.Duration())
	needUpdate := false
	var typeFilterValues []*string
	for _, inst := range instances {
		ec2inst := inst.(*ec2Instance).instance
		if aws.StringValue(ec2inst.InstanceLifecycle) == "spot" {
			pk := priceKey{
				instanceType:     *ec2inst.InstanceType,
				spot:             true,
				availabilityZone: inst.(*ec2Instance).availabilityZone,
			}
			if instanceSet.pricesUpdated[pk].Before(staleTime) {
				needUpdate = true
			}
			typeFilterValues = append(typeFilterValues, ec2inst.InstanceType)
		}
	}
	if !needUpdate {
		return
	}
	// Get 3x update interval worth of pricing data. (Ideally the
	// AWS API would tell us "we have shown you all of the price
	// changes up to time T", but it doesn't, so we'll just ask
	// for 3 intervals worth of data on each update, de-duplicate
	// the data points, and not worry too much about occasionally
	// missing some data points when our lookups fail twice in a
	// row.
	dsphi := &ec2.DescribeSpotPriceHistoryInput{
		StartTime: aws.Time(updateTime.Add(-3 * instanceSet.ec2config.SpotPriceUpdateInterval.Duration())),
		Filters: []*ec2.Filter{
			&ec2.Filter{Name: aws.String("instance-type"), Values: typeFilterValues},
			&ec2.Filter{Name: aws.String("product-description"), Values: []*string{aws.String("Linux/UNIX")}},
		},
	}
	err := instanceSet.client.DescribeSpotPriceHistoryPages(dsphi, func(page *ec2.DescribeSpotPriceHistoryOutput, lastPage bool) bool {
		for _, ent := range page.SpotPriceHistory {
			if ent.InstanceType == nil || ent.SpotPrice == nil || ent.Timestamp == nil {
				// bogus record?
				continue
			}
			price, err := strconv.ParseFloat(*ent.SpotPrice, 64)
			if err != nil {
				// bogus record?
				continue
			}
			pk := priceKey{
				instanceType:     *ent.InstanceType,
				spot:             true,
				availabilityZone: *ent.AvailabilityZone,
			}
			instanceSet.prices[pk] = append(instanceSet.prices[pk], cloud.InstancePrice{
				StartTime: *ent.Timestamp,
				Price:     price,
			})
			instanceSet.pricesUpdated[pk] = updateTime
		}
		return true
	})
	if err != nil {
		instanceSet.logger.Warnf("error retrieving spot instance prices: %s", err)
	}

	expiredTime := updateTime.Add(-64 * instanceSet.ec2config.SpotPriceUpdateInterval.Duration())
	for pk, last := range instanceSet.pricesUpdated {
		if last.Before(expiredTime) {
			delete(instanceSet.pricesUpdated, pk)
			delete(instanceSet.prices, pk)
		}
	}
	for pk, prices := range instanceSet.prices {
		instanceSet.prices[pk] = cloud.NormalizePriceHistory(prices)
	}
}

func (instanceSet *ec2InstanceSet) Stop() {
}

type ec2Instance struct {
	provider         *ec2InstanceSet
	instance         *ec2.Instance
	availabilityZone string // sometimes available for spot instances
}

func (inst *ec2Instance) ID() cloud.InstanceID {
	return cloud.InstanceID(*inst.instance.InstanceId)
}

func (inst *ec2Instance) String() string {
	return *inst.instance.InstanceId
}

func (inst *ec2Instance) ProviderType() string {
	return *inst.instance.InstanceType
}

func (inst *ec2Instance) SetTags(newTags cloud.InstanceTags) error {
	var ec2tags []*ec2.Tag
	for k, v := range newTags {
		ec2tags = append(ec2tags, &ec2.Tag{
			Key:   aws.String(k),
			Value: aws.String(v),
		})
	}

	_, err := inst.provider.client.CreateTags(&ec2.CreateTagsInput{
		Resources: []*string{inst.instance.InstanceId},
		Tags:      ec2tags,
	})

	return err
}

func (inst *ec2Instance) Tags() cloud.InstanceTags {
	tags := make(map[string]string)

	for _, t := range inst.instance.Tags {
		tags[*t.Key] = *t.Value
	}

	return tags
}

func (inst *ec2Instance) Destroy() error {
	_, err := inst.provider.client.TerminateInstances(&ec2.TerminateInstancesInput{
		InstanceIds: []*string{inst.instance.InstanceId},
	})
	return err
}

func (inst *ec2Instance) Address() string {
	if inst.instance.PrivateIpAddress != nil {
		return *inst.instance.PrivateIpAddress
	}
	return ""
}

func (inst *ec2Instance) RemoteUser() string {
	return inst.provider.ec2config.AdminUsername
}

func (inst *ec2Instance) VerifyHostKey(ssh.PublicKey, *ssh.Client) error {
	return cloud.ErrNotImplemented
}

// PriceHistory returns the price history for this specific instance.
//
// AWS documentation is elusive about whether the hourly cost of a
// given spot instance changes as the current spot price changes for
// the corresponding instance type and availability zone. Our
// implementation assumes the answer is yes, based on the following
// hints.
//
// https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/spot-requests.html
// says: "After your Spot Instance is running, if the Spot price rises
// above your maximum price, Amazon EC2 interrupts your Spot
// Instance." (This doesn't address what happens when the spot price
// rises *without* exceeding your maximum price.)
//
// https://docs.aws.amazon.com/whitepapers/latest/cost-optimization-leveraging-ec2-spot-instances/how-spot-instances-work.html
// says: "You pay the Spot price that's in effect, billed to the
// nearest second." (But it's not explicitly stated whether "the price
// in effect" changes over time for a given instance.)
//
// The same page also says, in a discussion about the effect of
// specifying a maximum price: "Note that you never pay more than the
// Spot price that is in effect when your Spot Instance is running."
// (The use of the phrase "is running", as opposed to "was launched",
// hints that pricing is dynamic.)
func (inst *ec2Instance) PriceHistory(instType arvados.InstanceType) []cloud.InstancePrice {
	inst.provider.pricesLock.Lock()
	defer inst.provider.pricesLock.Unlock()
	// Note updateSpotPrices currently populates
	// inst.provider.prices only for spot instances, so if
	// spot==false here, we will return no data.
	pk := priceKey{
		instanceType:     *inst.instance.InstanceType,
		spot:             aws.StringValue(inst.instance.InstanceLifecycle) == "spot",
		availabilityZone: inst.availabilityZone,
	}
	var prices []cloud.InstancePrice
	for _, price := range inst.provider.prices[pk] {
		// ceil(added scratch space in GiB)
		gib := (instType.AddedScratch + 1<<30 - 1) >> 30
		monthly := inst.provider.ec2config.EBSPrice * float64(gib)
		hourly := monthly / 30 / 24
		price.Price += hourly
		prices = append(prices, price)
	}
	return prices
}

type rateLimitError struct {
	error
	earliestRetry time.Time
}

func (err rateLimitError) EarliestRetry() time.Time {
	return err.earliestRetry
}

var isCodeCapacity = map[string]bool{
	"InsufficientInstanceCapacity": true,
	"VcpuLimitExceeded":            true,
	"MaxSpotInstanceCountExceeded": true,
}

// isErrorCapacity returns whether the error is to be throttled based on its code.
// Returns false if error is nil.
func isErrorCapacity(err error) bool {
	if aerr, ok := err.(awserr.Error); ok && aerr != nil {
		if _, ok := isCodeCapacity[aerr.Code()]; ok {
			return true
		}
	}
	return false
}

type ec2QuotaError struct {
	error
}

func (er *ec2QuotaError) IsQuotaError() bool {
	return true
}

func wrapError(err error, throttleValue *atomic.Value) error {
	if request.IsErrorThrottle(err) {
		// Back off exponentially until an upstream call
		// either succeeds or returns a non-throttle error.
		d, _ := throttleValue.Load().(time.Duration)
		d = d*3/2 + time.Second
		if d < throttleDelayMin {
			d = throttleDelayMin
		} else if d > throttleDelayMax {
			d = throttleDelayMax
		}
		throttleValue.Store(d)
		return rateLimitError{error: err, earliestRetry: time.Now().Add(d)}
	} else if isErrorCapacity(err) {
		return &ec2QuotaError{err}
	} else if err != nil {
		throttleValue.Store(time.Duration(0))
		return err
	}
	throttleValue.Store(time.Duration(0))
	return nil
}
