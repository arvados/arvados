// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepweb

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadosclient"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"git.arvados.org/arvados.git/sdk/go/keepclient"
	"github.com/sirupsen/logrus"
	. "gopkg.in/check.v1"
)

type zipstage struct {
	arv  *arvados.Client
	ac   *arvadosclient.ArvadosClient
	kc   *keepclient.KeepClient
	coll arvados.Collection
}

func (s *IntegrationSuite) zipsetup(c *C, filedata map[string]string) zipstage {
	arv := arvados.NewClientFromEnv()
	arv.AuthToken = arvadostest.ActiveToken
	var coll arvados.Collection
	err := arv.RequestAndDecode(&coll, "POST", "arvados/v1/collections", nil, map[string]interface{}{"collection": map[string]interface{}{
		"name": "keep-web zip test collection",
		"properties": map[string]interface{}{
			"sailboat": "⛵",
		},
	}})
	c.Assert(err, IsNil)
	ac, err := arvadosclient.New(arv)
	c.Assert(err, IsNil)
	kc, err := keepclient.MakeKeepClient(ac)
	c.Assert(err, IsNil)
	fs, err := coll.FileSystem(arv, kc)
	c.Assert(err, IsNil)
	for path, data := range filedata {
		for i, c := range path {
			if c == '/' {
				fs.Mkdir(path[:i], 0777)
			}
		}
		f, err := fs.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0644)
		c.Assert(err, IsNil)
		_, err = f.Write([]byte(data))
		c.Assert(err, IsNil)
		err = f.Close()
		c.Assert(err, IsNil)
	}
	err = fs.Sync()
	c.Assert(err, IsNil)
	err = arv.RequestAndDecode(&coll, "GET", "arvados/v1/collections/"+coll.UUID, nil, nil)
	c.Assert(err, IsNil)

	return zipstage{
		arv:  arv,
		ac:   ac,
		kc:   kc,
		coll: coll,
	}
}

func (stage zipstage) teardown(c *C) {
	if stage.coll.UUID != "" {
		err := stage.arv.RequestAndDecode(&stage.coll, "DELETE", "arvados/v1/collections/"+stage.coll.UUID, nil, nil)
		c.Check(err, IsNil)
	}
}

func (s *IntegrationSuite) TestZip_EmptyCollection(c *C) {
	stage := s.zipsetup(c, nil)
	defer stage.teardown(c)
	_, resp := s.do("POST", s.collectionURL(stage.coll.UUID, ""), arvadostest.ActiveTokenV2, http.Header{"Accept": {"application/zip"}}, nil)
	if !c.Check(resp.StatusCode, Equals, http.StatusOK) {
		body, _ := io.ReadAll(resp.Body)
		c.Logf("response body: %q", body)
		return
	}
	zipdata, _ := io.ReadAll(resp.Body)
	zipr, err := zip.NewReader(bytes.NewReader(zipdata), int64(len(zipdata)))
	c.Assert(err, IsNil)
	c.Check(zipr.File, HasLen, 0)
}

func (s *IntegrationSuite) TestZip_Metadata(c *C) {
	s.testZip(c, testZipOptions{
		reqMethod:    "GET",
		reqQuery:     "?include_collection_metadata=1",
		reqToken:     arvadostest.ActiveTokenV2,
		expectStatus: 200,
		expectFiles:  []string{"collection.json", "dir1/dir/file1.txt", "dir1/file1.txt", "dir2/file2.txt", "file0.txt"},
		expectMetadata: map[string]interface{}{
			"name":               "keep-web zip test collection",
			"portable_data_hash": "6acf043b102afcf04e3be2443e7ea2ba+223",
			"properties": map[string]interface{}{
				"sailboat": "⛵",
			},
			"uuid": "{{stage.coll.UUID}}",
		},
	})
}

func (s *IntegrationSuite) TestZip_Logging(c *C) {
	s.testZip(c, testZipOptions{
		reqMethod:    "POST",
		reqToken:     arvadostest.ActiveTokenV2,
		expectStatus: 200,
		expectFiles:  []string{"dir1/dir/file1.txt", "dir1/file1.txt", "dir2/file2.txt", "file0.txt"},
		expectLogsMatch: []string{
			`(?ms).*msg="File download".*`,
			`(?ms).*user_uuid=` + arvadostest.ActiveUserUUID + `\s.*`,
			`(?ms).*user_full_name="Active User".*`,
			`(?ms).*portable_data_hash=6acf043b102afcf04e3be2443e7ea2ba\+223.*`,
			`(?ms).*collection_file_path=\s.*`,
		},
	})
}

func (s *IntegrationSuite) TestZip_Logging_OneFile(c *C) {
	s.testZip(c, testZipOptions{
		reqMethod:      "POST",
		reqContentType: "application/json",
		reqToken:       arvadostest.ActiveTokenV2,
		reqBody:        `["dir1/file1.txt"]`,
		expectStatus:   200,
		expectFiles:    []string{"dir1/file1.txt"},
		expectLogsMatch: []string{
			`(?ms).*collection_file_path=dir1/file1.txt.*`,
		},
	})
}

func (s *IntegrationSuite) TestZip_EntireCollection_GET(c *C) {
	s.testZip(c, testZipOptions{
		reqMethod:    "GET",
		reqToken:     arvadostest.ActiveTokenV2,
		expectStatus: 200,
		expectFiles:  []string{"dir1/dir/file1.txt", "dir1/file1.txt", "dir2/file2.txt", "file0.txt"},
	})
}

func (s *IntegrationSuite) TestZip_EntireCollection_JSON(c *C) {
	s.testZip(c, testZipOptions{
		reqMethod:      "GET",
		reqContentType: "application/json",
		reqToken:       arvadostest.ActiveTokenV2,
		reqBody:        `[]`,
		expectStatus:   200,
		expectFiles:    []string{"dir1/dir/file1.txt", "dir1/file1.txt", "dir2/file2.txt", "file0.txt"},
	})
}

func (s *IntegrationSuite) TestZip_EntireCollection_Slash(c *C) {
	s.testZip(c, testZipOptions{
		reqMethod:      "GET",
		reqContentType: "application/json",
		reqToken:       arvadostest.ActiveTokenV2,
		reqBody:        `["/"]`,
		expectStatus:   200,
		expectFiles:    []string{"dir1/dir/file1.txt", "dir1/file1.txt", "dir2/file2.txt", "file0.txt"},
	})
}

func (s *IntegrationSuite) TestZip_SelectDirectory_Form(c *C) {
	s.testZip(c, testZipOptions{
		reqMethod:      "POST",
		reqContentType: "application/x-www-form-urlencoded",
		reqToken:       arvadostest.ActiveTokenV2,
		reqBody:        (url.Values{"files": {"dir1"}}).Encode(),
		expectStatus:   200,
		expectFiles:    []string{"dir1/dir/file1.txt", "dir1/file1.txt"},
	})
}

func (s *IntegrationSuite) TestZip_SelectDirectory_JSON(c *C) {
	s.testZip(c, testZipOptions{
		reqMethod:         "POST",
		reqContentType:    "application/json",
		reqToken:          arvadostest.ActiveTokenV2,
		reqBody:           `["dir1"]`,
		expectStatus:      200,
		expectFiles:       []string{"dir1/dir/file1.txt", "dir1/file1.txt"},
		expectDisposition: `attachment; filename="keep-web zip test collection - 2 files"`,
	})
}

func (s *IntegrationSuite) TestZip_SelectDirectory_TrailingSlash(c *C) {
	s.testZip(c, testZipOptions{
		reqMethod:         "POST",
		reqContentType:    "application/json",
		reqToken:          arvadostest.ActiveTokenV2,
		reqBody:           `["dir1/"]`,
		expectStatus:      200,
		expectFiles:       []string{"dir1/dir/file1.txt", "dir1/file1.txt"},
		expectDisposition: `attachment; filename="keep-web zip test collection - 2 files"`,
	})
}

func (s *IntegrationSuite) TestZip_SelectFile(c *C) {
	s.testZip(c, testZipOptions{
		reqMethod:         "POST",
		reqContentType:    "application/json",
		reqToken:          arvadostest.ActiveTokenV2,
		reqBody:           `["dir1/file1.txt"]`,
		expectStatus:      200,
		expectFiles:       []string{"dir1/file1.txt"},
		expectDisposition: `attachment; filename="keep-web zip test collection - file1.txt"`,
	})
}

func (s *IntegrationSuite) TestZip_SelectFiles_Query(c *C) {
	s.testZip(c, testZipOptions{
		reqMethod:         "POST",
		reqQuery:          "?" + (&url.Values{"files": []string{"dir1/file1.txt", "dir2/file2.txt"}}).Encode(),
		reqContentType:    "application/json",
		reqToken:          arvadostest.ActiveTokenV2,
		expectStatus:      200,
		expectFiles:       []string{"dir1/file1.txt", "dir2/file2.txt"},
		expectDisposition: `attachment; filename="keep-web zip test collection - 2 files"`,
	})
}

func (s *IntegrationSuite) TestZip_SelectFile_UsePathStyle(c *C) {
	s.testZip(c, testZipOptions{
		usePathStyle:      true,
		reqMethod:         "POST",
		reqContentType:    "application/json",
		reqToken:          arvadostest.ActiveTokenV2,
		reqBody:           `["dir1/file1.txt"]`,
		expectStatus:      200,
		expectFiles:       []string{"dir1/file1.txt"},
		expectDisposition: `attachment; filename="keep-web zip test collection - file1.txt"`,
	})
}

func (s *IntegrationSuite) TestZip_SelectFile_UsePathStyle_PDH(c *C) {
	s.testZip(c, testZipOptions{
		usePathStyle:      true,
		usePDH:            true,
		reqMethod:         "POST",
		reqQuery:          "?include_collection_metadata=1",
		reqContentType:    "application/json",
		reqToken:          arvadostest.ActiveTokenV2,
		reqBody:           `["dir1/file1.txt"]`,
		expectStatus:      200,
		expectFiles:       []string{"collection.json", "dir1/file1.txt"},
		expectDisposition: `attachment; filename="6acf043b102afcf04e3be2443e7ea2ba+223 - file1.txt"`,
		expectMetadata: map[string]interface{}{
			"portable_data_hash": "6acf043b102afcf04e3be2443e7ea2ba+223",
		},
	})
}

func (s *IntegrationSuite) TestZip_SelectRedundantFile(c *C) {
	s.testZip(c, testZipOptions{
		reqMethod:      "POST",
		reqContentType: "application/json",
		reqToken:       arvadostest.ActiveTokenV2,
		reqBody:        `["dir1/dir", "dir1/dir/file1.txt"]`,
		expectStatus:   200,
		expectFiles:    []string{"dir1/dir/file1.txt"},
	})
}

func (s *IntegrationSuite) TestZip_SelectNonexistentFile(c *C) {
	s.testZip(c, testZipOptions{
		reqMethod:       "POST",
		reqContentType:  "application/json",
		reqToken:        arvadostest.ActiveTokenV2,
		reqBody:         `["dir1", "file404.txt"]`,
		expectStatus:    404,
		expectBodyMatch: `"file404.txt": file does not exist\n`,
	})
}

func (s *IntegrationSuite) TestZip_SelectBlankFilename(c *C) {
	s.testZip(c, testZipOptions{
		reqMethod:       "POST",
		reqContentType:  "application/json",
		reqToken:        arvadostest.ActiveTokenV2,
		reqBody:         `[""]`,
		expectStatus:    404,
		expectBodyMatch: `"": file does not exist\n`,
	})
}

func (s *IntegrationSuite) TestZip_JSON_Error(c *C) {
	s.testZip(c, testZipOptions{
		reqMethod:       "POST",
		reqContentType:  "application/json",
		reqToken:        arvadostest.ActiveTokenV2,
		reqBody:         `["dir1/dir"`,
		expectStatus:    http.StatusBadRequest,
		expectBodyMatch: `.*unexpected EOF.*\n`,
	})
}

// Download-via-POST is still allowed if upload permission is turned
// off.
func (s *IntegrationSuite) TestZip_WebDAVPermission_OK(c *C) {
	s.handler.Cluster.Collections.WebDAVPermission.User.Upload = false
	s.testZip(c, testZipOptions{
		reqMethod:      "POST",
		reqContentType: "application/json",
		reqToken:       arvadostest.ActiveTokenV2,
		expectFiles:    []string{"dir1/dir/file1.txt", "dir1/file1.txt", "dir2/file2.txt", "file0.txt"},
		expectStatus:   http.StatusOK,
	})
}

func (s *IntegrationSuite) TestZip_WebDAVPermission_Forbidden(c *C) {
	s.handler.Cluster.Collections.WebDAVPermission.User.Download = false
	s.testZip(c, testZipOptions{
		reqMethod:       "POST",
		reqContentType:  "application/json",
		reqToken:        arvadostest.ActiveTokenV2,
		expectStatus:    http.StatusForbidden,
		expectBodyMatch: `Not permitted\n`,
	})
}

type testZipOptions struct {
	filedata          map[string]string // if nil, use default set (see testZip)
	usePDH            bool
	usePathStyle      bool
	reqMethod         string
	reqQuery          string
	reqContentType    string
	reqToken          string
	reqBody           string
	expectStatus      int
	expectFiles       []string
	expectBodyMatch   string
	expectDisposition string
	expectMetadata    map[string]interface{}
	expectLogsMatch   []string
}

func (s *IntegrationSuite) testZip(c *C, opts testZipOptions) {
	logbuf := new(bytes.Buffer)
	logger := logrus.New()
	logger.Out = io.MultiWriter(logbuf, ctxlog.LogWriter(c.Log))
	s.ctx = ctxlog.Context(context.Background(), logger)

	if opts.filedata == nil {
		opts.filedata = map[string]string{
			"dir1/dir/file1.txt": "file1",
			"dir1/file1.txt":     "file1",
			"dir2/file2.txt":     "file2",
			"file0.txt":          "file0",
		}
	}
	stage := s.zipsetup(c, opts.filedata)
	defer stage.teardown(c)
	var collID string
	if opts.usePDH {
		collID = stage.coll.PortableDataHash
	} else {
		collID = stage.coll.UUID
	}
	var url string
	if opts.usePathStyle {
		s.handler.Cluster.Services.WebDAVDownload.ExternalURL.Host = "collections.example.com"
		s.handler.Cluster.Services.WebDAVDownload.ExternalURL.Scheme = "http"
		url = "http://collections.example.com/c=" + collID
	} else {
		url = s.collectionURL(collID, "")
	}
	_, resp := s.do(opts.reqMethod, url+opts.reqQuery, opts.reqToken, http.Header{
		"Accept":       {"application/zip"},
		"Content-Type": {opts.reqContentType},
	}, []byte(opts.reqBody))
	if !c.Check(resp.StatusCode, Equals, opts.expectStatus) || opts.expectStatus != 200 {
		body, _ := io.ReadAll(resp.Body)
		c.Logf("response body: %q", body)
		if opts.expectBodyMatch != "" {
			c.Check(string(body), Matches, opts.expectBodyMatch)
		}
		return
	}
	zipdata, _ := io.ReadAll(resp.Body)
	zipr, err := zip.NewReader(bytes.NewReader(zipdata), int64(len(zipdata)))
	c.Assert(err, IsNil)
	c.Check(zipFileNames(zipr), DeepEquals, opts.expectFiles)
	if opts.expectDisposition != "" {
		c.Check(resp.Header.Get("Content-Disposition"), Equals, opts.expectDisposition)
	}
	f, err := zipr.Open("collection.json")
	if opts.expectMetadata != nil && c.Check(err, IsNil) {
		if opts.expectMetadata["uuid"] == "{{stage.coll.UUID}}" {
			opts.expectMetadata["uuid"] = stage.coll.UUID
		}
		var gotMetadata map[string]interface{}
		json.NewDecoder(f).Decode(&gotMetadata)
		c.Check(gotMetadata, DeepEquals, opts.expectMetadata)
	}
	for _, re := range opts.expectLogsMatch {
		c.Check(logbuf.String(), Matches, re)
	}
}

func zipFileNames(zipr *zip.Reader) []string {
	var names []string
	for _, file := range zipr.File {
		names = append(names, file.Name)
	}
	return names
}
