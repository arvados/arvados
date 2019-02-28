// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package ec2

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"git.curoverse.com/arvados.git/lib/cloud"
	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

const ARVADOS_DISPATCH_ID = "arvados-dispatch-id"
const TAG_PREFIX = "arvados-dispatch-tag-"

// Driver is the ec2 implementation of the cloud.Driver interface.
var Driver = cloud.DriverFunc(newEC2InstanceSet)

type ec2InstanceSetConfig struct {
	AccessKeyID     string
	SecretAccessKey string
	Region          string
	SecurityGroupId string
	SubnetId        string
	AdminUsername   string
	KeyPairName     string
}

type ec2InstanceSet struct {
	ec2config    ec2InstanceSetConfig
	dispatcherID cloud.InstanceSetID
	logger       logrus.FieldLogger
	client       *ec2.EC2
	importedKey  bool
}

func newEC2InstanceSet(config json.RawMessage, dispatcherID cloud.InstanceSetID, logger logrus.FieldLogger) (prv cloud.InstanceSet, err error) {
	instanceSet := &ec2InstanceSet{
		dispatcherID: dispatcherID,
		logger:       logger,
	}
	err = json.Unmarshal(config, &instanceSet.ec2config)
	if err != nil {
		return nil, err
	}
	awsConfig := aws.NewConfig().
		WithCredentials(credentials.NewStaticCredentials(
			instanceSet.ec2config.AccessKeyID,
			instanceSet.ec2config.SecretAccessKey,
			"")).
		WithRegion(instanceSet.ec2config.Region)
	instanceSet.client = ec2.New(session.Must(session.NewSession(awsConfig)))
	return instanceSet, nil
}

func (instanceSet *ec2InstanceSet) Create(
	instanceType arvados.InstanceType,
	imageID cloud.ImageID,
	newTags cloud.InstanceTags,
	initCommand cloud.InitCommand,
	publicKey ssh.PublicKey) (cloud.Instance, error) {

	if !instanceSet.importedKey {
		instanceSet.client.ImportKeyPair(&ec2.ImportKeyPairInput{
			KeyName:           &instanceSet.ec2config.KeyPairName,
			PublicKeyMaterial: ssh.MarshalAuthorizedKey(publicKey),
		})
		instanceSet.importedKey = true
	}

	ec2tags := []*ec2.Tag{
		&ec2.Tag{
			Key:   aws.String(ARVADOS_DISPATCH_ID),
			Value: aws.String(string(instanceSet.dispatcherID)),
		},
		&ec2.Tag{
			Key:   aws.String("arvados-class"),
			Value: aws.String("dynamic-compute"),
		},
	}
	for k, v := range newTags {
		ec2tags = append(ec2tags, &ec2.Tag{
			Key:   aws.String(TAG_PREFIX + k),
			Value: aws.String(v),
		})
	}

	rii := ec2.RunInstancesInput{
		ImageId:      aws.String(string(imageID)),
		InstanceType: &instanceType.ProviderType,
		MaxCount:     aws.Int64(1),
		MinCount:     aws.Int64(1),
		KeyName:      &instanceSet.ec2config.KeyPairName,

		NetworkInterfaces: []*ec2.InstanceNetworkInterfaceSpecification{
			&ec2.InstanceNetworkInterfaceSpecification{
				AssociatePublicIpAddress: aws.Bool(false),
				DeleteOnTermination:      aws.Bool(true),
				DeviceIndex:              aws.Int64(0),
				Groups:                   []*string{&instanceSet.ec2config.SecurityGroupId},
				SubnetId:                 &instanceSet.ec2config.SubnetId,
			}},
		DisableApiTermination:             aws.Bool(false),
		InstanceInitiatedShutdownBehavior: aws.String("terminate"),
		UserData: aws.String(base64.StdEncoding.EncodeToString([]byte("#!/bin/sh\n" + initCommand + "\n"))),
		TagSpecifications: []*ec2.TagSpecification{
			&ec2.TagSpecification{
				ResourceType: aws.String("instance"),
				Tags:         ec2tags,
			}},
	}

	if instanceType.ExtraScratch > 0 {
		rii.BlockDeviceMappings = []*ec2.BlockDeviceMapping{&ec2.BlockDeviceMapping{
			DeviceName: aws.String("/dev/xvdt"),
			Ebs: &ec2.EbsBlockDevice{
				DeleteOnTermination: aws.Bool(true),
				VolumeSize:          aws.Int64((int64(instanceType.ExtraScratch) / 1000000000) + 1),
				VolumeType:          aws.String("gp2"),
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

	if err != nil {
		return nil, err
	}

	return &ec2Instance{
		provider: instanceSet,
		instance: rsv.Instances[0],
	}, nil
}

func (instanceSet *ec2InstanceSet) Instances(cloud.InstanceTags) (instances []cloud.Instance, err error) {
	dii := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{&ec2.Filter{
			Name:   aws.String("tag:" + ARVADOS_DISPATCH_ID),
			Values: []*string{aws.String(string(instanceSet.dispatcherID))},
		}}}

	for {
		dio, err := instanceSet.client.DescribeInstances(dii)
		if err != nil {
			return nil, err
		}

		for _, rsv := range dio.Reservations {
			for _, inst := range rsv.Instances {
				instances = append(instances, &ec2Instance{instanceSet, inst})
			}
		}
		if dio.NextToken == nil {
			return instances, err
		}
		dii.NextToken = dio.NextToken
	}
}

func (az *ec2InstanceSet) Stop() {
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
	ec2tags := []*ec2.Tag{
		&ec2.Tag{
			Key:   aws.String(ARVADOS_DISPATCH_ID),
			Value: aws.String(string(inst.provider.dispatcherID)),
		},
	}
	for k, v := range newTags {
		ec2tags = append(ec2tags, &ec2.Tag{
			Key:   aws.String(TAG_PREFIX + k),
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
		if strings.HasPrefix(*t.Key, TAG_PREFIX) {
			tags[(*t.Key)[len(TAG_PREFIX):]] = *t.Value
		}
	}

	return tags
}

func (inst *ec2Instance) Destroy() error {
	log.Printf("terminating %v", *inst.instance.InstanceId)
	_, err := inst.provider.client.TerminateInstances(&ec2.TerminateInstancesInput{
		InstanceIds: []*string{inst.instance.InstanceId},
	})
	return err
}

func (inst *ec2Instance) Address() string {
	if inst.instance.PrivateIpAddress != nil {
		return *inst.instance.PrivateIpAddress
	} else {
		return ""
	}
}

func (inst *ec2Instance) RemoteUser() string {
	return inst.provider.ec2config.AdminUsername
}

func (inst *ec2Instance) VerifyHostKey(ssh.PublicKey, *ssh.Client) error {
	return cloud.ErrNotImplemented
}
