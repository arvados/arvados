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
	"sync"
	"sync/atomic"
	"time"

	"git.arvados.org/arvados.git/lib/cloud"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"github.com/aws/aws-sdk-go/aws"
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
	AccessKeyID      string
	SecretAccessKey  string
	Region           string
	SecurityGroupIDs arvados.StringSet
	SubnetID         string
	AdminUsername    string
	EBSVolumeType    string
}

type ec2Interface interface {
	DescribeKeyPairs(input *ec2.DescribeKeyPairsInput) (*ec2.DescribeKeyPairsOutput, error)
	ImportKeyPair(input *ec2.ImportKeyPairInput) (*ec2.ImportKeyPairOutput, error)
	RunInstances(input *ec2.RunInstancesInput) (*ec2.Reservation, error)
	DescribeInstances(input *ec2.DescribeInstancesInput) (*ec2.DescribeInstancesOutput, error)
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
					instances = append(instances, &ec2Instance{instanceSet, inst})
				}
			}
		}
		if dio.NextToken == nil {
			return instances, err
		}
		dii.NextToken = dio.NextToken
	}
}

func (instanceSet *ec2InstanceSet) Stop() {
}

type ec2Instance struct {
	provider *ec2InstanceSet
	instance *ec2.Instance
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

type rateLimitError struct {
	error
	earliestRetry time.Time
}

func (err rateLimitError) EarliestRetry() time.Time {
	return err.earliestRetry
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
	} else if err != nil {
		throttleValue.Store(time.Duration(0))
		return err
	}
	throttleValue.Store(time.Duration(0))
	return nil
}
