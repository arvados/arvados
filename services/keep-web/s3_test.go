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

func (s *IntegrationSuite) s3setup(c *check.C) (*arvados.Client, arvados.Collection, *s3.Bucket) {
	var coll arvados.Collection
	arv := arvados.NewClientFromEnv()
	arv.AuthToken = arvadostest.ActiveToken
	err := arv.RequestAndDecode(&coll, "POST", "arvados/v1/collections", nil, map[string]interface{}{"collection": map[string]interface{}{
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
	bucket := &s3.Bucket{
		S3:   client,
		Name: coll.UUID,
	}
	return arv, coll, bucket
}

func (s *IntegrationSuite) s3teardown(c *check.C, arv *arvados.Client, coll arvados.Collection) {
	err := arv.RequestAndDecode(&coll, "DELETE", "arvados/v1/collections/"+coll.UUID, nil, nil)
	c.Check(err, check.IsNil)
}

func (s *IntegrationSuite) TestS3GetObject(c *check.C) {
	arv, coll, bucket := s.s3setup(c)
	defer s.s3teardown(c, arv, coll)

	rdr, err := bucket.GetReader("emptyfile")
	c.Assert(err, check.IsNil)
	buf, err := ioutil.ReadAll(rdr)
	c.Check(err, check.IsNil)
	c.Check(len(buf), check.Equals, 0)
	err = rdr.Close()
	c.Check(err, check.IsNil)

	rdr, err = bucket.GetReader("missingfile")
	c.Check(err, check.NotNil)

	rdr, err = bucket.GetReader("sailboat.txt")
	c.Check(err, check.IsNil)
	buf, err = ioutil.ReadAll(rdr)
	c.Check(err, check.IsNil)
	c.Check(buf, check.DeepEquals, []byte("⛵\n"))
	err = rdr.Close()
	c.Check(err, check.IsNil)
}

func (s *IntegrationSuite) TestS3PutObjectSuccess(c *check.C) {
	arv, coll, bucket := s.s3setup(c)
	defer s.s3teardown(c, arv, coll)

	for _, trial := range []struct {
		objname string
		size    int
	}{
		{
			objname: "newfile",
			size:    128000000,
		}, {
			objname: "newdir/newfile",
			size:    1 << 26,
		}, {
			objname: "newdir1/newdir2/newfile",
			size:    0,
		},
	} {
		c.Logf("=== %v", trial)

		_, err := bucket.GetReader(trial.objname)
		c.Assert(err, check.NotNil)

		buf := make([]byte, trial.size)
		rand.Read(buf)

		err = bucket.PutReader(trial.objname, bytes.NewReader(buf), int64(len(buf)), "application/octet-stream", s3.Private, s3.Options{})
		c.Check(err, check.IsNil)

		rdr, err := bucket.GetReader(trial.objname)
		if !c.Check(err, check.IsNil) {
			continue
		}
		buf2, err := ioutil.ReadAll(rdr)
		c.Check(err, check.IsNil)
		c.Check(buf2, check.HasLen, len(buf))
		c.Check(buf2, check.DeepEquals, buf)
	}
}

func (s *IntegrationSuite) TestS3PutObjectFailure(c *check.C) {
	arv, coll, bucket := s.s3setup(c)
	defer s.s3teardown(c, arv, coll)

	for _, trial := range []struct {
		objname string
	}{
		{
			objname: "emptyfile/newname", // emptyfile exists, see s3setup()
		}, {
			objname: "emptyfile/", // emptyfile exists, see s3setup()
		}, {
			objname: "emptydir", // dir already exists, see s3setup()
		}, {
			objname: "emptydir/",
		}, {
			objname: "emptydir//",
		}, {
			objname: "newdir/",
		}, {
			objname: "newdir//",
		}, {
			objname: "/",
		}, {
			objname: "//",
		}, {
			objname: "foo//bar",
		}, {
			objname: "",
		},
	} {
		c.Logf("=== %v", trial)

		buf := make([]byte, 1234)
		rand.Read(buf)

		err := bucket.PutReader(trial.objname, bytes.NewReader(buf), int64(len(buf)), "application/octet-stream", s3.Private, s3.Options{})
		if !c.Check(err, check.NotNil, check.Commentf("name %q should be rejected", trial.objname)) {
			continue
		}

		_, err = bucket.GetReader(trial.objname)
		c.Check(err, check.NotNil)
	}
}
