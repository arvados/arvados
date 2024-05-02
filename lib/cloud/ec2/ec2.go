// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package ec2

import (
	"context"
	"crypto/md5"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"git.arvados.org/arvados.git/lib/cloud"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/smithy-go"
	"github.com/prometheus/client_golang/prometheus"
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
	SubnetID                sliceOrSingleString
	AdminUsername           string
	EBSVolumeType           types.VolumeType
	EBSPrice                float64
	IAMInstanceProfile      string
	SpotPriceUpdateInterval arvados.Duration
}

type sliceOrSingleString []string

// UnmarshalJSON unmarshals an array of strings, and also accepts ""
// as [], and "foo" as ["foo"].
func (ss *sliceOrSingleString) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		*ss = nil
	} else if data[0] == '[' {
		var slice []string
		err := json.Unmarshal(data, &slice)
		if err != nil {
			return err
		}
		if len(slice) == 0 {
			*ss = nil
		} else {
			*ss = slice
		}
	} else {
		var str string
		err := json.Unmarshal(data, &str)
		if err != nil {
			return err
		}
		if str == "" {
			*ss = nil
		} else {
			*ss = []string{str}
		}
	}
	return nil
}

type ec2Interface interface {
	DescribeKeyPairs(context.Context, *ec2.DescribeKeyPairsInput, ...func(*ec2.Options)) (*ec2.DescribeKeyPairsOutput, error)
	ImportKeyPair(context.Context, *ec2.ImportKeyPairInput, ...func(*ec2.Options)) (*ec2.ImportKeyPairOutput, error)
	RunInstances(context.Context, *ec2.RunInstancesInput, ...func(*ec2.Options)) (*ec2.RunInstancesOutput, error)
	DescribeInstances(context.Context, *ec2.DescribeInstancesInput, ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error)
	DescribeInstanceStatus(context.Context, *ec2.DescribeInstanceStatusInput, ...func(*ec2.Options)) (*ec2.DescribeInstanceStatusOutput, error)
	DescribeSpotPriceHistory(context.Context, *ec2.DescribeSpotPriceHistoryInput, ...func(*ec2.Options)) (*ec2.DescribeSpotPriceHistoryOutput, error)
	CreateTags(context.Context, *ec2.CreateTagsInput, ...func(*ec2.Options)) (*ec2.CreateTagsOutput, error)
	TerminateInstances(context.Context, *ec2.TerminateInstancesInput, ...func(*ec2.Options)) (*ec2.TerminateInstancesOutput, error)
}

type ec2InstanceSet struct {
	ec2config              ec2InstanceSetConfig
	currentSubnetIDIndex   int32
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

	mInstances      *prometheus.GaugeVec
	mInstanceStarts *prometheus.CounterVec
}

func newEC2InstanceSet(config json.RawMessage, instanceSetID cloud.InstanceSetID, _ cloud.SharedResourceTags, logger logrus.FieldLogger, reg *prometheus.Registry) (prv cloud.InstanceSet, err error) {
	instanceSet := &ec2InstanceSet{
		instanceSetID: instanceSetID,
		logger:        logger,
	}
	err = json.Unmarshal(config, &instanceSet.ec2config)
	if err != nil {
		return nil, err
	}

	if len(instanceSet.ec2config.AccessKeyID)+len(instanceSet.ec2config.SecretAccessKey) > 0 {
		// AWS SDK will use credentials in environment vars if
		// present.
		os.Setenv("AWS_ACCESS_KEY_ID", instanceSet.ec2config.AccessKeyID)
		os.Setenv("AWS_SECRET_ACCESS_KEY", instanceSet.ec2config.SecretAccessKey)
	} else {
		os.Unsetenv("AWS_ACCESS_KEY_ID")
		os.Unsetenv("AWS_SECRET_ACCESS_KEY")
	}
	awsConfig, err := awsconfig.LoadDefaultConfig(context.TODO(),
		awsconfig.WithRegion(instanceSet.ec2config.Region))
	if err != nil {
		return nil, err
	}

	instanceSet.client = ec2.NewFromConfig(awsConfig)
	instanceSet.keys = make(map[string]string)
	if instanceSet.ec2config.EBSVolumeType == "" {
		instanceSet.ec2config.EBSVolumeType = "gp2"
	}

	// Set up metrics
	instanceSet.mInstances = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "arvados",
		Subsystem: "dispatchcloud",
		Name:      "ec2_instances",
		Help:      "Number of instances running",
	}, []string{"subnet_id"})
	instanceSet.mInstanceStarts = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "arvados",
		Subsystem: "dispatchcloud",
		Name:      "ec2_instance_starts_total",
		Help:      "Number of attempts to start a new instance",
	}, []string{"subnet_id", "success"})
	// Initialize all of the series we'll be reporting.  Otherwise
	// the {subnet=A, success=0} series doesn't appear in metrics
	// at all until there's a failure in subnet A.
	for _, subnet := range instanceSet.ec2config.SubnetID {
		instanceSet.mInstanceStarts.WithLabelValues(subnet, "0").Add(0)
		instanceSet.mInstanceStarts.WithLabelValues(subnet, "1").Add(0)
	}
	if len(instanceSet.ec2config.SubnetID) == 0 {
		instanceSet.mInstanceStarts.WithLabelValues("", "0").Add(0)
		instanceSet.mInstanceStarts.WithLabelValues("", "1").Add(0)
	}
	if reg != nil {
		reg.MustRegister(instanceSet.mInstances)
		reg.MustRegister(instanceSet.mInstanceStarts)
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

	ec2tags := []types.Tag{}
	for k, v := range newTags {
		ec2tags = append(ec2tags, types.Tag{
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
		InstanceType: types.InstanceType(instanceType.ProviderType),
		MaxCount:     aws.Int32(1),
		MinCount:     aws.Int32(1),

		NetworkInterfaces: []types.InstanceNetworkInterfaceSpecification{{
			AssociatePublicIpAddress: aws.Bool(false),
			DeleteOnTermination:      aws.Bool(true),
			DeviceIndex:              aws.Int32(0),
			Groups:                   groups,
		}},
		DisableApiTermination:             aws.Bool(false),
		InstanceInitiatedShutdownBehavior: types.ShutdownBehaviorTerminate,
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeInstance,
				Tags:         ec2tags,
			}},
		MetadataOptions: &types.InstanceMetadataOptionsRequest{
			// Require IMDSv2, as described at
			// https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/configuring-IMDS-new-instances.html
			HttpEndpoint: types.InstanceMetadataEndpointStateEnabled,
			HttpTokens:   types.HttpTokensStateRequired,
		},
		UserData: aws.String(base64.StdEncoding.EncodeToString([]byte("#!/bin/sh\n" + initCommand + "\n"))),
	}

	if publicKey != nil {
		keyname, err := instanceSet.getKeyName(publicKey)
		if err != nil {
			return nil, err
		}
		rii.KeyName = &keyname
	}

	if instanceType.AddedScratch > 0 {
		rii.BlockDeviceMappings = []types.BlockDeviceMapping{{
			DeviceName: aws.String("/dev/xvdt"),
			Ebs: &types.EbsBlockDevice{
				DeleteOnTermination: aws.Bool(true),
				VolumeSize:          aws.Int32(int32((int64(instanceType.AddedScratch) + (1<<30 - 1)) >> 30)),
				VolumeType:          instanceSet.ec2config.EBSVolumeType,
			}}}
	}

	if instanceType.Preemptible {
		rii.InstanceMarketOptions = &types.InstanceMarketOptionsRequest{
			MarketType: types.MarketTypeSpot,
			SpotOptions: &types.SpotMarketOptions{
				InstanceInterruptionBehavior: types.InstanceInterruptionBehaviorTerminate,
				MaxPrice:                     aws.String(fmt.Sprintf("%v", instanceType.Price)),
			}}
	}

	if instanceSet.ec2config.IAMInstanceProfile != "" {
		rii.IamInstanceProfile = &types.IamInstanceProfileSpecification{
			Name: aws.String(instanceSet.ec2config.IAMInstanceProfile),
		}
	}

	var rsv *ec2.RunInstancesOutput
	var errToReturn error
	subnets := instanceSet.ec2config.SubnetID
	currentSubnetIDIndex := int(atomic.LoadInt32(&instanceSet.currentSubnetIDIndex))
	for tryOffset := 0; ; tryOffset++ {
		tryIndex := 0
		trySubnet := ""
		if len(subnets) > 0 {
			tryIndex = (currentSubnetIDIndex + tryOffset) % len(subnets)
			trySubnet = subnets[tryIndex]
			rii.NetworkInterfaces[0].SubnetId = aws.String(trySubnet)
		}
		var err error
		rsv, err = instanceSet.client.RunInstances(context.TODO(), &rii)
		instanceSet.mInstanceStarts.WithLabelValues(trySubnet, boolLabelValue[err == nil]).Add(1)
		if !isErrorCapacity(errToReturn) || isErrorCapacity(err) {
			// We want to return the last capacity error,
			// if any; otherwise the last non-capacity
			// error.
			errToReturn = err
		}
		if isErrorSubnetSpecific(err) &&
			tryOffset < len(subnets)-1 {
			instanceSet.logger.WithError(err).WithField("SubnetID", subnets[tryIndex]).
				Warn("RunInstances failed, trying next subnet")
			continue
		}
		// Succeeded, or exhausted all subnets, or got a
		// non-subnet-related error.
		//
		// We intentionally update currentSubnetIDIndex even
		// in the non-retryable-failure case here to avoid a
		// situation where successive calls to Create() keep
		// returning errors for the same subnet (perhaps
		// "subnet full") and never reveal the errors for the
		// other configured subnets (perhaps "subnet ID
		// invalid").
		atomic.StoreInt32(&instanceSet.currentSubnetIDIndex, int32(tryIndex))
		break
	}
	if rsv == nil || len(rsv.Instances) == 0 {
		return nil, wrapError(errToReturn, &instanceSet.throttleDelayCreate)
	}
	return &ec2Instance{
		provider: instanceSet,
		instance: rsv.Instances[0],
	}, nil
}

func (instanceSet *ec2InstanceSet) getKeyName(publicKey ssh.PublicKey) (string, error) {
	instanceSet.keysMtx.Lock()
	defer instanceSet.keysMtx.Unlock()
	md5keyFingerprint, sha1keyFingerprint, err := awsKeyFingerprint(publicKey)
	if err != nil {
		return "", fmt.Errorf("Could not make key fingerprint: %v", err)
	}
	if keyname, ok := instanceSet.keys[md5keyFingerprint]; ok {
		return keyname, nil
	}
	keyout, err := instanceSet.client.DescribeKeyPairs(context.TODO(), &ec2.DescribeKeyPairsInput{
		Filters: []types.Filter{{
			Name:   aws.String("fingerprint"),
			Values: []string{md5keyFingerprint, sha1keyFingerprint},
		}},
	})
	if err != nil {
		return "", fmt.Errorf("Could not search for keypair: %v", err)
	}
	if len(keyout.KeyPairs) > 0 {
		return *(keyout.KeyPairs[0].KeyName), nil
	}
	keyname := "arvados-dispatch-keypair-" + md5keyFingerprint
	_, err = instanceSet.client.ImportKeyPair(context.TODO(), &ec2.ImportKeyPairInput{
		KeyName:           &keyname,
		PublicKeyMaterial: ssh.MarshalAuthorizedKey(publicKey),
	})
	if err != nil {
		return "", fmt.Errorf("Could not import keypair: %v", err)
	}
	instanceSet.keys[md5keyFingerprint] = keyname
	return keyname, nil
}

func (instanceSet *ec2InstanceSet) Instances(tags cloud.InstanceTags) (instances []cloud.Instance, err error) {
	var filters []types.Filter
	for k, v := range tags {
		filters = append(filters, types.Filter{
			Name:   aws.String("tag:" + k),
			Values: []string{v},
		})
	}
	needAZs := false
	dii := &ec2.DescribeInstancesInput{Filters: filters}
	for {
		dio, err := instanceSet.client.DescribeInstances(context.TODO(), dii)
		err = wrapError(err, &instanceSet.throttleDelayInstances)
		if err != nil {
			return nil, err
		}

		for _, rsv := range dio.Reservations {
			for _, inst := range rsv.Instances {
				switch inst.State.Name {
				case types.InstanceStateNameShuttingDown:
				case types.InstanceStateNameTerminated:
				default:
					instances = append(instances, &ec2Instance{
						provider: instanceSet,
						instance: inst,
					})
					if inst.InstanceLifecycle == types.InstanceLifecycleTypeSpot {
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
		disi := &ec2.DescribeInstanceStatusInput{IncludeAllInstances: aws.Bool(true)}
		for {
			page, err := instanceSet.client.DescribeInstanceStatus(context.TODO(), disi)
			if err != nil {
				instanceSet.logger.Warnf("error getting instance statuses: %s", err)
				break
			}
			for _, ent := range page.InstanceStatuses {
				az[*ent.InstanceId] = *ent.AvailabilityZone
			}
			if page.NextToken == nil {
				break
			}
			disi.NextToken = page.NextToken
		}
		for _, inst := range instances {
			inst := inst.(*ec2Instance)
			inst.availabilityZone = az[*inst.instance.InstanceId]
		}
		instanceSet.updateSpotPrices(instances)
	}

	// Count instances in each subnet, and report in metrics.
	subnetInstances := map[string]int{"": 0}
	for _, subnet := range instanceSet.ec2config.SubnetID {
		subnetInstances[subnet] = 0
	}
	for _, inst := range instances {
		subnet := inst.(*ec2Instance).instance.SubnetId
		if subnet != nil {
			subnetInstances[*subnet]++
		} else {
			subnetInstances[""]++
		}
	}
	for subnet, count := range subnetInstances {
		instanceSet.mInstances.WithLabelValues(subnet).Set(float64(count))
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
	allTypes := map[types.InstanceType]bool{}

	for _, inst := range instances {
		ec2inst := inst.(*ec2Instance).instance
		if ec2inst.InstanceLifecycle == types.InstanceLifecycleTypeSpot {
			pk := priceKey{
				instanceType:     string(ec2inst.InstanceType),
				spot:             true,
				availabilityZone: inst.(*ec2Instance).availabilityZone,
			}
			if instanceSet.pricesUpdated[pk].Before(staleTime) {
				needUpdate = true
			}
			allTypes[ec2inst.InstanceType] = true
		}
	}
	if !needUpdate {
		return
	}
	var typeFilterValues []string
	for instanceType := range allTypes {
		typeFilterValues = append(typeFilterValues, string(instanceType))
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
		Filters: []types.Filter{
			types.Filter{Name: aws.String("instance-type"), Values: typeFilterValues},
			types.Filter{Name: aws.String("product-description"), Values: []string{"Linux/UNIX"}},
		},
	}
	for {
		page, err := instanceSet.client.DescribeSpotPriceHistory(context.TODO(), dsphi)
		if err != nil {
			instanceSet.logger.Warnf("error retrieving spot instance prices: %s", err)
			break
		}
		for _, ent := range page.SpotPriceHistory {
			if ent.InstanceType == "" || ent.SpotPrice == nil || ent.Timestamp == nil {
				// bogus record?
				continue
			}
			price, err := strconv.ParseFloat(*ent.SpotPrice, 64)
			if err != nil {
				// bogus record?
				continue
			}
			pk := priceKey{
				instanceType:     string(ent.InstanceType),
				spot:             true,
				availabilityZone: *ent.AvailabilityZone,
			}
			instanceSet.prices[pk] = append(instanceSet.prices[pk], cloud.InstancePrice{
				StartTime: *ent.Timestamp,
				Price:     price,
			})
			instanceSet.pricesUpdated[pk] = updateTime
		}
		if page.NextToken == nil {
			break
		}
		dsphi.NextToken = page.NextToken
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
	instance         types.Instance
	availabilityZone string // sometimes available for spot instances
}

func (inst *ec2Instance) ID() cloud.InstanceID {
	return cloud.InstanceID(*inst.instance.InstanceId)
}

func (inst *ec2Instance) String() string {
	return *inst.instance.InstanceId
}

func (inst *ec2Instance) ProviderType() string {
	return string(inst.instance.InstanceType)
}

func (inst *ec2Instance) SetTags(newTags cloud.InstanceTags) error {
	var ec2tags []types.Tag
	for k, v := range newTags {
		ec2tags = append(ec2tags, types.Tag{
			Key:   aws.String(k),
			Value: aws.String(v),
		})
	}

	_, err := inst.provider.client.CreateTags(context.TODO(), &ec2.CreateTagsInput{
		Resources: []string{*inst.instance.InstanceId},
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
	_, err := inst.provider.client.TerminateInstances(context.TODO(), &ec2.TerminateInstancesInput{
		InstanceIds: []string{*inst.instance.InstanceId},
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
		instanceType:     string(inst.instance.InstanceType),
		spot:             inst.instance.InstanceLifecycle == types.InstanceLifecycleTypeSpot,
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

type capacityError struct {
	error
	isInstanceTypeSpecific bool
}

func (er *capacityError) IsCapacityError() bool {
	return true
}

func (er *capacityError) IsInstanceTypeSpecific() bool {
	return er.isInstanceTypeSpecific
}

var isCodeQuota = map[string]bool{
	"InstanceLimitExceeded":             true,
	"InsufficientAddressCapacity":       true,
	"InsufficientFreeAddressesInSubnet": true,
	"InsufficientVolumeCapacity":        true,
	"MaxSpotInstanceCountExceeded":      true,
	"VcpuLimitExceeded":                 true,
}

// isErrorQuota returns whether the error indicates we have reached
// some usage quota/limit -- i.e., immediately retrying with an equal
// or larger instance type will probably not work.
//
// Returns false if error is nil.
func isErrorQuota(err error) bool {
	var aerr smithy.APIError
	if errors.As(err, &aerr) {
		if _, ok := isCodeQuota[aerr.ErrorCode()]; ok {
			return true
		}
	}
	return false
}

var reSubnetSpecificInvalidParameterMessage = regexp.MustCompile(`(?ms).*( subnet |sufficient free [Ii]pv[46] addresses).*`)

// isErrorSubnetSpecific returns true if the problem encountered by
// RunInstances might be avoided by trying a different subnet.
func isErrorSubnetSpecific(err error) bool {
	var aerr smithy.APIError
	if !errors.As(err, &aerr) {
		return false
	}
	code := aerr.ErrorCode()
	return strings.Contains(code, "Subnet") ||
		code == "InsufficientInstanceCapacity" ||
		code == "InsufficientVolumeCapacity" ||
		code == "Unsupported" ||
		// See TestIsErrorSubnetSpecific for examples of why
		// we look for substrings in code/message instead of
		// only using specific codes here.
		(strings.Contains(code, "InvalidParameter") &&
			reSubnetSpecificInvalidParameterMessage.MatchString(aerr.ErrorMessage()))
}

// isErrorCapacity returns true if the error indicates lack of
// capacity (either temporary or permanent) to run a specific instance
// type -- i.e., retrying with a different instance type might
// succeed.
func isErrorCapacity(err error) bool {
	var aerr smithy.APIError
	if !errors.As(err, &aerr) {
		return false
	}
	code := aerr.ErrorCode()
	return code == "InsufficientInstanceCapacity" ||
		(code == "Unsupported" && strings.Contains(aerr.ErrorMessage(), "requested instance type"))
}

type ec2QuotaError struct {
	error
}

func (er *ec2QuotaError) IsQuotaError() bool {
	return true
}

func isThrottleError(err error) bool {
	var aerr smithy.APIError
	if !errors.As(err, &aerr) {
		return false
	}
	_, is := retry.DefaultThrottleErrorCodes[aerr.ErrorCode()]
	return is
}

func wrapError(err error, throttleValue *atomic.Value) error {
	if isThrottleError(err) {
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
	} else if isErrorQuota(err) {
		return &ec2QuotaError{err}
	} else if isErrorCapacity(err) {
		return &capacityError{err, true}
	} else if err != nil {
		throttleValue.Store(time.Duration(0))
		return err
	}
	throttleValue.Store(time.Duration(0))
	return nil
}

var boolLabelValue = map[bool]string{false: "0", true: "1"}
