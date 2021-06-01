// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package diagnostics

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"github.com/sirupsen/logrus"
)

type Command struct {
	projectName string
}

func (diag Command) RunCommand(prog string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	f := flag.NewFlagSet(prog, flag.ContinueOnError)
	f.StringVar(&diag.projectName, "project-name", "scratch area for diagnostics", "name of project to find/create in home project and use for temporary/test objects")
	loglevel := f.String("log-level", "info", "logging level (debug, info, warning, error)")
	checkInternal := f.Bool("internal-client", false, "check that this host is considered an \"internal\" client")
	checkExternal := f.Bool("external-client", false, "check that this host is considered an \"external\" client")
	timeout := f.Duration("timeout", 10*time.Second, "timeout for http requests")
	err := f.Parse(args)
	if err == flag.ErrHelp {
		return 0
	} else if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}

	ctx := context.Background()

	logger := ctxlog.New(stdout, "text", *loglevel)
	logger.SetFormatter(&logrus.TextFormatter{DisableTimestamp: true, DisableLevelTruncation: true})

	infof := logger.Infof
	warnf := logger.Warnf
	debugf := logger.Debugf
	var errors []string
	errorf := func(f string, args ...interface{}) {
		logger.Errorf(f, args...)
		errors = append(errors, fmt.Sprintf(f, args...))
	}
	defer func() {
		if len(errors) == 0 {
			logger.Info("--- no errors ---")
		} else {
			fmt.Fprint(stdout, "\n--- cut here --- error summary ---\n\n")
			for _, e := range errors {
				logger.Error(e)
			}
		}
	}()

	client := arvados.NewClientFromEnv()

	var dd arvados.DiscoveryDocument
	ddpath := "discovery/v1/apis/arvados/v1/rest"
	testname := fmt.Sprintf("getting discovery document from https://%s/%s", client.APIHost, ddpath)
	logger.Info(testname)
	err = client.RequestAndDecode(&dd, "GET", ddpath, nil, nil)
	if err != nil {
		errorf("%s: %s", testname, err)
	} else {
		infof("%s: ok, BlobSignatureTTL is %d", testname, dd.BlobSignatureTTL)
	}

	var cluster arvados.Cluster
	cfgpath := "arvados/v1/config"
	testname = fmt.Sprintf("getting exported config from https://%s/%s", client.APIHost, cfgpath)
	logger.Info(testname)
	err = client.RequestAndDecode(&cluster, "GET", cfgpath, nil, nil)
	if err != nil {
		errorf("%s: %s", testname, err)
	} else {
		infof("%s: ok, Collections.BlobSigning = %v", testname, cluster.Collections.BlobSigning)
	}

	var user arvados.User
	testname = "getting current user record"
	logger.Info(testname)
	err = client.RequestAndDecode(&user, "GET", "arvados/v1/users/current", nil, nil)
	if err != nil {
		errorf("%s: %s", testname, err)
		return 2
	} else {
		infof("%s: ok, uuid = %s", testname, user.UUID)
	}

	// uncomment to create some spurious errors
	// cluster.Services.WebDAVDownload.ExternalURL.Host = "0.0.0.0:9"

	// TODO: detect routing errors here, like finding wb2 at the
	// wb1 address.
	for _, svc := range []*arvados.Service{
		&cluster.Services.Keepproxy,
		&cluster.Services.WebDAV,
		&cluster.Services.WebDAVDownload,
		&cluster.Services.Websocket,
		&cluster.Services.Workbench1,
		&cluster.Services.Workbench2,
	} {
		testname = fmt.Sprintf("connecting to service endpoint %s", svc.ExternalURL)
		logger.Info(testname)
		u := svc.ExternalURL
		if strings.HasPrefix(u.Scheme, "ws") {
			// We can do a real websocket test elsewhere,
			// but for now we'll just check the https
			// connection.
			u.Scheme = "http" + u.Scheme[2:]
		}
		if svc == &cluster.Services.WebDAV && strings.HasPrefix(u.Host, "*") {
			u.Host = "d41d8cd98f00b204e9800998ecf8427e-0" + u.Host[1:]
		}
		req, err := http.NewRequest(http.MethodGet, u.String(), nil)
		if err != nil {
			errorf("%s: %s", testname, err)
			continue
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			errorf("%s: %s", testname, err)
			continue
		}
		resp.Body.Close()
		infof("%s: ok", testname)
	}

	for _, url := range []string{
		cluster.Services.Controller.ExternalURL.String(),
		cluster.Services.Keepproxy.ExternalURL.String() + "d41d8cd98f00b204e9800998ecf8427e+0",
		cluster.Services.WebDAVDownload.ExternalURL.String(),
	} {
		testname = fmt.Sprintf("checking CORS headers at %s", url)
		logger.Info(testname)
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			errorf("%s: %s", testname, err)
			continue
		}
		req.Header.Set("Origin", "https://example.com")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			errorf("%s: %s", testname, err)
			continue
		}
		if hdr := resp.Header.Get("Access-Control-Allow-Origin"); hdr != "*" {
			warnf("%s: expected \"Access-Control-Allow-Origin: *\", got %q", testname, hdr)
		} else {
			infof("%s: ok", testname)
		}
	}

	var keeplist arvados.KeepServiceList
	testname = "checking internal/external client detection"
	logger.Info(testname)
	err = client.RequestAndDecode(&keeplist, "GET", "arvados/v1/keep_services/accessible", nil, arvados.ListOptions{Limit: -1})
	if err != nil {
		errorf("%s: error getting keep services list: %s", testname, err)
	} else if len(keeplist.Items) == 0 {
		errorf("%s: controller did not return any keep services", testname)
	} else {
		found := map[string]int{}
		for _, ks := range keeplist.Items {
			found[ks.ServiceType]++
		}
		infof := infof
		isInternal := found["proxy"] == 0 && len(keeplist.Items) > 0
		isExternal := found["proxy"] > 0 && found["proxy"] == len(keeplist.Items)
		if (*checkInternal && !isInternal) || (*checkExternal && !isExternal) {
			infof = errorf
		}
		if isExternal {
			infof("%s: controller returned only proxy services, this host is considered \"external\"", testname)
		} else if isInternal {
			infof("%s: controller returned only non-proxy services, this host is considered \"internal\"", testname)
		} else {
			errorf("%s: controller returned both proxy and non-proxy services: %v", testname, found)
		}
	}

	var project arvados.Group
	var grplist arvados.GroupList
	testname = fmt.Sprintf("finding/creating %q project", diag.projectName)
	logger.Info(testname)
	err = client.RequestAndDecode(&grplist, "GET", "arvados/v1/groups", nil, arvados.ListOptions{
		Filters: []arvados.Filter{
			{"name", "=", diag.projectName},
			{"group_class", "=", "project"},
			{"owner_uuid", "=", user.UUID}},
		Limit: -1})
	if err != nil {
		errorf("%s: list groups: %s", testname, err)
	} else if len(grplist.Items) < 1 {
		infof("%s: list groups: ok, no results", testname)
		err = client.RequestAndDecode(&project, "POST", "arvados/v1/groups", nil, map[string]interface{}{"group": map[string]interface{}{
			"name":        diag.projectName,
			"group_class": "project",
		}})
		if err != nil {
			errorf("%s: create project: %s", testname, err)
		} else {
			infof("%s: created project, uuid = %s", testname, project.UUID)
		}
	} else {
		project = grplist.Items[0]
		infof("%s: ok, using existing project, uuid = %s", testname, project.UUID)
	}

	testname = "creating temporary collection"
	logger.Info(testname)
	var collection arvados.Collection
	err = client.RequestAndDecode(&collection, "POST", "arvados/v1/collections", nil, map[string]interface{}{
		"ensure_unique_name": true,
		"collection": map[string]interface{}{
			"name":     "test collection",
			"trash_at": time.Now().Add(time.Hour)}})
	if err != nil {
		errorf("%s: %s", testname, err)
	} else {
		infof("%s: ok, uuid = %s", testname, collection.UUID)
		defer func() {
			testname := "deleting temporary collection"
			logger.Info(testname)
			err := client.RequestAndDecode(nil, "DELETE", "arvados/v1/collections/"+collection.UUID, nil, nil)
			if err != nil {
				errorf("%s: %s", testname, err)
			} else {
				infof("%s: ok", testname)
			}
		}()
	}

	testname = "uploading file via webdav"
	logger.Info(testname)
	func() {
		if collection.UUID == "" {
			infof("%s: skipping, no test collection", testname)
			return
		}
		req, err := http.NewRequest("PUT", cluster.Services.WebDAVDownload.ExternalURL.String()+"c="+collection.UUID+"/testfile", bytes.NewBufferString("testfiledata"))
		if err != nil {
			errorf("%s: BUG? http.NewRequest: %s", testname, err)
			return
		}
		req.Header.Set("Authorization", "Bearer "+client.AuthToken)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			errorf("%s: error performing http request: %s", testname, err)
			return
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			errorf("%s: status %s", testname, resp.Status)
			return
		}
		infof("%s: ok, status %s", testname, resp.Status)
		err = client.RequestAndDecode(&collection, "GET", "arvados/v1/collections/"+collection.UUID, nil, nil)
		if err != nil {
			errorf("%s: get updated collection: %s", testname, err)
			return
		}
		infof("%s: get updated collection: ok, pdh %s", testname, collection.PortableDataHash)
	}()

	davurl := cluster.Services.WebDAV.ExternalURL
	testname = fmt.Sprintf("checking WebDAV ExternalURL wildcard (%s)", davurl)
	logger.Info(testname)
	if strings.HasPrefix(davurl.Host, "*--") || strings.HasPrefix(davurl.Host, "*.") {
		infof("%s: looks ok", testname)
	} else if davurl.Host == "" {
		warnf("%s: host missing - content previews will not work", testname)
	} else {
		warnf("%s: host has no leading wildcard - content previews will not work unless TrustAllContent==true", testname)
	}

	for _, trial := range []struct {
		status  int
		fileurl string
	}{
		{http.StatusNotFound, strings.Replace(davurl.String(), "*", "d41d8cd98f00b204e9800998ecf8427e-0", 1) + "foo"},
		{http.StatusNotFound, strings.Replace(davurl.String(), "*", "d41d8cd98f00b204e9800998ecf8427e-0", 1) + "testfile"},
		{http.StatusNotFound, cluster.Services.WebDAVDownload.ExternalURL.String() + "c=d41d8cd98f00b204e9800998ecf8427e+0/_/foo"},
		{http.StatusNotFound, cluster.Services.WebDAVDownload.ExternalURL.String() + "c=d41d8cd98f00b204e9800998ecf8427e+0/_/testfile"},
		{http.StatusOK, strings.Replace(davurl.String(), "*", strings.Replace(collection.PortableDataHash, "+", "-", -1), 1) + "testfile"},
		{http.StatusOK, cluster.Services.WebDAVDownload.ExternalURL.String() + "c=" + collection.UUID + "/_/testfile"},
	} {
		func() {
			testname := fmt.Sprintf("downloading from webdav (%s)", trial.fileurl)
			logger.Info(testname)
			if collection.UUID == "" {
				errorf("%s: skipping, no test collection", testname)
				return
			}
			req, err := http.NewRequest("GET", trial.fileurl, nil)
			if err != nil {
				errorf("%s: %s", testname, err)
				return
			}
			req.Header.Set("Authorization", "Bearer "+client.AuthToken)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				errorf("%s: %s", testname, err)
				return
			}
			defer resp.Body.Close()
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				errorf("%s: error reading response: %s", testname, err)
			}
			if resp.StatusCode != trial.status {
				errorf("%s: unexpected response status: %s", testname, resp.Status)
			} else if trial.status == http.StatusOK && string(body) != "testfiledata" {
				errorf("%s: unexpected response content: %q", testname, body)
			} else {
				infof("%s: ok", testname)
			}
		}()
	}

	var vm arvados.VirtualMachine
	var vmlist arvados.VirtualMachineList
	testname = "getting list of virtual machines"
	logger.Info(testname)
	err = client.RequestAndDecode(&vmlist, "GET", "arvados/v1/virtual_machines", nil, arvados.ListOptions{Limit: 999999})
	if err != nil {
		errorf("%s: %s", testname, err)
	} else if len(vmlist.Items) < 1 {
		errorf("%s: none found", testname)
	} else {
		vm = vmlist.Items[0]
		infof("%s: ok", testname)
	}

	testname = "getting workbench1 webshell page"
	logger.Info(testname)
	func() {
		if vm.UUID == "" {
			errorf("%s: skipping, no vm available", testname)
			return
		}
		webshelltermurl := cluster.Services.Workbench1.ExternalURL.String() + "virtual_machines/" + vm.UUID + "/webshell/testusername"
		debugf("%s: url %s", testname, webshelltermurl)
		req, err := http.NewRequest("GET", webshelltermurl, nil)
		if err != nil {
			errorf("%s: %s", testname, err)
			return
		}
		req.Header.Set("Authorization", "Bearer "+client.AuthToken)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			errorf("%s: %s", testname, err)
			return
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			errorf("%s: error reading response: %s", testname, err)
		}
		if resp.StatusCode != http.StatusOK {
			errorf("%s: unexpected response status: %s %q", testname, resp.Status, body)
			return
		}
		infof("%s: ok", testname)
	}()

	testname = "connecting to webshell service"
	logger.Info(testname)
	func() {
		ctx, cancel := context.WithDeadline(ctx, time.Now().Add(*timeout))
		defer cancel()
		if vm.UUID == "" {
			errorf("%s: skipping, no vm available", testname)
			return
		}
		u := cluster.Services.WebShell.ExternalURL
		webshellurl := u.String() + vm.Hostname + "?"
		if strings.HasPrefix(u.Host, "*") {
			u.Host = vm.Hostname + u.Host[1:]
			webshellurl = u.String() + "?"
		}
		debugf("%s: url %s", testname, webshellurl)
		req, err := http.NewRequestWithContext(ctx, "POST", webshellurl, bytes.NewBufferString(url.Values{
			"width":   {"80"},
			"height":  {"25"},
			"session": {"xyzzy"},
			"rooturl": {webshellurl},
		}.Encode()))
		if err != nil {
			errorf("%s: %s", testname, err)
			return
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			errorf("%s: %s", testname, err)
			return
		}
		defer resp.Body.Close()
		debugf("%s: response status %s", testname, resp.Status)
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			errorf("%s: error reading response: %s", testname, err)
		}
		debugf("%s: response body %q", testname, body)
		// We don't speak the protocol, so we get a 400 error
		// from the webshell server even if everything is
		// OK. Anything else (404, 502, ???) indicates a
		// problem.
		if resp.StatusCode != http.StatusBadRequest {
			errorf("%s: unexpected response status: %s, %q", testname, resp.Status, body)
			return
		}
		infof("%s: ok", testname)
	}()
	return 0
}
