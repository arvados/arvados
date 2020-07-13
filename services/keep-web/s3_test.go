// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"bytes"
	"crypto/rand"
	"io/ioutil"
	"os"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadosclient"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"git.arvados.org/arvados.git/sdk/go/keepclient"
	"github.com/AdRoll/goamz/aws"
	"github.com/AdRoll/goamz/s3"
	check "gopkg.in/check.v1"
)

type s3stage struct {
	arv        *arvados.Client
	proj       arvados.Group
	projbucket *s3.Bucket
	coll       arvados.Collection
	collbucket *s3.Bucket
}

func (s *IntegrationSuite) s3setup(c *check.C) s3stage {
	var proj arvados.Group
	var coll arvados.Collection
	arv := arvados.NewClientFromEnv()
	arv.AuthToken = arvadostest.ActiveToken
	err := arv.RequestAndDecode(&proj, "POST", "arvados/v1/groups", nil, map[string]interface{}{
		"group": map[string]interface{}{
			"group_class": "project",
			"name":        "keep-web s3 test",
		},
		"ensure_unique_name": true,
	})
	c.Assert(err, check.IsNil)
	err = arv.RequestAndDecode(&coll, "POST", "arvados/v1/collections", nil, map[string]interface{}{"collection": map[string]interface{}{
		"owner_uuid":    proj.UUID,
		"name":          "keep-web s3 test collection",
		"manifest_text": ". d41d8cd98f00b204e9800998ecf8427e+0 0:0:emptyfile\n./emptydir d41d8cd98f00b204e9800998ecf8427e+0 0:0:.\n",
	}})
	c.Assert(err, check.IsNil)
	ac, err := arvadosclient.New(arv)
	c.Assert(err, check.IsNil)
	kc, err := keepclient.MakeKeepClient(ac)
	c.Assert(err, check.IsNil)
	fs, err := coll.FileSystem(arv, kc)
	c.Assert(err, check.IsNil)
	f, err := fs.OpenFile("sailboat.txt", os.O_CREATE|os.O_WRONLY, 0644)
	c.Assert(err, check.IsNil)
	_, err = f.Write([]byte("⛵\n"))
	c.Assert(err, check.IsNil)
	err = f.Close()
	c.Assert(err, check.IsNil)
	err = fs.Sync()
	c.Assert(err, check.IsNil)

	auth := aws.NewAuth(arvadostest.ActiveTokenV2, arvadostest.ActiveTokenV2, "", time.Now().Add(time.Hour))
	region := aws.Region{
		Name:       s.testServer.Addr,
		S3Endpoint: "http://" + s.testServer.Addr,
	}
	client := s3.New(*auth, region)
	return s3stage{
		arv:  arv,
		proj: proj,
		projbucket: &s3.Bucket{
			S3:   client,
			Name: proj.UUID,
		},
		coll: coll,
		collbucket: &s3.Bucket{
			S3:   client,
			Name: coll.UUID,
		},
	}
}

func (stage s3stage) teardown(c *check.C) {
	if stage.coll.UUID != "" {
		err := stage.arv.RequestAndDecode(&stage.coll, "DELETE", "arvados/v1/collections/"+stage.coll.UUID, nil, nil)
		c.Check(err, check.IsNil)
	}
}

func (s *IntegrationSuite) TestS3CollectionGetObject(c *check.C) {
	stage := s.s3setup(c)
	defer stage.teardown(c)
	s.testS3GetObject(c, stage.collbucket, "")
}
func (s *IntegrationSuite) TestS3ProjectGetObject(c *check.C) {
	stage := s.s3setup(c)
	defer stage.teardown(c)
	s.testS3GetObject(c, stage.projbucket, stage.coll.Name+"/")
}
func (s *IntegrationSuite) testS3GetObject(c *check.C, bucket *s3.Bucket, prefix string) {
	rdr, err := bucket.GetReader(prefix + "emptyfile")
	c.Assert(err, check.IsNil)
	buf, err := ioutil.ReadAll(rdr)
	c.Check(err, check.IsNil)
	c.Check(len(buf), check.Equals, 0)
	err = rdr.Close()
	c.Check(err, check.IsNil)

	rdr, err = bucket.GetReader(prefix + "missingfile")
	c.Check(err, check.NotNil)

	rdr, err = bucket.GetReader(prefix + "sailboat.txt")
	c.Assert(err, check.IsNil)
	buf, err = ioutil.ReadAll(rdr)
	c.Check(err, check.IsNil)
	c.Check(buf, check.DeepEquals, []byte("⛵\n"))
	err = rdr.Close()
	c.Check(err, check.IsNil)
}

func (s *IntegrationSuite) TestS3CollectionPutObjectSuccess(c *check.C) {
	stage := s.s3setup(c)
	defer stage.teardown(c)
	s.testS3PutObjectSuccess(c, stage.collbucket, "")
}
func (s *IntegrationSuite) TestS3ProjectPutObjectSuccess(c *check.C) {
	stage := s.s3setup(c)
	defer stage.teardown(c)
	s.testS3PutObjectSuccess(c, stage.projbucket, stage.coll.Name+"/")
}
func (s *IntegrationSuite) testS3PutObjectSuccess(c *check.C, bucket *s3.Bucket, prefix string) {
	for _, trial := range []struct {
		path string
		size int
	}{
		{
			path: "newfile",
			size: 128000000,
		}, {
			path: "newdir/newfile",
			size: 1 << 26,
		}, {
			path: "newdir1/newdir2/newfile",
			size: 0,
		},
	} {
		c.Logf("=== %v", trial)

		objname := prefix + trial.path

		_, err := bucket.GetReader(objname)
		c.Assert(err, check.NotNil)

		buf := make([]byte, trial.size)
		rand.Read(buf)

		err = bucket.PutReader(objname, bytes.NewReader(buf), int64(len(buf)), "application/octet-stream", s3.Private, s3.Options{})
		c.Check(err, check.IsNil)

		rdr, err := bucket.GetReader(objname)
		if !c.Check(err, check.IsNil) {
			continue
		}
		buf2, err := ioutil.ReadAll(rdr)
		c.Check(err, check.IsNil)
		c.Check(buf2, check.HasLen, len(buf))
		c.Check(buf2, check.DeepEquals, buf)
	}
}

func (s *IntegrationSuite) TestS3CollectionPutObjectFailure(c *check.C) {
	stage := s.s3setup(c)
	defer stage.teardown(c)
	s.testS3PutObjectFailure(c, stage.collbucket, "")
}
func (s *IntegrationSuite) TestS3ProjectPutObjectFailure(c *check.C) {
	stage := s.s3setup(c)
	defer stage.teardown(c)
	s.testS3PutObjectFailure(c, stage.projbucket, stage.coll.Name+"/")
}
func (s *IntegrationSuite) testS3PutObjectFailure(c *check.C, bucket *s3.Bucket, prefix string) {
	for _, trial := range []struct {
		path string
	}{
		{
			path: "emptyfile/newname", // emptyfile exists, see s3setup()
		}, {
			path: "emptyfile/", // emptyfile exists, see s3setup()
		}, {
			path: "emptydir", // dir already exists, see s3setup()
		}, {
			path: "emptydir/",
		}, {
			path: "emptydir//",
		}, {
			path: "newdir/",
		}, {
			path: "newdir//",
		}, {
			path: "/",
		}, {
			path: "//",
		}, {
			path: "foo//bar",
		}, {
			path: "",
		},
	} {
		c.Logf("=== %v", trial)

		objname := prefix + trial.path

		buf := make([]byte, 1234)
		rand.Read(buf)

		err := bucket.PutReader(objname, bytes.NewReader(buf), int64(len(buf)), "application/octet-stream", s3.Private, s3.Options{})
		if !c.Check(err, check.NotNil, check.Commentf("name %q should be rejected", objname)) {
			continue
		}

		_, err = bucket.GetReader(objname)
		c.Check(err, check.NotNil)
	}
}
