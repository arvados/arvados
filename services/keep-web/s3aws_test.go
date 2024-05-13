// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepweb

import (
	"bytes"
	"context"
	"errors"
	"io/ioutil"

	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	check "gopkg.in/check.v1"
)

func (s *IntegrationSuite) TestS3AWSSDK(c *check.C) {
	stage := s.s3setup(c)
	defer stage.teardown(c)

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		func(o *config.LoadOptions) error {
			o.Credentials = credentials.StaticCredentialsProvider{
				Value: aws.Credentials{
					AccessKeyID:     arvadostest.ActiveTokenUUID,
					SecretAccessKey: arvadostest.ActiveToken,
					Source:          "test suite configuration",
				},
			}
			o.EndpointResolverWithOptions = aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				if service == "S3" {
					return aws.Endpoint{
						URL:               s.testServer.URL,
						HostnameImmutable: true,
						SigningRegion:     "test-region",
						Source:            aws.EndpointSourceCustom,
					}, nil
				}
				// else, use default
				return aws.Endpoint{}, &aws.EndpointNotFoundError{Err: errors.New("endpoint not overridden")}
			})
			return nil
		})
	c.Assert(err, check.IsNil)
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.Region = "test-region"
		o.UsePathStyle = true
	})
	resp, err := client.ListObjectsV2(context.Background(), &s3.ListObjectsV2Input{
		Bucket:            aws.String(arvadostest.FooCollection),
		MaxKeys:           aws.Int32(100),
		Prefix:            aws.String(""),
		ContinuationToken: nil,
	})
	c.Assert(err, check.IsNil)
	c.Check(resp.Contents, check.HasLen, 1)
	for _, key := range resp.Contents {
		c.Check(*key.Key, check.Equals, "foo")
	}

	p := make([]byte, 100000000)
	for i := range p {
		p[i] = byte('a')
	}
	_, err = client.PutObject(context.Background(), &s3.PutObjectInput{
		Body:        bytes.NewReader(p),
		Bucket:      aws.String(stage.collbucket.Name),
		ContentType: aws.String("application/octet-stream"),
		Key:         aws.String("aaaa"),
	})
	c.Assert(err, check.IsNil)

	getresp, err := client.GetObject(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(stage.collbucket.Name),
		Key:    aws.String("aaaa"),
	})
	c.Assert(err, check.IsNil)
	getdata, err := ioutil.ReadAll(getresp.Body)
	c.Assert(err, check.IsNil)
	c.Check(bytes.Equal(getdata, p), check.Equals, true)
}
