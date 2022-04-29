// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepweb

import (
	"bytes"
	"context"
	"io/ioutil"

	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/defaults"
	"github.com/aws/aws-sdk-go-v2/aws/ec2metadata"
	"github.com/aws/aws-sdk-go-v2/aws/ec2rolecreds"
	"github.com/aws/aws-sdk-go-v2/aws/endpoints"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	check "gopkg.in/check.v1"
)

func (s *IntegrationSuite) TestS3AWSSDK(c *check.C) {
	stage := s.s3setup(c)
	defer stage.teardown(c)

	cfg := defaults.Config()
	cfg.Credentials = aws.NewChainProvider([]aws.CredentialsProvider{
		aws.NewStaticCredentialsProvider(arvadostest.ActiveTokenUUID, arvadostest.ActiveToken, ""),
		ec2rolecreds.New(ec2metadata.New(cfg)),
	})
	cfg.EndpointResolver = aws.EndpointResolverFunc(func(service, region string) (aws.Endpoint, error) {
		if service == "s3" {
			return aws.Endpoint{
				URL:           s.testServer.URL,
				SigningRegion: "custom-signing-region",
			}, nil
		}
		return endpoints.NewDefaultResolver().ResolveEndpoint(service, region)
	})
	client := s3.New(cfg)
	client.ForcePathStyle = true
	listreq := client.ListObjectsV2Request(&s3.ListObjectsV2Input{
		Bucket:            aws.String(arvadostest.FooCollection),
		MaxKeys:           aws.Int64(100),
		Prefix:            aws.String(""),
		ContinuationToken: nil,
	})
	resp, err := listreq.Send(context.Background())
	c.Assert(err, check.IsNil)
	c.Check(resp.Contents, check.HasLen, 1)
	for _, key := range resp.Contents {
		c.Check(*key.Key, check.Equals, "foo")
	}

	p := make([]byte, 100000000)
	for i := range p {
		p[i] = byte('a')
	}
	putreq := client.PutObjectRequest(&s3.PutObjectInput{
		Body:        bytes.NewReader(p),
		Bucket:      aws.String(stage.collbucket.Name),
		ContentType: aws.String("application/octet-stream"),
		Key:         aws.String("aaaa"),
	})
	_, err = putreq.Send(context.Background())
	c.Assert(err, check.IsNil)

	getreq := client.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(stage.collbucket.Name),
		Key:    aws.String("aaaa"),
	})
	getresp, err := getreq.Send(context.Background())
	c.Assert(err, check.IsNil)
	getdata, err := ioutil.ReadAll(getresp.Body)
	c.Assert(err, check.IsNil)
	c.Check(bytes.Equal(getdata, p), check.Equals, true)
}
