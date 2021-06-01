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
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"github.com/sirupsen/logrus"
)

type Command struct{}

func (cmd Command) RunCommand(prog string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	var diag diagnoser
	f := flag.NewFlagSet(prog, flag.ContinueOnError)
	f.StringVar(&diag.projectName, "project-name", "scratch area for diagnostics", "name of project to find/create in home project and use for temporary/test objects")
	f.StringVar(&diag.logLevel, "log-level", "info", "logging level (debug, info, warning, error)")
	f.BoolVar(&diag.checkInternal, "internal-client", false, "check that this host is considered an \"internal\" client")
	f.BoolVar(&diag.checkExternal, "external-client", false, "check that this host is considered an \"external\" client")
	f.DurationVar(&diag.timeout, "timeout", 10*time.Second, "timeout for http requests")
	err := f.Parse(args)
	if err == flag.ErrHelp {
		return 0
	} else if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	diag.logger = ctxlog.New(stdout, "text", diag.logLevel)
	diag.logger.SetFormatter(&logrus.TextFormatter{DisableTimestamp: true, DisableLevelTruncation: true})
	diag.runtests()
	if len(diag.errors) == 0 {
		diag.logger.Info("--- no errors ---")
		return 0
	} else {
		if diag.logger.Level > logrus.ErrorLevel {
			fmt.Fprint(stdout, "\n--- cut here --- error summary ---\n\n")
			for _, e := range diag.errors {
				diag.logger.Error(e)
			}
		}
		return 1
	}
}

type diagnoser struct {
	stdout        io.Writer
	stderr        io.Writer
	logLevel      string
	projectName   string
	checkInternal bool
	checkExternal bool
	timeout       time.Duration
	logger        *logrus.Logger
	errors        []string
	done          map[int]bool
}

func (diag *diagnoser) debugf(f string, args ...interface{}) {
	diag.logger.Debugf(f, args...)
}

func (diag *diagnoser) infof(f string, args ...interface{}) {
	diag.logger.Infof(f, args...)
}

func (diag *diagnoser) warnf(f string, args ...interface{}) {
	diag.logger.Warnf(f, args...)
}

func (diag *diagnoser) errorf(f string, args ...interface{}) {
	diag.logger.Errorf(f, args...)
	diag.errors = append(diag.errors, fmt.Sprintf(f, args...))
}

// Run the given func, logging appropriate messages before and after,
// adding timing info, etc.
//
// The id argument should be unique among tests, and shouldn't change
// when other tests are added/removed.
func (diag *diagnoser) dotest(id int, title string, fn func() error) {
	if diag.done == nil {
		diag.done = map[int]bool{}
	} else if diag.done[id] {
		diag.errorf("(bug) reused test id %d", id)
	}
	diag.done[id] = true

	diag.infof("%d %s", id, title)
	t0 := time.Now()
	err := fn()
	elapsed := fmt.Sprintf("%.0dms", time.Now().Sub(t0)/time.Millisecond)
	if err != nil {
		diag.errorf("%s (%s): %s", title, elapsed, err)
	}
	diag.debugf("%d %s (%s): ok", id, title, elapsed)
}

func (diag *diagnoser) runtests() {
	client := arvados.NewClientFromEnv()

	if client.APIHost == "" || client.AuthToken == "" {
		diag.errorf("ARVADOS_API_HOST and ARVADOS_API_TOKEN environment variables are not set -- aborting without running any tests")
		return
	}

	var dd arvados.DiscoveryDocument
	ddpath := "discovery/v1/apis/arvados/v1/rest"
	diag.dotest(10, fmt.Sprintf("getting discovery document from https://%s/%s", client.APIHost, ddpath), func() error {
		err := client.RequestAndDecode(&dd, "GET", ddpath, nil, nil)
		if err != nil {
			return err
		}
		diag.debugf("BlobSignatureTTL = %d", dd.BlobSignatureTTL)
		return nil
	})

	var cluster arvados.Cluster
	cfgpath := "arvados/v1/config"
	diag.dotest(20, fmt.Sprintf("getting exported config from https://%s/%s", client.APIHost, cfgpath), func() error {
		err := client.RequestAndDecode(&cluster, "GET", cfgpath, nil, nil)
		if err != nil {
			return err
		}
		diag.debugf("Collections.BlobSigning = %v", cluster.Collections.BlobSigning)
		return nil
	})

	var user arvados.User
	diag.dotest(30, "getting current user record", func() error {
		err := client.RequestAndDecode(&user, "GET", "arvados/v1/users/current", nil, nil)
		if err != nil {
			return err
		}
		diag.debugf("user uuid = %s", user.UUID)
		return nil
	})

	// uncomment to create some spurious errors
	// cluster.Services.WebDAVDownload.ExternalURL.Host = "0.0.0.0:9"

	// TODO: detect routing errors here, like finding wb2 at the
	// wb1 address.
	for i, svc := range []*arvados.Service{
		&cluster.Services.Keepproxy,
		&cluster.Services.WebDAV,
		&cluster.Services.WebDAVDownload,
		&cluster.Services.Websocket,
		&cluster.Services.Workbench1,
		&cluster.Services.Workbench2,
	} {
		diag.dotest(40+i, fmt.Sprintf("connecting to service endpoint %s", svc.ExternalURL), func() error {
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
				return err
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return err
			}
			resp.Body.Close()
			return nil
		})
	}

	for i, url := range []string{
		cluster.Services.Controller.ExternalURL.String(),
		cluster.Services.Keepproxy.ExternalURL.String() + "d41d8cd98f00b204e9800998ecf8427e+0",
		cluster.Services.WebDAVDownload.ExternalURL.String(),
	} {
		diag.dotest(50+i, fmt.Sprintf("checking CORS headers at %s", url), func() error {
			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				return err
			}
			req.Header.Set("Origin", "https://example.com")
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return err
			}
			if hdr := resp.Header.Get("Access-Control-Allow-Origin"); hdr != "*" {
				return fmt.Errorf("expected \"Access-Control-Allow-Origin: *\", got %q", hdr)
			}
			return nil
		})
	}

	var keeplist arvados.KeepServiceList
	diag.dotest(60, "checking internal/external client detection", func() error {
		err := client.RequestAndDecode(&keeplist, "GET", "arvados/v1/keep_services/accessible", nil, arvados.ListOptions{Limit: -1})
		if err != nil {
			return fmt.Errorf("error getting keep services list: %s", err)
		} else if len(keeplist.Items) == 0 {
			return fmt.Errorf("controller did not return any keep services")
		}
		found := map[string]int{}
		for _, ks := range keeplist.Items {
			found[ks.ServiceType]++
		}
		isInternal := found["proxy"] == 0 && len(keeplist.Items) > 0
		isExternal := found["proxy"] > 0 && found["proxy"] == len(keeplist.Items)
		if isExternal {
			diag.debugf("controller returned only proxy services, this host is treated as \"external\"")
		} else if isInternal {
			diag.debugf("controller returned only non-proxy services, this host is treated as \"internal\"")
		}
		if (diag.checkInternal && !isInternal) || (diag.checkExternal && !isExternal) {
			return fmt.Errorf("expecting internal=%v external=%v, but found internal=%v external=%v", diag.checkInternal, diag.checkExternal, isInternal, isExternal)
		}
		return nil
	})

	for i, ks := range keeplist.Items {
		u := url.URL{
			Scheme: "http",
			Host:   net.JoinHostPort(ks.ServiceHost, fmt.Sprintf("%d", ks.ServicePort)),
			Path:   "/",
		}
		if ks.ServiceSSLFlag {
			u.Scheme = "https"
		}
		diag.dotest(61+i, fmt.Sprintf("reading+writing via keep service at %s", u.String()), func() error {
			ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(diag.timeout))
			defer cancel()
			req, err := http.NewRequestWithContext(ctx, "PUT", u.String()+"d41d8cd98f00b204e9800998ecf8427e", nil)
			if err != nil {
				return err
			}
			req.Header.Set("Authorization", "Bearer "+client.AuthToken)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("reading response body: %s", err)
			}
			loc := strings.TrimSpace(string(body))
			if !strings.HasPrefix(loc, "d41d8") {
				return fmt.Errorf("unexpected response from write: %q", body)
			}

			req, err = http.NewRequestWithContext(ctx, "GET", u.String()+loc, nil)
			if err != nil {
				return err
			}
			req.Header.Set("Authorization", "Bearer "+client.AuthToken)
			resp, err = http.DefaultClient.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			body, err = ioutil.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("reading response body: %s", err)
			}
			if len(body) != 0 {
				return fmt.Errorf("unexpected response from read: %q", body)
			}

			return nil
		})
	}

	var project arvados.Group
	diag.dotest(80, fmt.Sprintf("finding/creating %q project", diag.projectName), func() error {
		var grplist arvados.GroupList
		err := client.RequestAndDecode(&grplist, "GET", "arvados/v1/groups", nil, arvados.ListOptions{
			Filters: []arvados.Filter{
				{"name", "=", diag.projectName},
				{"group_class", "=", "project"},
				{"owner_uuid", "=", user.UUID}},
			Limit: -1})
		if err != nil {
			return fmt.Errorf("list groups: %s", err)
		}
		if len(grplist.Items) > 0 {
			project = grplist.Items[0]
			diag.debugf("using existing project, uuid = %s", project.UUID)
			return nil
		}
		diag.debugf("list groups: ok, no results")
		err = client.RequestAndDecode(&project, "POST", "arvados/v1/groups", nil, map[string]interface{}{"group": map[string]interface{}{
			"name":        diag.projectName,
			"group_class": "project",
		}})
		if err != nil {
			return fmt.Errorf("create project: %s", err)
		}
		diag.debugf("created project, uuid = %s", project.UUID)
		return nil
	})

	var collection arvados.Collection
	diag.dotest(90, "creating temporary collection", func() error {
		err := client.RequestAndDecode(&collection, "POST", "arvados/v1/collections", nil, map[string]interface{}{
			"ensure_unique_name": true,
			"collection": map[string]interface{}{
				"name":     "test collection",
				"trash_at": time.Now().Add(time.Hour)}})
		if err != nil {
			return err
		}
		diag.debugf("ok, uuid = %s", collection.UUID)
		return nil
	})

	if collection.UUID != "" {
		defer func() {
			diag.dotest(9990, "deleting temporary collection", func() error {
				return client.RequestAndDecode(nil, "DELETE", "arvados/v1/collections/"+collection.UUID, nil, nil)
			})
		}()
	}

	diag.dotest(100, "uploading file via webdav", func() error {
		if collection.UUID == "" {
			return fmt.Errorf("skipping, no test collection")
		}
		req, err := http.NewRequest("PUT", cluster.Services.WebDAVDownload.ExternalURL.String()+"c="+collection.UUID+"/testfile", bytes.NewBufferString("testfiledata"))
		if err != nil {
			return fmt.Errorf("BUG? http.NewRequest: %s", err)
		}
		req.Header.Set("Authorization", "Bearer "+client.AuthToken)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return fmt.Errorf("error performing http request: %s", err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			return fmt.Errorf("status %s", resp.Status)
		}
		diag.debugf("ok, status %s", resp.Status)
		err = client.RequestAndDecode(&collection, "GET", "arvados/v1/collections/"+collection.UUID, nil, nil)
		if err != nil {
			return fmt.Errorf("get updated collection: %s", err)
		}
		diag.debugf("ok, pdh %s", collection.PortableDataHash)
		return nil
	})

	davurl := cluster.Services.WebDAV.ExternalURL
	diag.dotest(110, fmt.Sprintf("checking WebDAV ExternalURL wildcard (%s)", davurl), func() error {
		if davurl.Host == "" {
			return fmt.Errorf("host missing - content previews will not work")
		}
		if !strings.HasPrefix(davurl.Host, "*--") && !strings.HasPrefix(davurl.Host, "*.") && !cluster.Collections.TrustAllContent {
			diag.warnf("WebDAV ExternalURL has no leading wildcard and TrustAllContent==false - content previews will not work")
		}
		return nil
	})

	for i, trial := range []struct {
		needcoll bool
		status   int
		fileurl  string
	}{
		{false, http.StatusNotFound, strings.Replace(davurl.String(), "*", "d41d8cd98f00b204e9800998ecf8427e-0", 1) + "foo"},
		{false, http.StatusNotFound, strings.Replace(davurl.String(), "*", "d41d8cd98f00b204e9800998ecf8427e-0", 1) + "testfile"},
		{false, http.StatusNotFound, cluster.Services.WebDAVDownload.ExternalURL.String() + "c=d41d8cd98f00b204e9800998ecf8427e+0/_/foo"},
		{false, http.StatusNotFound, cluster.Services.WebDAVDownload.ExternalURL.String() + "c=d41d8cd98f00b204e9800998ecf8427e+0/_/testfile"},
		{true, http.StatusOK, strings.Replace(davurl.String(), "*", strings.Replace(collection.PortableDataHash, "+", "-", -1), 1) + "testfile"},
		{true, http.StatusOK, cluster.Services.WebDAVDownload.ExternalURL.String() + "c=" + collection.UUID + "/_/testfile"},
	} {
		diag.dotest(120+i, fmt.Sprintf("downloading from webdav (%s)", trial.fileurl), func() error {
			if trial.needcoll && collection.UUID == "" {
				return fmt.Errorf("skipping, no test collection")
			}
			req, err := http.NewRequest("GET", trial.fileurl, nil)
			if err != nil {
				return err
			}
			req.Header.Set("Authorization", "Bearer "+client.AuthToken)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("reading response: %s", err)
			}
			if resp.StatusCode != trial.status {
				return fmt.Errorf("unexpected response status: %s", resp.Status)
			}
			if trial.status == http.StatusOK && string(body) != "testfiledata" {
				return fmt.Errorf("unexpected response content: %q", body)
			}
			return nil
		})
	}

	var vm arvados.VirtualMachine
	diag.dotest(130, "getting list of virtual machines", func() error {
		var vmlist arvados.VirtualMachineList
		err := client.RequestAndDecode(&vmlist, "GET", "arvados/v1/virtual_machines", nil, arvados.ListOptions{Limit: 999999})
		if err != nil {
			return err
		}
		if len(vmlist.Items) < 1 {
			return fmt.Errorf("no VMs found")
		}
		vm = vmlist.Items[0]
		return nil
	})

	diag.dotest(140, "getting workbench1 webshell page", func() error {
		if vm.UUID == "" {
			return fmt.Errorf("skipping, no vm available")
		}
		webshelltermurl := cluster.Services.Workbench1.ExternalURL.String() + "virtual_machines/" + vm.UUID + "/webshell/testusername"
		diag.debugf("url %s", webshelltermurl)
		req, err := http.NewRequest("GET", webshelltermurl, nil)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+client.AuthToken)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("reading response: %s", err)
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("unexpected response status: %s %q", resp.Status, body)
		}
		return nil
	})

	diag.dotest(150, "connecting to webshell service", func() error {
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(diag.timeout))
		defer cancel()
		if vm.UUID == "" {
			return fmt.Errorf("skipping, no vm available")
		}
		u := cluster.Services.WebShell.ExternalURL
		webshellurl := u.String() + vm.Hostname + "?"
		if strings.HasPrefix(u.Host, "*") {
			u.Host = vm.Hostname + u.Host[1:]
			webshellurl = u.String() + "?"
		}
		diag.debugf("url %s", webshellurl)
		req, err := http.NewRequestWithContext(ctx, "POST", webshellurl, bytes.NewBufferString(url.Values{
			"width":   {"80"},
			"height":  {"25"},
			"session": {"xyzzy"},
			"rooturl": {webshellurl},
		}.Encode()))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		diag.debugf("response status %s", resp.Status)
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("reading response: %s", err)
		}
		diag.debugf("response body %q", body)
		// We don't speak the protocol, so we get a 400 error
		// from the webshell server even if everything is
		// OK. Anything else (404, 502, ???) indicates a
		// problem.
		if resp.StatusCode != http.StatusBadRequest {
			return fmt.Errorf("unexpected response status: %s, %q", resp.Status, body)
		}
		return nil
	})
}
