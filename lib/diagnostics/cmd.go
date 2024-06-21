// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package diagnostics

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"git.arvados.org/arvados.git/lib/cmd"
	"git.arvados.org/arvados.git/lib/config"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"git.arvados.org/arvados.git/sdk/go/health"
	"github.com/sirupsen/logrus"
)

type Command struct{}

func (Command) RunCommand(prog string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	var diag diagnoser
	f := flag.NewFlagSet(prog, flag.ContinueOnError)
	f.StringVar(&diag.projectName, "project-name", "scratch area for diagnostics", "`name` of project to find/create in home project and use for temporary/test objects")
	f.StringVar(&diag.logLevel, "log-level", "info", "logging `level` (debug, info, warning, error)")
	f.StringVar(&diag.dockerImage, "docker-image", "", "`image` (tag or portable data hash) to use when running a test container, or \"hello-world\" to use embedded hello-world image (default: build a custom image containing this executable, and run diagnostics inside the container too)")
	f.StringVar(&diag.dockerImageFrom, "docker-image-from", "debian:stable-slim", "`base` image to use when building a custom image (see https://doc.arvados.org/main/admin/diagnostics.html#container-options)")
	f.BoolVar(&diag.checkInternal, "internal-client", false, "check that this host is considered an \"internal\" client")
	f.BoolVar(&diag.checkExternal, "external-client", false, "check that this host is considered an \"external\" client")
	f.BoolVar(&diag.verbose, "v", false, "verbose: include more information in report")
	f.IntVar(&diag.priority, "priority", 500, "priority for test container (1..1000, or 0 to skip)")
	f.DurationVar(&diag.timeout, "timeout", 10*time.Second, "timeout for http requests")
	if ok, code := cmd.ParseFlags(f, prog, args, "", stderr); !ok {
		return code
	}
	diag.stdout = stdout
	diag.stderr = stderr
	diag.logger = ctxlog.New(stdout, "text", diag.logLevel)
	diag.logger.SetFormatter(&logrus.TextFormatter{DisableTimestamp: true, DisableLevelTruncation: true, PadLevelText: true})
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

// docker save hello-world > hello-world.tar
//
//go:embed hello-world.tar
var HelloWorldDockerImage []byte

type diagnoser struct {
	stdout          io.Writer
	stderr          io.Writer
	logLevel        string
	priority        int
	projectName     string
	dockerImage     string
	dockerImageFrom string
	checkInternal   bool
	checkExternal   bool
	verbose         bool
	timeout         time.Duration
	logger          *logrus.Logger
	errors          []string
	done            map[int]bool
}

func (diag *diagnoser) debugf(f string, args ...interface{}) {
	diag.logger.Debugf("  ... "+f, args...)
}

func (diag *diagnoser) infof(f string, args ...interface{}) {
	diag.logger.Infof("  ... "+f, args...)
}

func (diag *diagnoser) verbosef(f string, args ...interface{}) {
	if diag.verbose {
		diag.logger.Infof("  ... "+f, args...)
	}
}

func (diag *diagnoser) warnf(f string, args ...interface{}) {
	diag.logger.Warnf("  ... "+f, args...)
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

	diag.logger.Infof("%4d: %s", id, title)
	t0 := time.Now()
	err := fn()
	elapsed := fmt.Sprintf("%d ms", time.Now().Sub(t0)/time.Millisecond)
	if err != nil {
		diag.errorf("%4d: %s (%s): %s", id, title, elapsed, err)
	} else {
		diag.logger.Debugf("%4d: %s (%s): ok", id, title, elapsed)
	}
}

func (diag *diagnoser) runtests() {
	client := arvados.NewClientFromEnv()
	// Disable auto-retry, use context instead
	client.Timeout = 0

	if client.APIHost == "" || client.AuthToken == "" {
		diag.errorf("ARVADOS_API_HOST and ARVADOS_API_TOKEN environment variables are not set -- aborting without running any tests")
		return
	}

	hostname, err := os.Hostname()
	if err != nil {
		diag.warnf("error getting hostname: %s")
	} else {
		diag.verbosef("hostname = %s", hostname)
	}

	diag.dotest(5, "running health check (same as `arvados-server check`)", func() error {
		ldr := config.NewLoader(&bytes.Buffer{}, ctxlog.New(&bytes.Buffer{}, "text", "info"))
		ldr.SetupFlags(flag.NewFlagSet("diagnostics", flag.ContinueOnError))
		cfg, err := ldr.Load()
		if err != nil {
			diag.infof("skipping because config could not be loaded: %s", err)
			return nil
		}
		cluster, err := cfg.GetCluster("")
		if err != nil {
			return err
		}
		if cluster.SystemRootToken != os.Getenv("ARVADOS_API_TOKEN") {
			return fmt.Errorf("diagnostics usage error: %s is readable but SystemRootToken does not match $ARVADOS_API_TOKEN (to fix, either run 'arvados-client sudo diagnostics' to load everything from config file, or set ARVADOS_CONFIG=- to load nothing from config file)", ldr.Path)
		}
		agg := &health.Aggregator{Cluster: cluster}
		resp := agg.ClusterHealth()
		for _, e := range resp.Errors {
			diag.errorf("health check: %s", e)
		}
		if len(resp.Errors) > 0 {
			diag.infof("consider running `arvados-server check -yaml` for a comprehensive report")
		}
		diag.verbosef("reported clock skew = %v", resp.ClockSkew)
		reported := map[string]bool{}
		for _, result := range resp.Checks {
			version := strings.SplitN(result.Metrics.Version, " (go", 2)[0]
			if version != "" && !reported[version] {
				diag.verbosef("arvados version = %s", version)
				reported[version] = true
			}
		}
		reported = map[string]bool{}
		for _, result := range resp.Checks {
			if result.Server != "" && !reported[result.Server] {
				diag.verbosef("http frontend version = %s", result.Server)
				reported[result.Server] = true
			}
		}
		reported = map[string]bool{}
		for _, result := range resp.Checks {
			if sha := result.ConfigSourceSHA256; sha != "" && !reported[sha] {
				diag.verbosef("config file sha256 = %s", sha)
				reported[sha] = true
			}
		}
		return nil
	})

	var dd arvados.DiscoveryDocument
	ddpath := "discovery/v1/apis/arvados/v1/rest"
	diag.dotest(10, fmt.Sprintf("getting discovery document from https://%s/%s", client.APIHost, ddpath), func() error {
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(diag.timeout))
		defer cancel()
		err := client.RequestAndDecodeContext(ctx, &dd, "GET", ddpath, nil, nil)
		if err != nil {
			return err
		}
		diag.verbosef("BlobSignatureTTL = %d", dd.BlobSignatureTTL)
		return nil
	})

	var cluster arvados.Cluster
	cfgpath := "arvados/v1/config"
	cfgOK := false
	diag.dotest(20, fmt.Sprintf("getting exported config from https://%s/%s", client.APIHost, cfgpath), func() error {
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(diag.timeout))
		defer cancel()
		err := client.RequestAndDecodeContext(ctx, &cluster, "GET", cfgpath, nil, nil)
		if err != nil {
			return err
		}
		diag.verbosef("Collections.BlobSigning = %v", cluster.Collections.BlobSigning)
		cfgOK = true
		return nil
	})

	var user arvados.User
	diag.dotest(30, "getting current user record", func() error {
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(diag.timeout))
		defer cancel()
		err := client.RequestAndDecodeContext(ctx, &user, "GET", "arvados/v1/users/current", nil, nil)
		if err != nil {
			return err
		}
		diag.verbosef("user uuid = %s", user.UUID)
		return nil
	})

	if !cfgOK {
		diag.errorf("cannot proceed without cluster config -- aborting without running any further tests")
		return
	}

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
			ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(diag.timeout))
			defer cancel()
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
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
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
			ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(diag.timeout))
			defer cancel()
			req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
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
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(diag.timeout))
		defer cancel()
		err := client.RequestAndDecodeContext(ctx, &keeplist, "GET", "arvados/v1/keep_services/accessible", nil, arvados.ListOptions{Limit: 999999})
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
			diag.infof("controller returned only proxy services, this host is treated as \"external\"")
		} else if isInternal {
			diag.infof("controller returned only non-proxy services, this host is treated as \"internal\"")
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
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(diag.timeout))
		defer cancel()
		var grplist arvados.GroupList
		err := client.RequestAndDecodeContext(ctx, &grplist, "GET", "arvados/v1/groups", nil, arvados.ListOptions{
			Filters: []arvados.Filter{
				{"name", "=", diag.projectName},
				{"group_class", "=", "project"},
				{"owner_uuid", "=", user.UUID}},
			Limit: 999999})
		if err != nil {
			return fmt.Errorf("list groups: %s", err)
		}
		if len(grplist.Items) > 0 {
			project = grplist.Items[0]
			diag.verbosef("using existing project, uuid = %s", project.UUID)
			return nil
		}
		diag.debugf("list groups: ok, no results")
		err = client.RequestAndDecodeContext(ctx, &project, "POST", "arvados/v1/groups", nil, map[string]interface{}{"group": map[string]interface{}{
			"name":        diag.projectName,
			"group_class": "project",
		}})
		if err != nil {
			return fmt.Errorf("create project: %s", err)
		}
		diag.verbosef("created project, uuid = %s", project.UUID)
		return nil
	})

	var collection arvados.Collection
	diag.dotest(90, "creating temporary collection", func() error {
		if project.UUID == "" {
			return fmt.Errorf("skipping, no project to work in")
		}
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(diag.timeout))
		defer cancel()
		err := client.RequestAndDecodeContext(ctx, &collection, "POST", "arvados/v1/collections", nil, map[string]interface{}{
			"ensure_unique_name": true,
			"collection": map[string]interface{}{
				"owner_uuid": project.UUID,
				"name":       "test collection",
				"trash_at":   time.Now().Add(time.Hour)}})
		if err != nil {
			return err
		}
		diag.verbosef("ok, uuid = %s", collection.UUID)
		return nil
	})

	if collection.UUID != "" {
		defer func() {
			diag.dotest(9990, "deleting temporary collection", func() error {
				ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(diag.timeout))
				defer cancel()
				return client.RequestAndDecodeContext(ctx, nil, "DELETE", "arvados/v1/collections/"+collection.UUID, nil, nil)
			})
		}()
	}

	tempdir, err := ioutil.TempDir("", "arvados-diagnostics")
	if err != nil {
		diag.errorf("error creating temp dir: %s", err)
		return
	}
	defer os.RemoveAll(tempdir)

	var imageSHA2 string
	var dockerImageData []byte
	if diag.dockerImage != "" || diag.priority < 1 {
		// We won't be using the self-built docker image, so
		// don't build it.  But we will write the embedded
		// "hello-world" image to our test collection to test
		// upload/download, whether or not we're using it as a
		// docker image.
		dockerImageData = HelloWorldDockerImage

		if diag.priority > 0 {
			imageSHA2, err = getSHA2FromImageData(dockerImageData)
			if err != nil {
				diag.errorf("internal error/bug: %s", err)
				return
			}
		}
	} else if selfbin, err := os.Readlink("/proc/self/exe"); err != nil {
		diag.errorf("readlink /proc/self/exe: %s", err)
		return
	} else if selfbindata, err := os.ReadFile(selfbin); err != nil {
		diag.errorf("error reading %s: %s", selfbin, err)
		return
	} else {
		selfbinSha := fmt.Sprintf("%x", sha256.Sum256(selfbindata))
		tag := "arvados-client-diagnostics:" + selfbinSha[:9]
		err := os.WriteFile(tempdir+"/arvados-client", selfbindata, 0777)
		if err != nil {
			diag.errorf("error writing %s: %s", tempdir+"/arvados-client", err)
			return
		}

		dockerfile := "FROM " + diag.dockerImageFrom + "\n"
		dockerfile += "RUN apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install --yes --no-install-recommends libfuse2 ca-certificates && apt-get clean\n"
		dockerfile += "COPY /arvados-client /arvados-client\n"
		cmd := exec.Command("docker", "build", "--tag", tag, "-f", "-", tempdir)
		cmd.Stdin = strings.NewReader(dockerfile)
		cmd.Stdout = diag.stderr
		cmd.Stderr = diag.stderr
		err = cmd.Run()
		if err != nil {
			diag.errorf("error building docker image: %s", err)
			return
		}
		checkversion, err := exec.Command("docker", "run", tag, "/arvados-client", "version").CombinedOutput()
		if err != nil {
			diag.errorf("docker image does not seem to work: %s", err)
			return
		}
		diag.infof("arvados-client version: %s", checkversion)

		buf, err := exec.Command("docker", "inspect", "--format={{.Id}}", tag).Output()
		if err != nil {
			diag.errorf("docker inspect --format={{.Id}} %s: %s", tag, err)
			return
		}
		imageSHA2 = min64HexDigits.FindString(string(buf))
		if len(imageSHA2) != 64 {
			diag.errorf("docker inspect --format={{.Id}} output %q does not seem to contain sha256 digest", buf)
			return
		}

		buf, err = exec.Command("docker", "save", tag).Output()
		if err != nil {
			diag.errorf("docker save %s: %s", tag, err)
			return
		}
		diag.infof("docker image size is %d", len(buf))
		dockerImageData = buf
	}

	tarfilename := "sha256:" + imageSHA2 + ".tar"

	diag.dotest(100, "uploading file via webdav", func() error {
		timeout := diag.timeout
		if len(dockerImageData) > 10<<20 && timeout < time.Minute {
			// Extend the normal http timeout if we're
			// uploading a substantial docker image.
			timeout = time.Minute
		}
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(timeout))
		defer cancel()
		if collection.UUID == "" {
			return fmt.Errorf("skipping, no test collection")
		}
		t0 := time.Now()
		req, err := http.NewRequestWithContext(ctx, "PUT", cluster.Services.WebDAVDownload.ExternalURL.String()+"c="+collection.UUID+"/"+tarfilename, bytes.NewReader(dockerImageData))
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
		diag.verbosef("upload ok, status %s, %f MB/s", resp.Status, float64(len(dockerImageData))/time.Since(t0).Seconds()/1000000)
		err = client.RequestAndDecodeContext(ctx, &collection, "GET", "arvados/v1/collections/"+collection.UUID, nil, nil)
		if err != nil {
			return fmt.Errorf("get updated collection: %s", err)
		}
		diag.verbosef("upload pdh %s", collection.PortableDataHash)
		return nil
	})

	davurl := cluster.Services.WebDAV.ExternalURL
	davWildcard := strings.HasPrefix(davurl.Host, "*--") || strings.HasPrefix(davurl.Host, "*.")
	diag.dotest(110, fmt.Sprintf("checking WebDAV ExternalURL wildcard (%s)", davurl), func() error {
		if davurl.Host == "" {
			return fmt.Errorf("host missing - content previews will not work")
		}
		if !davWildcard && !cluster.Collections.TrustAllContent {
			diag.warnf("WebDAV ExternalURL has no leading wildcard and TrustAllContent==false - content previews will not work")
		}
		return nil
	})

	for i, trial := range []struct {
		needcoll     bool
		needWildcard bool
		status       int
		fileurl      string
	}{
		{false, false, http.StatusNotFound, strings.Replace(davurl.String(), "*", "d41d8cd98f00b204e9800998ecf8427e-0", 1) + "foo"},
		{false, false, http.StatusNotFound, strings.Replace(davurl.String(), "*", "d41d8cd98f00b204e9800998ecf8427e-0", 1) + tarfilename},
		{false, false, http.StatusNotFound, cluster.Services.WebDAVDownload.ExternalURL.String() + "c=d41d8cd98f00b204e9800998ecf8427e+0/_/foo"},
		{false, false, http.StatusNotFound, cluster.Services.WebDAVDownload.ExternalURL.String() + "c=d41d8cd98f00b204e9800998ecf8427e+0/_/" + tarfilename},
		{true, true, http.StatusOK, strings.Replace(davurl.String(), "*", strings.Replace(collection.PortableDataHash, "+", "-", -1), 1) + tarfilename},
		{true, false, http.StatusOK, cluster.Services.WebDAVDownload.ExternalURL.String() + "c=" + collection.UUID + "/_/" + tarfilename},
	} {
		diag.dotest(120+i, fmt.Sprintf("downloading from webdav (%s)", trial.fileurl), func() error {
			if trial.needWildcard && !davWildcard {
				diag.warnf("skipping collection-id-in-vhost test because WebDAV ExternalURL has no leading wildcard")
				return nil
			}
			ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(diag.timeout))
			defer cancel()
			if trial.needcoll && collection.UUID == "" {
				return fmt.Errorf("skipping, no test collection")
			}
			req, err := http.NewRequestWithContext(ctx, "GET", trial.fileurl, nil)
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
			if trial.status == http.StatusOK && !bytes.Equal(body, dockerImageData) {
				excerpt := body
				if len(excerpt) > 128 {
					excerpt = append([]byte(nil), body[:128]...)
					excerpt = append(excerpt, []byte("[...]")...)
				}
				return fmt.Errorf("unexpected response content: len %d, %q", len(body), excerpt)
			}
			return nil
		})
	}

	var vm arvados.VirtualMachine
	diag.dotest(130, "getting list of virtual machines", func() error {
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(diag.timeout))
		defer cancel()
		var vmlist arvados.VirtualMachineList
		err := client.RequestAndDecodeContext(ctx, &vmlist, "GET", "arvados/v1/virtual_machines", nil, arvados.ListOptions{Limit: 999999})
		if err != nil {
			return err
		}
		if len(vmlist.Items) < 1 {
			diag.warnf("no VMs found")
		} else {
			vm = vmlist.Items[0]
		}
		return nil
	})

	diag.dotest(150, "connecting to webshell service", func() error {
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(diag.timeout))
		defer cancel()
		if vm.UUID == "" {
			diag.warnf("skipping, no vm available")
			return nil
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

	diag.dotest(160, "running a container", func() error {
		if diag.priority < 1 {
			diag.infof("skipping (use priority > 0 if you want to run a container)")
			return nil
		}
		if project.UUID == "" {
			return fmt.Errorf("skipping, no project to work in")
		}

		timestamp := time.Now().Format(time.RFC3339)

		var ctrCommand []string
		switch diag.dockerImage {
		case "":
			if collection.UUID == "" {
				return fmt.Errorf("skipping, no test collection to use as docker image")
			}
			diag.dockerImage = collection.PortableDataHash
			ctrCommand = []string{"/arvados-client", "diagnostics",
				"-priority=0", // don't run a container
				"-log-level=" + diag.logLevel,
				"-internal-client=true"}
		case "hello-world":
			if collection.UUID == "" {
				return fmt.Errorf("skipping, no test collection to use as docker image")
			}
			diag.dockerImage = collection.PortableDataHash
			ctrCommand = []string{"/hello"}
		default:
			ctrCommand = []string{"echo", timestamp}
		}

		var cr arvados.ContainerRequest
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(diag.timeout))
		defer cancel()

		err := client.RequestAndDecodeContext(ctx, &cr, "POST", "arvados/v1/container_requests", nil, map[string]interface{}{"container_request": map[string]interface{}{
			"owner_uuid":      project.UUID,
			"name":            fmt.Sprintf("diagnostics container request %s", timestamp),
			"container_image": diag.dockerImage,
			"command":         ctrCommand,
			"use_existing":    false,
			"output_path":     "/mnt/output",
			"output_name":     fmt.Sprintf("diagnostics output %s", timestamp),
			"priority":        diag.priority,
			"state":           arvados.ContainerRequestStateCommitted,
			"mounts": map[string]map[string]interface{}{
				"/mnt/output": {
					"kind":     "collection",
					"writable": true,
				},
			},
			"runtime_constraints": arvados.RuntimeConstraints{
				API:          true,
				VCPUs:        1,
				RAM:          128 << 20,
				KeepCacheRAM: 64 << 20,
			},
		}})
		if err != nil {
			return err
		}
		diag.infof("container request uuid = %s", cr.UUID)
		diag.verbosef("container uuid = %s", cr.ContainerUUID)

		timeout := 10 * time.Minute
		diag.infof("container request submitted, waiting up to %v for container to run", arvados.Duration(timeout))
		deadline := time.Now().Add(timeout)

		var c arvados.Container
		for ; cr.State != arvados.ContainerRequestStateFinal && time.Now().Before(deadline); time.Sleep(2 * time.Second) {
			ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(diag.timeout))
			defer cancel()

			crStateWas := cr.State
			err := client.RequestAndDecodeContext(ctx, &cr, "GET", "arvados/v1/container_requests/"+cr.UUID, nil, nil)
			if err != nil {
				return err
			}
			if cr.State != crStateWas {
				diag.debugf("container request state = %s", cr.State)
			}

			cStateWas := c.State
			err = client.RequestAndDecodeContext(ctx, &c, "GET", "arvados/v1/containers/"+cr.ContainerUUID, nil, nil)
			if err != nil {
				return err
			}
			if c.State != cStateWas {
				diag.debugf("container state = %s", c.State)
			}

			cancel()
		}

		if cr.State != arvados.ContainerRequestStateFinal {
			err := client.RequestAndDecodeContext(context.Background(), &cr, "PATCH", "arvados/v1/container_requests/"+cr.UUID, nil, map[string]interface{}{
				"container_request": map[string]interface{}{
					"priority": 0,
				}})
			if err != nil {
				diag.infof("error canceling container request %s: %s", cr.UUID, err)
			} else {
				diag.debugf("canceled container request %s", cr.UUID)
			}
			return fmt.Errorf("timed out waiting for container to finish; container request %s state was %q, container %s state was %q", cr.UUID, cr.State, c.UUID, c.State)
		}
		if c.State != arvados.ContainerStateComplete {
			return fmt.Errorf("container request %s is final but container %s did not complete: container state = %q", cr.UUID, cr.ContainerUUID, c.State)
		}
		if c.ExitCode != 0 {
			return fmt.Errorf("container exited %d", c.ExitCode)
		}
		return nil
	})
}

func getSHA2FromImageData(dockerImageData []byte) (string, error) {
	tr := tar.NewReader(bytes.NewReader(dockerImageData))
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return "", fmt.Errorf("cannot find manifest.json in docker image tar file")
		}
		if err != nil {
			return "", fmt.Errorf("cannot read docker image tar file: %s", err)
		}
		if hdr.Name != "manifest.json" {
			continue
		}
		var manifest []struct {
			Config string
		}
		err = json.NewDecoder(tr).Decode(&manifest)
		if err != nil {
			return "", fmt.Errorf("cannot read manifest.json from docker image tar file: %s", err)
		}
		if len(manifest) == 0 {
			return "", fmt.Errorf("manifest.json is empty")
		}
		s := min64HexDigits.FindString(manifest[0].Config)
		if len(s) != 64 {
			return "", fmt.Errorf("found manifest.json but .[0].Config %q does not seem to contain sha256 digest", manifest[0].Config)
		}
		return s, nil
	}
}

var min64HexDigits = regexp.MustCompile(`[0-9a-f]{64,}`)
