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
	"strings"

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
		"description": "Description of test collection\n",
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
			"uuid":        "{{stage.coll.UUID}}",
			"description": "Description of test collection\n",
			"created_at":  "{{stage.coll.CreatedAt}}",
			"modified_at": "{{stage.coll.ModifiedAt}}",
			"modified_by_user": map[string]interface{}{
				"email":     "active-user@arvados.local",
				"full_name": "Active User",
				"username":  "active",
				"uuid":      arvadostest.ActiveUserUUID,
			},
		},
		expectZipComment: `Downloaded from https://collections.example.com/by_id/{{stage.coll.UUID}}/`,
	})
}

func (s *IntegrationSuite) TestZip_Logging(c *C) {
	s.testZip(c, testZipOptions{
		reqMethod:    "POST",
		reqToken:     arvadostest.ActiveTokenV2,
		expectStatus: 200,
		expectFiles:  []string{"dir1/dir/file1.txt", "dir1/file1.txt", "dir2/file2.txt", "file0.txt"},
		expectLogsMatch: []string{
			`(?ms).*\smsg="File download".*`,
			`(?ms).*\suser_uuid=` + arvadostest.ActiveUserUUID + `\s.*`,
			`(?ms).*\suser_full_name="Active User".*`,
			`(?ms).*\sportable_data_hash=6acf043b102afcf04e3be2443e7ea2ba\+223\s.*`,
			`(?ms).*\scollection_file_path=\s.*`,
		},
	})
}

func (s *IntegrationSuite) TestZip_Logging_OneFile(c *C) {
	s.testZip(c, testZipOptions{
		reqMethod:      "POST",
		reqContentType: "application/json",
		reqToken:       arvadostest.ActiveTokenV2,
		reqBody:        `{"files":["dir1/file1.txt"]}`,
		expectStatus:   200,
		expectFiles:    []string{"dir1/file1.txt"},
		expectLogsMatch: []string{
			`(?ms).*\scollection_file_path=dir1/file1.txt\s.*`,
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
		reqBody:        `{"files":[]}`,
		expectStatus:   200,
		expectFiles:    []string{"dir1/dir/file1.txt", "dir1/file1.txt", "dir2/file2.txt", "file0.txt"},
	})
}

func (s *IntegrationSuite) TestZip_EntireCollection_Slash(c *C) {
	s.testZip(c, testZipOptions{
		reqMethod:      "GET",
		reqContentType: "application/json",
		reqToken:       arvadostest.ActiveTokenV2,
		reqBody:        `{"files":["/"]}`,
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

func (s *IntegrationSuite) TestZip_SelectDirectory_SpecifyDownloadFilename_Form(c *C) {
	s.testZip(c, testZipOptions{
		reqMethod:         "POST",
		reqContentType:    "application/x-www-form-urlencoded",
		reqToken:          arvadostest.ActiveTokenV2,
		reqBody:           (url.Values{"files": {"dir1"}, "download_filename": {"Foo Bar.zip"}}).Encode(),
		expectStatus:      200,
		expectFiles:       []string{"dir1/dir/file1.txt", "dir1/file1.txt"},
		expectDisposition: `attachment; filename="Foo Bar.zip"`,
	})
}

func (s *IntegrationSuite) TestZip_SelectDirectory_JSON(c *C) {
	s.testZip(c, testZipOptions{
		reqMethod:         "POST",
		reqContentType:    "application/json",
		reqToken:          arvadostest.ActiveTokenV2,
		reqBody:           `{"files":["dir1"]}`,
		expectStatus:      200,
		expectFiles:       []string{"dir1/dir/file1.txt", "dir1/file1.txt"},
		expectDisposition: `attachment; filename="keep-web zip test collection - 2 files.zip"`,
	})
}

func (s *IntegrationSuite) TestZip_SelectDirectory_TrailingSlash(c *C) {
	s.testZip(c, testZipOptions{
		reqMethod:         "POST",
		reqContentType:    "application/json",
		reqToken:          arvadostest.ActiveTokenV2,
		reqBody:           `{"files":["dir1/"]}`,
		expectStatus:      200,
		expectFiles:       []string{"dir1/dir/file1.txt", "dir1/file1.txt"},
		expectDisposition: `attachment; filename="keep-web zip test collection - 2 files.zip"`,
	})
}

func (s *IntegrationSuite) TestZip_SelectDirectory_SpecifyDownloadFilename(c *C) {
	s.testZip(c, testZipOptions{
		reqMethod:         "POST",
		reqContentType:    "application/json",
		reqToken:          arvadostest.ActiveTokenV2,
		reqBody:           `{"files":["dir1/"],"download_filename":"Foo bar ⛵.zip"}`,
		expectStatus:      200,
		expectFiles:       []string{"dir1/dir/file1.txt", "dir1/file1.txt"},
		expectDisposition: `attachment; filename*=utf-8''Foo%20bar%20%E2%9B%B5.zip`,
	})
}

func (s *IntegrationSuite) TestZip_SelectFile(c *C) {
	s.testZip(c, testZipOptions{
		reqMethod:         "POST",
		reqContentType:    "application/json",
		reqToken:          arvadostest.ActiveTokenV2,
		reqBody:           `{"files":["dir1/file1.txt"]}`,
		expectStatus:      200,
		expectFiles:       []string{"dir1/file1.txt"},
		expectDisposition: `attachment; filename="keep-web zip test collection - file1.txt.zip"`,
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
		expectDisposition: `attachment; filename="keep-web zip test collection - 2 files.zip"`,
	})
}

func (s *IntegrationSuite) TestZip_SelectFiles_SpecifyDownloadFilename_Query(c *C) {
	s.testZip(c, testZipOptions{
		reqMethod: "POST",
		reqQuery: "?" + (&url.Values{
			"files":             []string{"dir1/file1.txt", "dir2/file2.txt"},
			"download_filename": []string{"Sue.zip"},
		}).Encode(),
		reqContentType:    "application/json",
		reqToken:          arvadostest.ActiveTokenV2,
		expectStatus:      200,
		expectFiles:       []string{"dir1/file1.txt", "dir2/file2.txt"},
		expectDisposition: `attachment; filename=Sue.zip`,
	})
}

func (s *IntegrationSuite) TestZip_SpecifyDownloadFilename_NoZipExt(c *C) {
	s.testZip(c, testZipOptions{
		reqMethod: "GET",
		reqQuery: "?" + (&url.Values{
			"download_filename": []string{"Sue.zap"},
		}).Encode(),
		reqContentType:    "application/json",
		reqToken:          arvadostest.ActiveTokenV2,
		expectStatus:      200,
		expectFiles:       []string{"dir1/dir/file1.txt", "dir1/file1.txt", "dir2/file2.txt", "file0.txt"},
		expectDisposition: `attachment; filename=Sue.zap.zip`,
	})
}

func (s *IntegrationSuite) TestZip_SelectFile_UseByIDStyle(c *C) {
	s.testZip(c, testZipOptions{
		useByIDStyle:      true,
		reqMethod:         "POST",
		reqContentType:    "application/json",
		reqToken:          arvadostest.ActiveTokenV2,
		reqBody:           `{"files":["dir1/file1.txt"]}`,
		expectStatus:      200,
		expectFiles:       []string{"dir1/file1.txt"},
		expectDisposition: `attachment; filename="keep-web zip test collection - file1.txt.zip"`,
	})
}

func (s *IntegrationSuite) TestZip_SelectFile_UsePathStyle(c *C) {
	s.testZip(c, testZipOptions{
		usePathStyle:      true,
		reqMethod:         "POST",
		reqContentType:    "application/json",
		reqToken:          arvadostest.ActiveTokenV2,
		reqBody:           `{"files":["dir1/file1.txt"]}`,
		expectStatus:      200,
		expectFiles:       []string{"dir1/file1.txt"},
		expectDisposition: `attachment; filename="keep-web zip test collection - file1.txt.zip"`,
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
		reqBody:           `{"files":["dir1/file1.txt"]}`,
		expectStatus:      200,
		expectFiles:       []string{"collection.json", "dir1/file1.txt"},
		expectDisposition: `attachment; filename="6acf043b102afcf04e3be2443e7ea2ba+223 - file1.txt.zip"`,
		expectMetadata: map[string]interface{}{
			"portable_data_hash": "6acf043b102afcf04e3be2443e7ea2ba+223",
		},
		expectZipComment: `Downloaded from http://collections.example.com/by_id/6acf043b102afcf04e3be2443e7ea2ba+223/`,
	})
}

func (s *IntegrationSuite) TestZip_SelectRedundantFile(c *C) {
	s.testZip(c, testZipOptions{
		reqMethod:      "POST",
		reqContentType: "application/json",
		reqToken:       arvadostest.ActiveTokenV2,
		reqBody:        `{"files":["dir1/dir", "dir1/dir/file1.txt"]}`,
		expectStatus:   200,
		expectFiles:    []string{"dir1/dir/file1.txt"},
	})
}

func (s *IntegrationSuite) TestZip_AcceptMediaTypeWithDirective(c *C) {
	s.testZip(c, testZipOptions{
		reqMethod:      "POST",
		reqContentType: "application/json",
		reqToken:       arvadostest.ActiveTokenV2,
		reqBody:        `{"files":["dir1/dir/file1.txt"]}`,
		reqAccept:      `application/zip; q=0.9`,
		expectStatus:   200,
		expectFiles:    []string{"dir1/dir/file1.txt"},
	})
}

func (s *IntegrationSuite) TestZip_AcceptMediaTypeInQuery(c *C) {
	s.testZip(c, testZipOptions{
		reqMethod:      "POST",
		reqContentType: "application/json",
		reqToken:       arvadostest.ActiveTokenV2,
		reqBody:        `{"files":["dir1/dir/file1.txt"]}`,
		reqQuery:       `?accept=application/zip&disposition=attachment`,
		reqAccept:      `text/html`,
		expectStatus:   200,
		expectFiles:    []string{"dir1/dir/file1.txt"},
	})
}

// disposition=attachment is implied, because usePathStyle causes
// testZip to use DownloadURL as the request vhost.
func (s *IntegrationSuite) TestZip_AcceptMediaTypeInQuery_ImplicitDisposition(c *C) {
	s.testZip(c, testZipOptions{
		usePathStyle:   true,
		reqMethod:      "POST",
		reqContentType: "application/json",
		reqToken:       arvadostest.ActiveTokenV2,
		reqBody:        `{"files":["dir1/dir/file1.txt"]}`,
		reqQuery:       `?accept=application/zip`,
		reqAccept:      `text/html`,
		expectStatus:   200,
		expectFiles:    []string{"dir1/dir/file1.txt"},
	})
}

func (s *IntegrationSuite) TestZip_SelectNonexistentFile(c *C) {
	s.testZip(c, testZipOptions{
		reqMethod:       "POST",
		reqContentType:  "application/json",
		reqToken:        arvadostest.ActiveTokenV2,
		reqBody:         `{"files":["dir1", "file404.txt"]}`,
		expectStatus:    404,
		expectBodyMatch: `"file404.txt": file does not exist\n`,
	})
}

func (s *IntegrationSuite) TestZip_SelectBlankFilename(c *C) {
	s.testZip(c, testZipOptions{
		reqMethod:       "POST",
		reqContentType:  "application/json",
		reqToken:        arvadostest.ActiveTokenV2,
		reqBody:         `{"files":[""]}`,
		expectStatus:    404,
		expectBodyMatch: `"": file does not exist\n`,
	})
}

func (s *IntegrationSuite) TestZip_JSON_Error(c *C) {
	s.testZip(c, testZipOptions{
		reqMethod:       "POST",
		reqContentType:  "application/json",
		reqToken:        arvadostest.ActiveTokenV2,
		reqBody:         `{"files":["dir1/dir"`,
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
	useByIDStyle      bool
	reqMethod         string
	reqQuery          string
	reqAccept         string
	reqContentType    string
	reqToken          string
	reqBody           string
	expectStatus      int
	expectFiles       []string
	expectBodyMatch   string
	expectDisposition string
	expectMetadata    map[string]interface{}
	expectZipComment  string
	expectLogsMatch   []string
}

func (s *IntegrationSuite) testZip(c *C, opts testZipOptions) {
	logbuf := new(bytes.Buffer)
	logger := logrus.New()
	logger.Out = io.MultiWriter(logbuf, ctxlog.LogWriter(c.Log))
	s.ctx = ctxlog.Context(context.Background(), logger)
	s.handler.Cluster.Services.WebDAVDownload.ExternalURL.Host = "collections.example.com"

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
		s.handler.Cluster.Services.WebDAVDownload.ExternalURL.Scheme = "http"
		url = "http://collections.example.com/c=" + collID
	} else if opts.useByIDStyle {
		s.handler.Cluster.Services.WebDAVDownload.ExternalURL.Scheme = "http"
		url = "http://collections.example.com/by_id/" + collID
	} else {
		url = s.collectionURL(collID, "")
	}
	var accept []string
	if opts.reqAccept != "" {
		accept = []string{opts.reqAccept}
	} else {
		accept = []string{"application/zip"}
	}
	_, resp := s.do(opts.reqMethod, url+opts.reqQuery, opts.reqToken, http.Header{
		"Accept":       accept,
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
	if opts.expectZipComment != "" {
		c.Check(zipr.Comment, Equals, strings.Replace(opts.expectZipComment, "{{stage.coll.UUID}}", stage.coll.UUID, -1))
	}
	f, err := zipr.Open("collection.json")
	c.Check(err == nil, Equals, opts.expectMetadata != nil,
		Commentf("collection.json file existence (%v) did not match expectation (%v)", err == nil, opts.expectMetadata != nil))
	if err == nil {
		defer f.Close()
		if opts.expectMetadata["uuid"] == "{{stage.coll.UUID}}" {
			opts.expectMetadata["uuid"] = stage.coll.UUID
		}
		if opts.expectMetadata["created_at"] == "{{stage.coll.CreatedAt}}" {
			opts.expectMetadata["created_at"] = stage.coll.CreatedAt.Format(rfc3339NanoFixed)
		}
		if opts.expectMetadata["modified_at"] == "{{stage.coll.ModifiedAt}}" {
			opts.expectMetadata["modified_at"] = stage.coll.ModifiedAt.Format(rfc3339NanoFixed)
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
