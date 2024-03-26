// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepweb

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"mime"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadosclient"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"git.arvados.org/arvados.git/sdk/go/keepclient"
	"github.com/AdRoll/goamz/aws"
	"github.com/AdRoll/goamz/s3"
	aws_aws "github.com/aws/aws-sdk-go/aws"
	aws_credentials "github.com/aws/aws-sdk-go/aws/credentials"
	aws_session "github.com/aws/aws-sdk-go/aws/session"
	aws_s3 "github.com/aws/aws-sdk-go/service/s3"
	check "gopkg.in/check.v1"
)

type s3stage struct {
	arv        *arvados.Client
	ac         *arvadosclient.ArvadosClient
	kc         *keepclient.KeepClient
	proj       arvados.Group
	projbucket *s3.Bucket
	subproj    arvados.Group
	coll       arvados.Collection
	collbucket *s3.Bucket
}

func (s *IntegrationSuite) s3setup(c *check.C) s3stage {
	var proj, subproj arvados.Group
	var coll arvados.Collection
	arv := arvados.NewClientFromEnv()
	arv.AuthToken = arvadostest.ActiveToken
	err := arv.RequestAndDecode(&proj, "POST", "arvados/v1/groups", nil, map[string]interface{}{
		"group": map[string]interface{}{
			"group_class": "project",
			"name":        "keep-web s3 test",
			"properties": map[string]interface{}{
				"project-properties-key": "project properties value",
			},
		},
		"ensure_unique_name": true,
	})
	c.Assert(err, check.IsNil)
	err = arv.RequestAndDecode(&subproj, "POST", "arvados/v1/groups", nil, map[string]interface{}{
		"group": map[string]interface{}{
			"owner_uuid":  proj.UUID,
			"group_class": "project",
			"name":        "keep-web s3 test subproject",
			"properties": map[string]interface{}{
				"subproject_properties_key": "subproject properties value",
				"invalid header key":        "this value will not be returned because key contains spaces",
			},
		},
	})
	c.Assert(err, check.IsNil)
	err = arv.RequestAndDecode(&coll, "POST", "arvados/v1/collections", nil, map[string]interface{}{"collection": map[string]interface{}{
		"owner_uuid":    proj.UUID,
		"name":          "keep-web s3 test collection",
		"manifest_text": ". d41d8cd98f00b204e9800998ecf8427e+0 0:0:emptyfile\n./emptydir d41d8cd98f00b204e9800998ecf8427e+0 0:0:.\n",
		"properties": map[string]interface{}{
			"string":   "string value",
			"array":    []string{"element1", "element2"},
			"object":   map[string]interface{}{"key": map[string]interface{}{"key2": "value⛵"}},
			"nonascii": "⛵",
			"newline":  "foo\r\nX-Bad: header",
			// This key cannot be expressed as a MIME
			// header key, so it will be silently skipped
			// (see "Inject" in PropertiesAsMetadata test)
			"a: a\r\nInject": "bogus",
		},
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
	err = arv.RequestAndDecode(&coll, "GET", "arvados/v1/collections/"+coll.UUID, nil, nil)
	c.Assert(err, check.IsNil)

	auth := aws.NewAuth(arvadostest.ActiveTokenUUID, arvadostest.ActiveToken, "", time.Now().Add(time.Hour))
	region := aws.Region{
		Name:       "zzzzz",
		S3Endpoint: s.testServer.URL,
	}
	client := s3.New(*auth, region)
	client.Signature = aws.V4Signature
	return s3stage{
		arv:  arv,
		ac:   ac,
		kc:   kc,
		proj: proj,
		projbucket: &s3.Bucket{
			S3:   client,
			Name: proj.UUID,
		},
		subproj: subproj,
		coll:    coll,
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
	if stage.proj.UUID != "" {
		err := stage.arv.RequestAndDecode(&stage.proj, "DELETE", "arvados/v1/groups/"+stage.proj.UUID, nil, nil)
		c.Check(err, check.IsNil)
	}
}

func (s *IntegrationSuite) TestS3Signatures(c *check.C) {
	stage := s.s3setup(c)
	defer stage.teardown(c)

	bucket := stage.collbucket
	for _, trial := range []struct {
		success   bool
		signature int
		accesskey string
		secretkey string
	}{
		{true, aws.V2Signature, arvadostest.ActiveToken, "none"},
		{true, aws.V2Signature, url.QueryEscape(arvadostest.ActiveTokenV2), "none"},
		{true, aws.V2Signature, strings.Replace(arvadostest.ActiveTokenV2, "/", "_", -1), "none"},
		{false, aws.V2Signature, "none", "none"},
		{false, aws.V2Signature, "none", arvadostest.ActiveToken},

		{true, aws.V4Signature, arvadostest.ActiveTokenUUID, arvadostest.ActiveToken},
		{true, aws.V4Signature, arvadostest.ActiveToken, arvadostest.ActiveToken},
		{true, aws.V4Signature, url.QueryEscape(arvadostest.ActiveTokenV2), url.QueryEscape(arvadostest.ActiveTokenV2)},
		{true, aws.V4Signature, strings.Replace(arvadostest.ActiveTokenV2, "/", "_", -1), strings.Replace(arvadostest.ActiveTokenV2, "/", "_", -1)},
		{false, aws.V4Signature, arvadostest.ActiveToken, ""},
		{false, aws.V4Signature, arvadostest.ActiveToken, "none"},
		{false, aws.V4Signature, "none", arvadostest.ActiveToken},
		{false, aws.V4Signature, "none", "none"},
	} {
		c.Logf("%#v", trial)
		bucket.S3.Auth = *(aws.NewAuth(trial.accesskey, trial.secretkey, "", time.Now().Add(time.Hour)))
		bucket.S3.Signature = trial.signature
		_, err := bucket.GetReader("emptyfile")
		if trial.success {
			c.Check(err, check.IsNil)
		} else {
			c.Check(err, check.NotNil)
		}
	}
}

func (s *IntegrationSuite) TestS3HeadBucket(c *check.C) {
	stage := s.s3setup(c)
	defer stage.teardown(c)

	for _, bucket := range []*s3.Bucket{stage.collbucket, stage.projbucket} {
		c.Logf("bucket %s", bucket.Name)
		exists, err := bucket.Exists("")
		c.Check(err, check.IsNil)
		c.Check(exists, check.Equals, true)
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

	// GetObject
	rdr, err = bucket.GetReader(prefix + "missingfile")
	c.Check(err.(*s3.Error).StatusCode, check.Equals, 404)
	c.Check(err.(*s3.Error).Code, check.Equals, `NoSuchKey`)
	c.Check(err, check.ErrorMatches, `The specified key does not exist.`)

	// HeadObject
	exists, err := bucket.Exists(prefix + "missingfile")
	c.Check(err, check.IsNil)
	c.Check(exists, check.Equals, false)

	// GetObject
	rdr, err = bucket.GetReader(prefix + "sailboat.txt")
	c.Assert(err, check.IsNil)
	buf, err = ioutil.ReadAll(rdr)
	c.Check(err, check.IsNil)
	c.Check(buf, check.DeepEquals, []byte("⛵\n"))
	err = rdr.Close()
	c.Check(err, check.IsNil)

	// HeadObject
	resp, err := bucket.Head(prefix+"sailboat.txt", nil)
	c.Check(err, check.IsNil)
	c.Check(resp.StatusCode, check.Equals, http.StatusOK)
	c.Check(resp.ContentLength, check.Equals, int64(4))

	// HeadObject with superfluous leading slashes
	exists, err = bucket.Exists(prefix + "//sailboat.txt")
	c.Check(err, check.IsNil)
	c.Check(exists, check.Equals, true)
}

func (s *IntegrationSuite) checkMetaEquals(c *check.C, hdr http.Header, expect map[string]string) {
	got := map[string]string{}
	for hk, hv := range hdr {
		if k := strings.TrimPrefix(hk, "X-Amz-Meta-"); k != hk && len(hv) == 1 {
			got[k] = hv[0]
		}
	}
	c.Check(got, check.DeepEquals, expect)
}

func (s *IntegrationSuite) TestS3PropertiesAsMetadata(c *check.C) {
	stage := s.s3setup(c)
	defer stage.teardown(c)

	expectCollectionTags := map[string]string{
		"String":   "string value",
		"Array":    `["element1","element2"]`,
		"Object":   mime.BEncoding.Encode("UTF-8", `{"key":{"key2":"value⛵"}}`),
		"Nonascii": "=?UTF-8?b?4pu1?=",
		"Newline":  mime.BEncoding.Encode("UTF-8", "foo\r\nX-Bad: header"),
	}
	expectSubprojectTags := map[string]string{
		"Subproject_properties_key": "subproject properties value",
	}
	expectProjectTags := map[string]string{
		"Project-Properties-Key": "project properties value",
	}

	c.Log("HEAD object with metadata from collection")
	resp, err := stage.collbucket.Head("sailboat.txt", nil)
	c.Assert(err, check.IsNil)
	s.checkMetaEquals(c, resp.Header, expectCollectionTags)

	c.Log("GET object with metadata from collection")
	rdr, hdr, err := stage.collbucket.GetReaderWithHeaders("sailboat.txt")
	c.Assert(err, check.IsNil)
	content, err := ioutil.ReadAll(rdr)
	c.Check(err, check.IsNil)
	rdr.Close()
	c.Check(content, check.HasLen, 4)
	s.checkMetaEquals(c, hdr, expectCollectionTags)
	c.Check(hdr["Inject"], check.IsNil)

	c.Log("HEAD bucket with metadata from collection")
	resp, err = stage.collbucket.Head("/", nil)
	c.Assert(err, check.IsNil)
	s.checkMetaEquals(c, resp.Header, expectCollectionTags)

	c.Log("HEAD directory placeholder with metadata from collection")
	resp, err = stage.projbucket.Head("keep-web s3 test collection/", nil)
	c.Assert(err, check.IsNil)
	s.checkMetaEquals(c, resp.Header, expectCollectionTags)

	c.Log("HEAD file with metadata from collection")
	resp, err = stage.projbucket.Head("keep-web s3 test collection/sailboat.txt", nil)
	c.Assert(err, check.IsNil)
	s.checkMetaEquals(c, resp.Header, expectCollectionTags)

	c.Log("HEAD directory placeholder with metadata from subproject")
	resp, err = stage.projbucket.Head("keep-web s3 test subproject/", nil)
	c.Assert(err, check.IsNil)
	s.checkMetaEquals(c, resp.Header, expectSubprojectTags)

	c.Log("HEAD bucket with metadata from project")
	resp, err = stage.projbucket.Head("/", nil)
	c.Assert(err, check.IsNil)
	s.checkMetaEquals(c, resp.Header, expectProjectTags)
}

func (s *IntegrationSuite) TestS3CollectionPutObjectSuccess(c *check.C) {
	stage := s.s3setup(c)
	defer stage.teardown(c)
	s.testS3PutObjectSuccess(c, stage.collbucket, "", stage.coll.UUID)
}
func (s *IntegrationSuite) TestS3ProjectPutObjectSuccess(c *check.C) {
	stage := s.s3setup(c)
	defer stage.teardown(c)
	s.testS3PutObjectSuccess(c, stage.projbucket, stage.coll.Name+"/", stage.coll.UUID)
}
func (s *IntegrationSuite) testS3PutObjectSuccess(c *check.C, bucket *s3.Bucket, prefix string, collUUID string) {
	// We insert a delay between test cases to ensure we exercise
	// rollover of expired sessions.
	sleep := time.Second / 100
	s.handler.Cluster.Collections.WebDAVCache.TTL = arvados.Duration(sleep * 3)

	for _, trial := range []struct {
		path        string
		size        int
		contentType string
	}{
		{
			path:        "newfile",
			size:        128000000,
			contentType: "application/octet-stream",
		}, {
			path:        "newdir/newfile",
			size:        1 << 26,
			contentType: "application/octet-stream",
		}, {
			path:        "/aaa",
			size:        2,
			contentType: "application/octet-stream",
		}, {
			path:        "//bbb",
			size:        2,
			contentType: "application/octet-stream",
		}, {
			path:        "ccc//",
			size:        0,
			contentType: "application/x-directory",
		}, {
			path:        "newdir1/newdir2/newfile",
			size:        0,
			contentType: "application/octet-stream",
		}, {
			path:        "newdir1/newdir2/newdir3/",
			size:        0,
			contentType: "application/x-directory",
		},
	} {
		time.Sleep(sleep)
		c.Logf("=== %v", trial)

		objname := prefix + trial.path

		_, err := bucket.GetReader(objname)
		if !c.Check(err, check.NotNil) {
			continue
		}
		c.Check(err.(*s3.Error).StatusCode, check.Equals, http.StatusNotFound)
		c.Check(err.(*s3.Error).Code, check.Equals, `NoSuchKey`)
		if !c.Check(err, check.ErrorMatches, `The specified key does not exist.`) {
			continue
		}

		buf := make([]byte, trial.size)
		rand.Read(buf)

		err = bucket.PutReader(objname, bytes.NewReader(buf), int64(len(buf)), trial.contentType, s3.Private, s3.Options{})
		c.Check(err, check.IsNil)

		rdr, err := bucket.GetReader(objname)
		if strings.HasSuffix(trial.path, "/") && !s.handler.Cluster.Collections.S3FolderObjects {
			c.Check(err, check.NotNil)
			continue
		} else if !c.Check(err, check.IsNil) {
			continue
		}
		buf2, err := ioutil.ReadAll(rdr)
		c.Check(err, check.IsNil)
		c.Check(buf2, check.HasLen, len(buf))
		c.Check(bytes.Equal(buf, buf2), check.Equals, true)

		// Check that the change is immediately visible via
		// (non-S3) webdav request.
		_, resp := s.do("GET", "http://"+collUUID+".keep-web.example/"+trial.path, arvadostest.ActiveTokenV2, nil)
		c.Check(resp.Code, check.Equals, http.StatusOK)
		if !strings.HasSuffix(trial.path, "/") {
			c.Check(resp.Body.Len(), check.Equals, trial.size)
		}
	}
}

func (s *IntegrationSuite) TestS3ProjectPutObjectNotSupported(c *check.C) {
	stage := s.s3setup(c)
	defer stage.teardown(c)
	bucket := stage.projbucket

	for _, trial := range []struct {
		path        string
		size        int
		contentType string
	}{
		{
			path:        "newfile",
			size:        1234,
			contentType: "application/octet-stream",
		}, {
			path:        "newdir/newfile",
			size:        1234,
			contentType: "application/octet-stream",
		}, {
			path:        "newdir2/",
			size:        0,
			contentType: "application/x-directory",
		},
	} {
		c.Logf("=== %v", trial)

		_, err := bucket.GetReader(trial.path)
		c.Check(err.(*s3.Error).StatusCode, check.Equals, 404)
		c.Check(err.(*s3.Error).Code, check.Equals, `NoSuchKey`)
		c.Assert(err, check.ErrorMatches, `The specified key does not exist.`)

		buf := make([]byte, trial.size)
		rand.Read(buf)

		err = bucket.PutReader(trial.path, bytes.NewReader(buf), int64(len(buf)), trial.contentType, s3.Private, s3.Options{})
		c.Check(err.(*s3.Error).StatusCode, check.Equals, 400)
		c.Check(err.(*s3.Error).Code, check.Equals, `InvalidArgument`)
		c.Check(err, check.ErrorMatches, `(mkdir "/by_id/zzzzz-j7d0g-[a-z0-9]{15}/newdir2?"|open "/zzzzz-j7d0g-[a-z0-9]{15}/newfile") failed: invalid (argument|operation)`)

		_, err = bucket.GetReader(trial.path)
		c.Check(err.(*s3.Error).StatusCode, check.Equals, 404)
		c.Check(err.(*s3.Error).Code, check.Equals, `NoSuchKey`)
		c.Assert(err, check.ErrorMatches, `The specified key does not exist.`)
	}
}

func (s *IntegrationSuite) TestS3CollectionDeleteObject(c *check.C) {
	stage := s.s3setup(c)
	defer stage.teardown(c)
	s.testS3DeleteObject(c, stage.collbucket, "")
}
func (s *IntegrationSuite) TestS3ProjectDeleteObject(c *check.C) {
	stage := s.s3setup(c)
	defer stage.teardown(c)
	s.testS3DeleteObject(c, stage.projbucket, stage.coll.Name+"/")
}
func (s *IntegrationSuite) testS3DeleteObject(c *check.C, bucket *s3.Bucket, prefix string) {
	s.handler.Cluster.Collections.S3FolderObjects = true
	for _, trial := range []struct {
		path string
	}{
		{"/"},
		{"nonexistentfile"},
		{"emptyfile"},
		{"sailboat.txt"},
		{"sailboat.txt/"},
		{"emptydir"},
		{"emptydir/"},
	} {
		objname := prefix + trial.path
		comment := check.Commentf("objname %q", objname)

		err := bucket.Del(objname)
		if trial.path == "/" {
			c.Check(err, check.NotNil)
			continue
		}
		c.Check(err, check.IsNil, comment)
		_, err = bucket.GetReader(objname)
		c.Check(err, check.NotNil, comment)
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
	s.handler.Cluster.Collections.S3FolderObjects = false

	var wg sync.WaitGroup
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
			path: "",
		},
	} {
		trial := trial
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.Logf("=== %v", trial)

			objname := prefix + trial.path

			buf := make([]byte, 1234)
			rand.Read(buf)

			err := bucket.PutReader(objname, bytes.NewReader(buf), int64(len(buf)), "application/octet-stream", s3.Private, s3.Options{})
			if !c.Check(err, check.ErrorMatches, `(invalid object name.*|open ".*" failed.*|object name conflicts with existing object|Missing object name in PUT request.)`, check.Commentf("PUT %q should fail", objname)) {
				return
			}

			if objname != "" && objname != "/" {
				_, err = bucket.GetReader(objname)
				c.Check(err.(*s3.Error).StatusCode, check.Equals, 404)
				c.Check(err.(*s3.Error).Code, check.Equals, `NoSuchKey`)
				c.Check(err, check.ErrorMatches, `The specified key does not exist.`, check.Commentf("GET %q should return 404", objname))
			}
		}()
	}
	wg.Wait()
}

func (stage *s3stage) writeBigDirs(c *check.C, dirs int, filesPerDir int) {
	fs, err := stage.coll.FileSystem(stage.arv, stage.kc)
	c.Assert(err, check.IsNil)
	for d := 0; d < dirs; d++ {
		dir := fmt.Sprintf("dir%d", d)
		c.Assert(fs.Mkdir(dir, 0755), check.IsNil)
		for i := 0; i < filesPerDir; i++ {
			f, err := fs.OpenFile(fmt.Sprintf("%s/file%d.txt", dir, i), os.O_CREATE|os.O_WRONLY, 0644)
			c.Assert(err, check.IsNil)
			c.Assert(f.Close(), check.IsNil)
		}
	}
	c.Assert(fs.Sync(), check.IsNil)
}

func (s *IntegrationSuite) sign(c *check.C, req *http.Request, key, secret string) {
	scope := "20200202/zzzzz/service/aws4_request"
	signedHeaders := "date"
	req.Header.Set("Date", time.Now().UTC().Format(time.RFC1123))
	stringToSign, err := s3stringToSign(s3SignAlgorithm, scope, signedHeaders, req)
	c.Assert(err, check.IsNil)
	sig, err := s3signature(secret, scope, signedHeaders, stringToSign)
	c.Assert(err, check.IsNil)
	req.Header.Set("Authorization", s3SignAlgorithm+" Credential="+key+"/"+scope+", SignedHeaders="+signedHeaders+", Signature="+sig)
}

func (s *IntegrationSuite) TestS3VirtualHostStyleRequests(c *check.C) {
	stage := s.s3setup(c)
	defer stage.teardown(c)
	for _, trial := range []struct {
		url            string
		method         string
		body           string
		responseCode   int
		responseRegexp []string
	}{
		{
			url:            "https://" + stage.collbucket.Name + ".example.com/",
			method:         "GET",
			responseCode:   http.StatusOK,
			responseRegexp: []string{`(?ms).*sailboat\.txt.*`},
		},
		{
			url:            "https://" + strings.Replace(stage.coll.PortableDataHash, "+", "-", -1) + ".example.com/",
			method:         "GET",
			responseCode:   http.StatusOK,
			responseRegexp: []string{`(?ms).*sailboat\.txt.*`},
		},
		{
			url:            "https://" + stage.projbucket.Name + ".example.com/?prefix=" + stage.coll.Name + "/&delimiter=/",
			method:         "GET",
			responseCode:   http.StatusOK,
			responseRegexp: []string{`(?ms).*sailboat\.txt.*`},
		},
		{
			url:            "https://" + stage.projbucket.Name + ".example.com/" + stage.coll.Name + "/sailboat.txt",
			method:         "GET",
			responseCode:   http.StatusOK,
			responseRegexp: []string{`⛵\n`},
		},
		{
			url:          "https://" + stage.projbucket.Name + ".example.com/" + stage.coll.Name + "/beep",
			method:       "PUT",
			body:         "boop",
			responseCode: http.StatusOK,
		},
		{
			url:            "https://" + stage.projbucket.Name + ".example.com/" + stage.coll.Name + "/beep",
			method:         "GET",
			responseCode:   http.StatusOK,
			responseRegexp: []string{`boop`},
		},
		{
			url:          "https://" + stage.projbucket.Name + ".example.com/" + stage.coll.Name + "//boop",
			method:       "GET",
			responseCode: http.StatusNotFound,
		},
		{
			url:          "https://" + stage.projbucket.Name + ".example.com/" + stage.coll.Name + "//boop",
			method:       "PUT",
			body:         "boop",
			responseCode: http.StatusOK,
		},
		{
			url:            "https://" + stage.projbucket.Name + ".example.com/" + stage.coll.Name + "//boop",
			method:         "GET",
			responseCode:   http.StatusOK,
			responseRegexp: []string{`boop`},
		},
	} {
		url, err := url.Parse(trial.url)
		c.Assert(err, check.IsNil)
		req, err := http.NewRequest(trial.method, url.String(), bytes.NewReader([]byte(trial.body)))
		c.Assert(err, check.IsNil)
		s.sign(c, req, arvadostest.ActiveTokenUUID, arvadostest.ActiveToken)
		rr := httptest.NewRecorder()
		s.handler.ServeHTTP(rr, req)
		resp := rr.Result()
		c.Check(resp.StatusCode, check.Equals, trial.responseCode)
		body, err := ioutil.ReadAll(resp.Body)
		c.Assert(err, check.IsNil)
		for _, re := range trial.responseRegexp {
			c.Check(string(body), check.Matches, re)
		}
	}
}

func (s *IntegrationSuite) TestS3NormalizeURIForSignature(c *check.C) {
	stage := s.s3setup(c)
	defer stage.teardown(c)
	for _, trial := range []struct {
		rawPath        string
		normalizedPath string
	}{
		{"/foo", "/foo"},                           // boring case
		{"/foo%5fbar", "/foo_bar"},                 // _ must not be escaped
		{"/foo%2fbar", "/foo/bar"},                 // / must not be escaped
		{"/(foo)/[];,", "/%28foo%29/%5B%5D%3B%2C"}, // ()[];, must be escaped
		{"/foo%5bbar", "/foo%5Bbar"},               // %XX must be uppercase
		{"//foo///.bar", "/foo/.bar"},              // "//" and "///" must be squashed to "/"
	} {
		c.Logf("trial %q", trial)

		date := time.Now().UTC().Format("20060102T150405Z")
		scope := "20200202/zzzzz/S3/aws4_request"
		canonicalRequest := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s", "GET", trial.normalizedPath, "", "host:host.example.com\n", "host", "")
		c.Logf("canonicalRequest %q", canonicalRequest)
		expect := fmt.Sprintf("%s\n%s\n%s\n%s", s3SignAlgorithm, date, scope, hashdigest(sha256.New(), canonicalRequest))
		c.Logf("expected stringToSign %q", expect)

		req, err := http.NewRequest("GET", "https://host.example.com"+trial.rawPath, nil)
		req.Header.Set("X-Amz-Date", date)
		req.Host = "host.example.com"
		c.Assert(err, check.IsNil)

		obtained, err := s3stringToSign(s3SignAlgorithm, scope, "host", req)
		if !c.Check(err, check.IsNil) {
			continue
		}
		c.Check(obtained, check.Equals, expect)
	}
}

func (s *IntegrationSuite) TestS3GetBucketLocation(c *check.C) {
	stage := s.s3setup(c)
	defer stage.teardown(c)
	for _, bucket := range []*s3.Bucket{stage.collbucket, stage.projbucket} {
		req, err := http.NewRequest("GET", bucket.URL("/"), nil)
		c.Check(err, check.IsNil)
		req.Header.Set("Authorization", "AWS "+arvadostest.ActiveTokenV2+":none")
		req.URL.RawQuery = "location"
		resp, err := http.DefaultClient.Do(req)
		c.Assert(err, check.IsNil)
		c.Check(resp.Header.Get("Content-Type"), check.Equals, "application/xml")
		buf, err := ioutil.ReadAll(resp.Body)
		c.Assert(err, check.IsNil)
		c.Check(string(buf), check.Equals, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<LocationConstraint><LocationConstraint xmlns=\"http://s3.amazonaws.com/doc/2006-03-01/\">zzzzz</LocationConstraint></LocationConstraint>\n")
	}
}

func (s *IntegrationSuite) TestS3GetBucketVersioning(c *check.C) {
	stage := s.s3setup(c)
	defer stage.teardown(c)
	for _, bucket := range []*s3.Bucket{stage.collbucket, stage.projbucket} {
		req, err := http.NewRequest("GET", bucket.URL("/"), nil)
		c.Check(err, check.IsNil)
		req.Header.Set("Authorization", "AWS "+arvadostest.ActiveTokenV2+":none")
		req.URL.RawQuery = "versioning"
		resp, err := http.DefaultClient.Do(req)
		c.Assert(err, check.IsNil)
		c.Check(resp.Header.Get("Content-Type"), check.Equals, "application/xml")
		buf, err := ioutil.ReadAll(resp.Body)
		c.Assert(err, check.IsNil)
		c.Check(string(buf), check.Equals, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<VersioningConfiguration xmlns=\"http://s3.amazonaws.com/doc/2006-03-01/\"/>\n")
	}
}

func (s *IntegrationSuite) TestS3UnsupportedAPIs(c *check.C) {
	stage := s.s3setup(c)
	defer stage.teardown(c)
	for _, trial := range []struct {
		method   string
		path     string
		rawquery string
	}{
		{"GET", "/", "acl&versionId=1234"},    // GetBucketAcl
		{"GET", "/foo", "acl&versionId=1234"}, // GetObjectAcl
		{"PUT", "/", "acl"},                   // PutBucketAcl
		{"PUT", "/foo", "acl"},                // PutObjectAcl
		{"DELETE", "/", "tagging"},            // DeleteBucketTagging
		{"DELETE", "/foo", "tagging"},         // DeleteObjectTagging
	} {
		for _, bucket := range []*s3.Bucket{stage.collbucket, stage.projbucket} {
			c.Logf("trial %v bucket %v", trial, bucket)
			req, err := http.NewRequest(trial.method, bucket.URL(trial.path), nil)
			c.Check(err, check.IsNil)
			req.Header.Set("Authorization", "AWS "+arvadostest.ActiveTokenV2+":none")
			req.URL.RawQuery = trial.rawquery
			resp, err := http.DefaultClient.Do(req)
			c.Assert(err, check.IsNil)
			c.Check(resp.Header.Get("Content-Type"), check.Equals, "application/xml")
			buf, err := ioutil.ReadAll(resp.Body)
			c.Assert(err, check.IsNil)
			c.Check(string(buf), check.Matches, "(?ms).*InvalidRequest.*API not supported.*")
		}
	}
}

// If there are no CommonPrefixes entries, the CommonPrefixes XML tag
// should not appear at all.
func (s *IntegrationSuite) TestS3ListNoCommonPrefixes(c *check.C) {
	stage := s.s3setup(c)
	defer stage.teardown(c)

	req, err := http.NewRequest("GET", stage.collbucket.URL("/"), nil)
	c.Assert(err, check.IsNil)
	req.Header.Set("Authorization", "AWS "+arvadostest.ActiveTokenV2+":none")
	req.URL.RawQuery = "prefix=asdfasdfasdf&delimiter=/"
	resp, err := http.DefaultClient.Do(req)
	c.Assert(err, check.IsNil)
	buf, err := ioutil.ReadAll(resp.Body)
	c.Assert(err, check.IsNil)
	c.Check(string(buf), check.Not(check.Matches), `(?ms).*CommonPrefixes.*`)
}

// If there is no delimiter in the request, or the results are not
// truncated, the NextMarker XML tag should not appear in the response
// body.
func (s *IntegrationSuite) TestS3ListNoNextMarker(c *check.C) {
	stage := s.s3setup(c)
	defer stage.teardown(c)

	for _, query := range []string{"prefix=e&delimiter=/", ""} {
		req, err := http.NewRequest("GET", stage.collbucket.URL("/"), nil)
		c.Assert(err, check.IsNil)
		req.Header.Set("Authorization", "AWS "+arvadostest.ActiveTokenV2+":none")
		req.URL.RawQuery = query
		resp, err := http.DefaultClient.Do(req)
		c.Assert(err, check.IsNil)
		buf, err := ioutil.ReadAll(resp.Body)
		c.Assert(err, check.IsNil)
		c.Check(string(buf), check.Not(check.Matches), `(?ms).*NextMarker.*`)
	}
}

// List response should include KeyCount field.
func (s *IntegrationSuite) TestS3ListKeyCount(c *check.C) {
	stage := s.s3setup(c)
	defer stage.teardown(c)

	req, err := http.NewRequest("GET", stage.collbucket.URL("/"), nil)
	c.Assert(err, check.IsNil)
	req.Header.Set("Authorization", "AWS "+arvadostest.ActiveTokenV2+":none")
	req.URL.RawQuery = "prefix=&delimiter=/"
	resp, err := http.DefaultClient.Do(req)
	c.Assert(err, check.IsNil)
	buf, err := ioutil.ReadAll(resp.Body)
	c.Assert(err, check.IsNil)
	c.Check(string(buf), check.Matches, `(?ms).*<KeyCount>2</KeyCount>.*`)
}

func (s *IntegrationSuite) TestS3CollectionList(c *check.C) {
	stage := s.s3setup(c)
	defer stage.teardown(c)

	var markers int
	for markers, s.handler.Cluster.Collections.S3FolderObjects = range []bool{false, true} {
		dirs := 2000
		filesPerDir := 2
		stage.writeBigDirs(c, dirs, filesPerDir)
		// Total # objects is:
		//                 2 file entries from s3setup (emptyfile and sailboat.txt)
		//                +1 fake "directory" marker from s3setup (emptydir) (if enabled)
		//             +dirs fake "directory" marker from writeBigDirs (dir0/, dir1/) (if enabled)
		// +filesPerDir*dirs file entries from writeBigDirs (dir0/file0.txt, etc.)
		s.testS3List(c, stage.collbucket, "", 4000, markers+2+(filesPerDir+markers)*dirs)
		s.testS3List(c, stage.collbucket, "", 131, markers+2+(filesPerDir+markers)*dirs)
		s.testS3List(c, stage.collbucket, "", 51, markers+2+(filesPerDir+markers)*dirs)
		s.testS3List(c, stage.collbucket, "dir0/", 71, filesPerDir+markers)
	}
}
func (s *IntegrationSuite) testS3List(c *check.C, bucket *s3.Bucket, prefix string, pageSize, expectFiles int) {
	c.Logf("testS3List: prefix=%q pageSize=%d S3FolderObjects=%v", prefix, pageSize, s.handler.Cluster.Collections.S3FolderObjects)
	expectPageSize := pageSize
	if expectPageSize > 1000 {
		expectPageSize = 1000
	}
	gotKeys := map[string]s3.Key{}
	nextMarker := ""
	pages := 0
	for {
		resp, err := bucket.List(prefix, "", nextMarker, pageSize)
		if !c.Check(err, check.IsNil) {
			break
		}
		c.Check(len(resp.Contents) <= expectPageSize, check.Equals, true)
		if pages++; !c.Check(pages <= (expectFiles/expectPageSize)+1, check.Equals, true) {
			break
		}
		for _, key := range resp.Contents {
			if _, dup := gotKeys[key.Key]; dup {
				c.Errorf("got duplicate key %q on page %d", key.Key, pages)
			}
			gotKeys[key.Key] = key
			if strings.Contains(key.Key, "sailboat.txt") {
				c.Check(key.Size, check.Equals, int64(4))
			}
		}
		if !resp.IsTruncated {
			c.Check(resp.NextMarker, check.Equals, "")
			break
		}
		if !c.Check(resp.NextMarker, check.Not(check.Equals), "") {
			break
		}
		nextMarker = resp.NextMarker
	}
	if !c.Check(len(gotKeys), check.Equals, expectFiles) {
		var sorted []string
		for k := range gotKeys {
			sorted = append(sorted, k)
		}
		sort.Strings(sorted)
		for _, k := range sorted {
			c.Logf("got %s", k)
		}
	}
}

func (s *IntegrationSuite) TestS3CollectionListRollup(c *check.C) {
	for _, s.handler.Cluster.Collections.S3FolderObjects = range []bool{false, true} {
		s.testS3CollectionListRollup(c)
	}
}

func (s *IntegrationSuite) testS3CollectionListRollup(c *check.C) {
	stage := s.s3setup(c)
	defer stage.teardown(c)

	dirs := 2
	filesPerDir := 500
	stage.writeBigDirs(c, dirs, filesPerDir)
	err := stage.collbucket.PutReader("dingbats", &bytes.Buffer{}, 0, "application/octet-stream", s3.Private, s3.Options{})
	c.Assert(err, check.IsNil)
	var allfiles []string
	for marker := ""; ; {
		resp, err := stage.collbucket.List("", "", marker, 20000)
		c.Check(err, check.IsNil)
		for _, key := range resp.Contents {
			if len(allfiles) == 0 || allfiles[len(allfiles)-1] != key.Key {
				allfiles = append(allfiles, key.Key)
			}
		}
		marker = resp.NextMarker
		if marker == "" {
			break
		}
	}
	markers := 0
	if s.handler.Cluster.Collections.S3FolderObjects {
		markers = 1
	}
	c.Check(allfiles, check.HasLen, dirs*(filesPerDir+markers)+3+markers)

	gotDirMarker := map[string]bool{}
	for _, name := range allfiles {
		isDirMarker := strings.HasSuffix(name, "/")
		if markers == 0 {
			c.Check(isDirMarker, check.Equals, false, check.Commentf("name %q", name))
		} else if isDirMarker {
			gotDirMarker[name] = true
		} else if i := strings.LastIndex(name, "/"); i >= 0 {
			c.Check(gotDirMarker[name[:i+1]], check.Equals, true, check.Commentf("name %q", name))
			gotDirMarker[name[:i+1]] = true // skip redundant complaints about this dir marker
		}
	}

	for _, trial := range []struct {
		prefix    string
		delimiter string
		marker    string
	}{
		{"", "", ""},
		{"di", "/", ""},
		{"di", "r", ""},
		{"di", "n", ""},
		{"dir0", "/", ""},
		{"dir0/", "/", ""},
		{"dir0/f", "/", ""},
		{"dir0", "", ""},
		{"dir0/", "", ""},
		{"dir0/f", "", ""},
		{"dir0", "/", "dir0/file14.txt"},       // one commonprefix, "dir0/"
		{"dir0", "/", "dir0/zzzzfile.txt"},     // no commonprefixes
		{"", "", "dir0/file14.txt"},            // middle page, skip walking dir1
		{"", "", "dir1/file14.txt"},            // middle page, skip walking dir0
		{"", "", "dir1/file498.txt"},           // last page of results
		{"dir1/file", "", "dir1/file498.txt"},  // last page of results, with prefix
		{"dir1/file", "/", "dir1/file498.txt"}, // last page of results, with prefix + delimiter
		{"dir1", "Z", "dir1/file498.txt"},      // delimiter "Z" never appears
		{"dir2", "/", ""},                      // prefix "dir2" does not exist
		{"", "/", ""},
	} {
		c.Logf("\n\n=== trial %+v markers=%d", trial, markers)

		maxKeys := 20
		resp, err := stage.collbucket.List(trial.prefix, trial.delimiter, trial.marker, maxKeys)
		c.Check(err, check.IsNil)
		if resp.IsTruncated && trial.delimiter == "" {
			// goamz List method fills in the missing
			// NextMarker field if resp.IsTruncated, so
			// now we can't really tell whether it was
			// sent by the server or by goamz. In cases
			// where it should be empty but isn't, assume
			// it's goamz's fault.
			resp.NextMarker = ""
		}

		var expectKeys []string
		var expectPrefixes []string
		var expectNextMarker string
		var expectTruncated bool
		for _, key := range allfiles {
			full := len(expectKeys)+len(expectPrefixes) >= maxKeys
			if !strings.HasPrefix(key, trial.prefix) || key <= trial.marker {
				continue
			} else if idx := strings.Index(key[len(trial.prefix):], trial.delimiter); trial.delimiter != "" && idx >= 0 {
				prefix := key[:len(trial.prefix)+idx+1]
				if len(expectPrefixes) > 0 && expectPrefixes[len(expectPrefixes)-1] == prefix {
					// same prefix as previous key
				} else if full {
					expectTruncated = true
				} else {
					expectPrefixes = append(expectPrefixes, prefix)
					expectNextMarker = prefix
				}
			} else if full {
				expectTruncated = true
				break
			} else {
				expectKeys = append(expectKeys, key)
				if trial.delimiter != "" {
					expectNextMarker = key
				}
			}
		}
		if !expectTruncated {
			expectNextMarker = ""
		}

		var gotKeys []string
		for _, key := range resp.Contents {
			gotKeys = append(gotKeys, key.Key)
		}
		var gotPrefixes []string
		for _, prefix := range resp.CommonPrefixes {
			gotPrefixes = append(gotPrefixes, prefix)
		}
		commentf := check.Commentf("trial %+v markers=%d", trial, markers)
		c.Check(gotKeys, check.DeepEquals, expectKeys, commentf)
		c.Check(gotPrefixes, check.DeepEquals, expectPrefixes, commentf)
		c.Check(resp.NextMarker, check.Equals, expectNextMarker, commentf)
		c.Check(resp.IsTruncated, check.Equals, expectTruncated, commentf)
		c.Logf("=== trial %+v keys %q prefixes %q nextMarker %q", trial, gotKeys, gotPrefixes, resp.NextMarker)
	}
}

func (s *IntegrationSuite) TestS3ListObjectsV2ManySubprojects(c *check.C) {
	stage := s.s3setup(c)
	defer stage.teardown(c)
	projects := 50
	collectionsPerProject := 2
	for i := 0; i < projects; i++ {
		var subproj arvados.Group
		err := stage.arv.RequestAndDecode(&subproj, "POST", "arvados/v1/groups", nil, map[string]interface{}{
			"group": map[string]interface{}{
				"owner_uuid":  stage.subproj.UUID,
				"group_class": "project",
				"name":        fmt.Sprintf("keep-web s3 test subproject %d", i),
			},
		})
		c.Assert(err, check.IsNil)
		for j := 0; j < collectionsPerProject; j++ {
			err = stage.arv.RequestAndDecode(nil, "POST", "arvados/v1/collections", nil, map[string]interface{}{"collection": map[string]interface{}{
				"owner_uuid":    subproj.UUID,
				"name":          fmt.Sprintf("keep-web s3 test collection %d", j),
				"manifest_text": ". d41d8cd98f00b204e9800998ecf8427e+0 0:0:emptyfile\n./emptydir d41d8cd98f00b204e9800998ecf8427e+0 0:0:.\n",
			}})
			c.Assert(err, check.IsNil)
		}
	}
	c.Logf("setup complete")

	sess := aws_session.Must(aws_session.NewSession(&aws_aws.Config{
		Region:           aws_aws.String("auto"),
		Endpoint:         aws_aws.String(s.testServer.URL),
		Credentials:      aws_credentials.NewStaticCredentials(url.QueryEscape(arvadostest.ActiveTokenV2), url.QueryEscape(arvadostest.ActiveTokenV2), ""),
		S3ForcePathStyle: aws_aws.Bool(true),
	}))
	client := aws_s3.New(sess)
	ctx := context.Background()
	params := aws_s3.ListObjectsV2Input{
		Bucket:    aws_aws.String(stage.proj.UUID),
		Delimiter: aws_aws.String("/"),
		Prefix:    aws_aws.String("keep-web s3 test subproject/"),
		MaxKeys:   aws_aws.Int64(int64(projects / 2)),
	}
	for page := 1; ; page++ {
		t0 := time.Now()
		result, err := client.ListObjectsV2WithContext(ctx, &params)
		if !c.Check(err, check.IsNil) {
			break
		}
		c.Logf("got page %d in %v with len(Contents) == %d, len(CommonPrefixes) == %d", page, time.Since(t0), len(result.Contents), len(result.CommonPrefixes))
		if !*result.IsTruncated {
			break
		}
		params.ContinuationToken = result.NextContinuationToken
		*params.MaxKeys = *params.MaxKeys/2 + 1
	}
}

func (s *IntegrationSuite) TestS3ListObjectsV2(c *check.C) {
	stage := s.s3setup(c)
	defer stage.teardown(c)
	dirs := 2
	filesPerDir := 40
	stage.writeBigDirs(c, dirs, filesPerDir)

	sess := aws_session.Must(aws_session.NewSession(&aws_aws.Config{
		Region:           aws_aws.String("auto"),
		Endpoint:         aws_aws.String(s.testServer.URL),
		Credentials:      aws_credentials.NewStaticCredentials(url.QueryEscape(arvadostest.ActiveTokenV2), url.QueryEscape(arvadostest.ActiveTokenV2), ""),
		S3ForcePathStyle: aws_aws.Bool(true),
	}))

	stringOrNil := func(s string) *string {
		if s == "" {
			return nil
		} else {
			return &s
		}
	}

	client := aws_s3.New(sess)
	ctx := context.Background()

	for _, trial := range []struct {
		prefix               string
		delimiter            string
		startAfter           string
		maxKeys              int
		expectKeys           int
		expectCommonPrefixes map[string]bool
	}{
		{
			// Expect {filesPerDir plus the dir itself}
			// for each dir, plus emptydir, emptyfile, and
			// sailboat.txt.
			expectKeys: (filesPerDir+1)*dirs + 3,
		},
		{
			maxKeys:    15,
			expectKeys: (filesPerDir+1)*dirs + 3,
		},
		{
			startAfter: "dir0/z",
			maxKeys:    15,
			// Expect {filesPerDir plus the dir itself}
			// for each dir except dir0, plus emptydir,
			// emptyfile, and sailboat.txt.
			expectKeys: (filesPerDir+1)*(dirs-1) + 3,
		},
		{
			maxKeys:              1,
			delimiter:            "/",
			expectKeys:           2, // emptyfile, sailboat.txt
			expectCommonPrefixes: map[string]bool{"dir0/": true, "dir1/": true, "emptydir/": true},
		},
		{
			startAfter:           "dir0/z",
			maxKeys:              15,
			delimiter:            "/",
			expectKeys:           2, // emptyfile, sailboat.txt
			expectCommonPrefixes: map[string]bool{"dir1/": true, "emptydir/": true},
		},
		{
			startAfter:           "dir0/file10.txt",
			maxKeys:              15,
			delimiter:            "/",
			expectKeys:           2,
			expectCommonPrefixes: map[string]bool{"dir0/": true, "dir1/": true, "emptydir/": true},
		},
		{
			startAfter:           "dir0/file10.txt",
			maxKeys:              15,
			prefix:               "d",
			delimiter:            "/",
			expectKeys:           0,
			expectCommonPrefixes: map[string]bool{"dir0/": true, "dir1/": true},
		},
	} {
		c.Logf("[trial %+v]", trial)
		params := aws_s3.ListObjectsV2Input{
			Bucket:     aws_aws.String(stage.collbucket.Name),
			Prefix:     stringOrNil(trial.prefix),
			Delimiter:  stringOrNil(trial.delimiter),
			StartAfter: stringOrNil(trial.startAfter),
			MaxKeys:    aws_aws.Int64(int64(trial.maxKeys)),
		}
		keySeen := map[string]bool{}
		prefixSeen := map[string]bool{}
		for {
			result, err := client.ListObjectsV2WithContext(ctx, &params)
			if !c.Check(err, check.IsNil) {
				break
			}
			c.Check(result.Name, check.DeepEquals, aws_aws.String(stage.collbucket.Name))
			c.Check(result.Prefix, check.DeepEquals, aws_aws.String(trial.prefix))
			c.Check(result.Delimiter, check.DeepEquals, aws_aws.String(trial.delimiter))
			// The following two fields are expected to be
			// nil (i.e., no tag in XML response) rather
			// than "" when the corresponding request
			// field was empty or nil.
			c.Check(result.StartAfter, check.DeepEquals, stringOrNil(trial.startAfter))
			c.Check(result.ContinuationToken, check.DeepEquals, params.ContinuationToken)

			if trial.maxKeys > 0 {
				c.Check(result.MaxKeys, check.DeepEquals, aws_aws.Int64(int64(trial.maxKeys)))
				c.Check(len(result.Contents)+len(result.CommonPrefixes) <= trial.maxKeys, check.Equals, true)
			} else {
				c.Check(result.MaxKeys, check.DeepEquals, aws_aws.Int64(int64(s3MaxKeys)))
			}

			for _, ent := range result.Contents {
				c.Assert(ent.Key, check.NotNil)
				c.Check(*ent.Key > trial.startAfter, check.Equals, true)
				c.Check(keySeen[*ent.Key], check.Equals, false, check.Commentf("dup key %q", *ent.Key))
				keySeen[*ent.Key] = true
			}
			for _, ent := range result.CommonPrefixes {
				c.Assert(ent.Prefix, check.NotNil)
				c.Check(strings.HasSuffix(*ent.Prefix, trial.delimiter), check.Equals, true, check.Commentf("bad CommonPrefix %q", *ent.Prefix))
				if strings.HasPrefix(trial.startAfter, *ent.Prefix) {
					// If we asked for
					// startAfter=dir0/file10.txt,
					// we expect dir0/ to be
					// returned as a common prefix
				} else {
					c.Check(*ent.Prefix > trial.startAfter, check.Equals, true)
				}
				c.Check(prefixSeen[*ent.Prefix], check.Equals, false, check.Commentf("dup common prefix %q", *ent.Prefix))
				prefixSeen[*ent.Prefix] = true
			}
			if *result.IsTruncated && c.Check(result.NextContinuationToken, check.Not(check.Equals), "") {
				params.ContinuationToken = aws_aws.String(*result.NextContinuationToken)
			} else {
				break
			}
		}
		c.Check(keySeen, check.HasLen, trial.expectKeys)
		c.Check(prefixSeen, check.HasLen, len(trial.expectCommonPrefixes))
		if len(trial.expectCommonPrefixes) > 0 {
			c.Check(prefixSeen, check.DeepEquals, trial.expectCommonPrefixes)
		}
	}
}

func (s *IntegrationSuite) TestS3ListObjectsV2EncodingTypeURL(c *check.C) {
	stage := s.s3setup(c)
	defer stage.teardown(c)
	dirs := 2
	filesPerDir := 40
	stage.writeBigDirs(c, dirs, filesPerDir)

	sess := aws_session.Must(aws_session.NewSession(&aws_aws.Config{
		Region:           aws_aws.String("auto"),
		Endpoint:         aws_aws.String(s.testServer.URL),
		Credentials:      aws_credentials.NewStaticCredentials(url.QueryEscape(arvadostest.ActiveTokenV2), url.QueryEscape(arvadostest.ActiveTokenV2), ""),
		S3ForcePathStyle: aws_aws.Bool(true),
	}))

	client := aws_s3.New(sess)
	ctx := context.Background()

	result, err := client.ListObjectsV2WithContext(ctx, &aws_s3.ListObjectsV2Input{
		Bucket:       aws_aws.String(stage.collbucket.Name),
		Prefix:       aws_aws.String("dir0/"),
		Delimiter:    aws_aws.String("/"),
		StartAfter:   aws_aws.String("dir0/"),
		EncodingType: aws_aws.String("url"),
	})
	c.Assert(err, check.IsNil)
	c.Check(*result.Prefix, check.Equals, "dir0%2F")
	c.Check(*result.Delimiter, check.Equals, "%2F")
	c.Check(*result.StartAfter, check.Equals, "dir0%2F")
	for _, ent := range result.Contents {
		c.Check(*ent.Key, check.Matches, "dir0%2F.*")
	}
	result, err = client.ListObjectsV2WithContext(ctx, &aws_s3.ListObjectsV2Input{
		Bucket:       aws_aws.String(stage.collbucket.Name),
		Delimiter:    aws_aws.String("/"),
		EncodingType: aws_aws.String("url"),
	})
	c.Assert(err, check.IsNil)
	c.Check(*result.Delimiter, check.Equals, "%2F")
	c.Check(result.CommonPrefixes, check.HasLen, dirs+1)
	for _, ent := range result.CommonPrefixes {
		c.Check(*ent.Prefix, check.Matches, ".*%2F")
	}
}

// TestS3cmd checks compatibility with the s3cmd command line tool, if
// it's installed. As of Debian buster, s3cmd is only in backports, so
// `arvados-server install` don't install it, and this test skips if
// it's not installed.
func (s *IntegrationSuite) TestS3cmd(c *check.C) {
	if _, err := exec.LookPath("s3cmd"); err != nil {
		c.Skip("s3cmd not found")
		return
	}

	stage := s.s3setup(c)
	defer stage.teardown(c)

	cmd := exec.Command("s3cmd", "--no-ssl", "--host="+s.testServer.URL[7:], "--host-bucket="+s.testServer.URL[7:], "--access_key="+arvadostest.ActiveTokenUUID, "--secret_key="+arvadostest.ActiveToken, "ls", "s3://"+arvadostest.FooCollection)
	buf, err := cmd.CombinedOutput()
	c.Check(err, check.IsNil)
	c.Check(string(buf), check.Matches, `.* 3 +s3://`+arvadostest.FooCollection+`/foo\n`)

	// This tests whether s3cmd's path normalization agrees with
	// keep-web's signature verification wrt chars like "|"
	// (neither reserved nor unreserved) and "," (not normally
	// percent-encoded in a path).
	tmpfile := c.MkDir() + "/dstfile"
	cmd = exec.Command("s3cmd", "--no-ssl", "--host="+s.testServer.URL[7:], "--host-bucket="+s.testServer.URL[7:], "--access_key="+arvadostest.ActiveTokenUUID, "--secret_key="+arvadostest.ActiveToken, "get", "s3://"+arvadostest.FooCollection+"/foo,;$[|]bar", tmpfile)
	buf, err = cmd.CombinedOutput()
	c.Check(err, check.NotNil)
	// As of commit b7520e5c25e1bf25c1a8bf5aa2eadb299be8f606
	// (between debian bullseye and bookworm versions), s3cmd
	// started catching the NoSuchKey error code and replacing it
	// with "Source object '%s' does not exist.".
	c.Check(string(buf), check.Matches, `(?ms).*(NoSuchKey|Source object.*does not exist).*\n`)
}

func (s *IntegrationSuite) TestS3BucketInHost(c *check.C) {
	stage := s.s3setup(c)
	defer stage.teardown(c)

	hdr, body, _ := s.runCurl(c, "AWS "+arvadostest.ActiveTokenV2+":none", stage.coll.UUID+".collections.example.com", "/sailboat.txt")
	c.Check(hdr, check.Matches, `(?s)HTTP/1.1 200 OK\r\n.*`)
	c.Check(body, check.Equals, "⛵\n")
}
