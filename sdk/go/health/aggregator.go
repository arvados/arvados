// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package health

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"git.arvados.org/arvados.git/lib/cmd"
	"git.arvados.org/arvados.git/lib/config"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/auth"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"github.com/ghodss/yaml"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

const (
	defaultTimeout = arvados.Duration(2 * time.Second)
	maxClockSkew   = time.Minute
)

// Aggregator implements service.Handler. It handles "GET /_health/all"
// by checking the health of all configured services on the cluster
// and responding 200 if everything is healthy.
type Aggregator struct {
	setupOnce  sync.Once
	httpClient *http.Client
	timeout    arvados.Duration

	Cluster *arvados.Cluster

	// If non-nil, Log is called after handling each request.
	Log func(*http.Request, error)

	// If non-nil, report clock skew on each health-check.
	MetricClockSkew prometheus.Gauge
}

func (agg *Aggregator) setup() {
	agg.httpClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: agg.Cluster.TLS.Insecure,
			},
		},
	}
	if agg.timeout == 0 {
		// this is always the case, except in the test suite
		agg.timeout = defaultTimeout
	}
}

func (agg *Aggregator) CheckHealth() error {
	return nil
}

func (agg *Aggregator) Done() <-chan struct{} {
	return nil
}

func (agg *Aggregator) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	agg.setupOnce.Do(agg.setup)
	sendErr := func(statusCode int, err error) {
		resp.WriteHeader(statusCode)
		json.NewEncoder(resp).Encode(map[string]string{"error": err.Error()})
		if agg.Log != nil {
			agg.Log(req, err)
		}
	}

	resp.Header().Set("Content-Type", "application/json")

	if !agg.checkAuth(req) {
		sendErr(http.StatusUnauthorized, errUnauthorized)
		return
	}
	if req.URL.Path == "/_health/all" {
		json.NewEncoder(resp).Encode(agg.ClusterHealth())
	} else if req.URL.Path == "/_health/ping" {
		resp.Write(healthyBody)
	} else {
		sendErr(http.StatusNotFound, errNotFound)
		return
	}
	if agg.Log != nil {
		agg.Log(req, nil)
	}
}

type ClusterHealthResponse struct {
	// "OK" if all needed services are OK, otherwise "ERROR".
	Health string

	// An entry for each known health check of each known instance
	// of each needed component: "instance of service S on node N
	// reports health-check C is OK."
	Checks map[string]CheckResult

	// An entry for each service type: "service S is OK." This
	// exposes problems that can't be expressed in Checks, like
	// "service S is needed, but isn't configured to run
	// anywhere."
	Services map[arvados.ServiceName]ServiceHealth

	// Difference between min/max timestamps in individual
	// health-check responses.
	ClockSkew arvados.Duration

	Errors []string
}

type CheckResult struct {
	Health         string
	Error          string                 `json:",omitempty"`
	HTTPStatusCode int                    `json:",omitempty"`
	Response       map[string]interface{} `json:",omitempty"`
	ResponseTime   json.Number
	ClockTime      time.Time
	Metrics
	respTime time.Duration
}

type Metrics struct {
	ConfigSourceTimestamp time.Time
	ConfigSourceSHA256    string
	Version               string
}

type ServiceHealth struct {
	Health string // "OK", "ERROR", or "SKIP"
	N      int
}

func (agg *Aggregator) ClusterHealth() ClusterHealthResponse {
	agg.setupOnce.Do(agg.setup)
	resp := ClusterHealthResponse{
		Health:   "OK",
		Checks:   make(map[string]CheckResult),
		Services: make(map[arvados.ServiceName]ServiceHealth),
	}

	mtx := sync.Mutex{}
	wg := sync.WaitGroup{}
	for svcName, svc := range agg.Cluster.Services.Map() {
		// Ensure svc is listed in resp.Services.
		mtx.Lock()
		if _, ok := resp.Services[svcName]; !ok {
			resp.Services[svcName] = ServiceHealth{Health: "MISSING"}
		}
		mtx.Unlock()

		checkURLs := map[arvados.URL]bool{}
		for addr := range svc.InternalURLs {
			checkURLs[addr] = true
		}
		if len(checkURLs) == 0 && svc.ExternalURL.Host != "" {
			checkURLs[svc.ExternalURL] = true
		}
		for addr := range checkURLs {
			wg.Add(1)
			go func(svcName arvados.ServiceName, addr arvados.URL) {
				defer wg.Done()
				var result CheckResult
				pingURL, err := agg.pingURL(addr)
				if err != nil {
					result = CheckResult{
						Health: "ERROR",
						Error:  err.Error(),
					}
				} else {
					result = agg.ping(pingURL)
					if result.Health != "SKIP" {
						m, err := agg.metrics(pingURL)
						if err != nil && result.Error == "" {
							result.Error = "metrics: " + err.Error()
						}
						result.Metrics = m
					}
				}

				mtx.Lock()
				defer mtx.Unlock()
				resp.Checks[fmt.Sprintf("%s+%s", svcName, pingURL)] = result
				if result.Health == "OK" || result.Health == "SKIP" {
					h := resp.Services[svcName]
					h.N++
					if result.Health == "OK" || h.N == 1 {
						// "" => "SKIP" or "OK"
						// "SKIP" => "OK"
						h.Health = result.Health
					}
					resp.Services[svcName] = h
				} else {
					resp.Health = "ERROR"
					resp.Errors = append(resp.Errors, fmt.Sprintf("%s: %s: %s", svcName, result.Health, result.Error))
				}
			}(svcName, addr)
		}
	}
	wg.Wait()

	// Report ERROR if a needed service didn't fail any checks
	// merely because it isn't configured to run anywhere.
	for svcName, sh := range resp.Services {
		switch svcName {
		case arvados.ServiceNameDispatchCloud,
			arvados.ServiceNameDispatchLSF,
			arvados.ServiceNameDispatchSLURM:
			// ok to not run any given dispatcher
		case arvados.ServiceNameHealth,
			arvados.ServiceNameWorkbench1,
			arvados.ServiceNameWorkbench2:
			// typically doesn't have InternalURLs in config
		default:
			if sh.Health != "OK" && sh.Health != "SKIP" {
				resp.Health = "ERROR"
				resp.Errors = append(resp.Errors, fmt.Sprintf("%s: %s: no InternalURLs configured", svcName, sh.Health))
				continue
			}
		}
	}

	// Check for clock skew between hosts
	var maxResponseTime time.Duration
	var clockMin, clockMax time.Time
	for _, result := range resp.Checks {
		if result.ClockTime.IsZero() {
			continue
		}
		if clockMin.IsZero() || result.ClockTime.Before(clockMin) {
			clockMin = result.ClockTime
		}
		if result.ClockTime.After(clockMax) {
			clockMax = result.ClockTime
		}
		if result.respTime > maxResponseTime {
			maxResponseTime = result.respTime
		}
	}
	skew := clockMax.Sub(clockMin)
	resp.ClockSkew = arvados.Duration(skew)
	if skew > maxClockSkew+maxResponseTime {
		msg := fmt.Sprintf("clock skew detected: maximum timestamp spread is %s (exceeds warning threshold of %s)", resp.ClockSkew, arvados.Duration(maxClockSkew))
		resp.Errors = append(resp.Errors, msg)
		resp.Health = "ERROR"
	}
	if agg.MetricClockSkew != nil {
		agg.MetricClockSkew.Set(skew.Seconds())
	}

	// Check for mismatched config files
	var newest Metrics
	for _, result := range resp.Checks {
		if result.Metrics.ConfigSourceTimestamp.After(newest.ConfigSourceTimestamp) {
			newest = result.Metrics
		}
	}
	var mismatches []string
	for target, result := range resp.Checks {
		if hash := result.Metrics.ConfigSourceSHA256; hash != "" && hash != newest.ConfigSourceSHA256 {
			mismatches = append(mismatches, target)
		}
	}
	for _, target := range mismatches {
		msg := fmt.Sprintf("outdated config: %s: config file (sha256 %s) does not match latest version with timestamp %s",
			strings.TrimSuffix(target, "/_health/ping"),
			resp.Checks[target].Metrics.ConfigSourceSHA256,
			newest.ConfigSourceTimestamp.Format(time.RFC3339))
		resp.Errors = append(resp.Errors, msg)
		resp.Health = "ERROR"
	}

	// Check for services running a different version than we are.
	for target, result := range resp.Checks {
		if result.Metrics.Version != "" && !sameVersion(result.Metrics.Version, cmd.Version.String()) {
			msg := fmt.Sprintf("version mismatch: %s is running %s -- expected %s",
				strings.TrimSuffix(target, "/_health/ping"),
				result.Metrics.Version,
				cmd.Version.String())
			resp.Errors = append(resp.Errors, msg)
			resp.Health = "ERROR"
		}
	}
	return resp
}

func (agg *Aggregator) pingURL(svcURL arvados.URL) (*url.URL, error) {
	base := url.URL(svcURL)
	return base.Parse("/_health/ping")
}

func (agg *Aggregator) ping(target *url.URL) (result CheckResult) {
	t0 := time.Now()
	defer func() {
		result.respTime = time.Since(t0)
		result.ResponseTime = json.Number(fmt.Sprintf("%.6f", result.respTime.Seconds()))
	}()
	result.Health = "ERROR"

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(agg.timeout))
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", target.String(), nil)
	if err != nil {
		result.Error = err.Error()
		return
	}
	req.Header.Set("Authorization", "Bearer "+agg.Cluster.ManagementToken)

	// Avoid workbench1's redirect-http-to-https feature
	req.Header.Set("X-Forwarded-Proto", "https")

	resp, err := agg.httpClient.Do(req)
	if urlerr, ok := err.(*url.Error); ok {
		if neterr, ok := urlerr.Err.(*net.OpError); ok && isLocalHost(target.Hostname()) {
			result = CheckResult{
				Health: "SKIP",
				Error:  neterr.Error(),
			}
			err = nil
			return
		}
	}
	if err != nil {
		result.Error = err.Error()
		return
	}
	result.HTTPStatusCode = resp.StatusCode
	err = json.NewDecoder(resp.Body).Decode(&result.Response)
	if err != nil {
		result.Error = fmt.Sprintf("cannot decode response: %s", err)
	} else if resp.StatusCode != http.StatusOK {
		result.Error = fmt.Sprintf("HTTP %d %s", resp.StatusCode, resp.Status)
	} else if h, _ := result.Response["health"].(string); h != "OK" {
		if e, ok := result.Response["error"].(string); ok && e != "" {
			result.Error = e
			return
		} else {
			result.Error = fmt.Sprintf("health=%q in ping response", h)
			return
		}
	}
	result.Health = "OK"
	result.ClockTime, _ = time.Parse(time.RFC1123, resp.Header.Get("Date"))
	return
}

var (
	reConfigMetric  = regexp.MustCompile(`arvados_config_source_timestamp_seconds{sha256="([0-9a-f]+)"} (\d[\d\.e\+]+)`)
	reVersionMetric = regexp.MustCompile(`arvados_version_running{version="([^"]+)"} 1`)
)

func (agg *Aggregator) metrics(pingURL *url.URL) (result Metrics, err error) {
	metricsURL, err := pingURL.Parse("/metrics")
	if err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(agg.timeout))
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", metricsURL.String(), nil)
	if err != nil {
		return
	}
	req.Header.Set("Authorization", "Bearer "+agg.Cluster.ManagementToken)

	// Avoid workbench1's redirect-http-to-https feature
	req.Header.Set("X-Forwarded-Proto", "https")

	resp, err := agg.httpClient.Do(req)
	if err != nil {
		return
	} else if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("%s: HTTP %d %s", metricsURL.String(), resp.StatusCode, resp.Status)
		return
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		if m := reConfigMetric.FindSubmatch(scanner.Bytes()); len(m) == 3 && len(m[1]) > 0 {
			result.ConfigSourceSHA256 = string(m[1])
			unixtime, _ := strconv.ParseFloat(string(m[2]), 64)
			result.ConfigSourceTimestamp = time.UnixMicro(int64(unixtime * 1e6))
		} else if m = reVersionMetric.FindSubmatch(scanner.Bytes()); len(m) == 2 && len(m[1]) > 0 {
			result.Version = string(m[1])
		}
	}
	if err = scanner.Err(); err != nil {
		err = fmt.Errorf("error parsing response from %s: %w", metricsURL.String(), err)
		return
	}
	return
}

// Test whether host is an easily recognizable loopback address:
// 0.0.0.0, 127.x.x.x, ::1, or localhost.
func isLocalHost(host string) bool {
	ip := net.ParseIP(host)
	return ip.IsLoopback() || bytes.Equal(ip.To4(), []byte{0, 0, 0, 0}) || strings.EqualFold(host, "localhost")
}

func (agg *Aggregator) checkAuth(req *http.Request) bool {
	creds := auth.CredentialsFromRequest(req)
	for _, token := range creds.Tokens {
		if token != "" && token == agg.Cluster.ManagementToken {
			return true
		}
	}
	return false
}

var errSilent = errors.New("")

var CheckCommand cmd.Handler = checkCommand{}

type checkCommand struct{}

func (ccmd checkCommand) RunCommand(prog string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	logger := ctxlog.New(stderr, "json", "info")
	ctx := ctxlog.Context(context.Background(), logger)
	err := ccmd.run(ctx, prog, args, stdin, stdout, stderr)
	if err != nil {
		if err != errSilent {
			fmt.Fprintln(stdout, err.Error())
		}
		return 1
	}
	return 0
}

func (ccmd checkCommand) run(ctx context.Context, prog string, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("", flag.ContinueOnError)
	flags.SetOutput(stderr)
	loader := config.NewLoader(stdin, ctxlog.New(stderr, "text", "info"))
	loader.SetupFlags(flags)
	versionFlag := flags.Bool("version", false, "Write version information to stdout and exit 0")
	timeout := flags.Duration("timeout", defaultTimeout.Duration(), "Maximum time to wait for health responses")
	outputYAML := flags.Bool("yaml", false, "Output full health report in YAML format (default mode shows errors as plain text, is silent on success)")
	if ok, _ := cmd.ParseFlags(flags, prog, args, "", stderr); !ok {
		// cmd.ParseFlags already reported the error
		return errSilent
	} else if *versionFlag {
		cmd.Version.RunCommand(prog, args, stdin, stdout, stderr)
		return nil
	}
	cfg, err := loader.Load()
	if err != nil {
		return err
	}
	cluster, err := cfg.GetCluster("")
	if err != nil {
		return err
	}
	logger := ctxlog.New(stderr, cluster.SystemLogs.Format, cluster.SystemLogs.LogLevel).WithFields(logrus.Fields{
		"ClusterID": cluster.ClusterID,
	})
	ctx = ctxlog.Context(ctx, logger)
	agg := Aggregator{Cluster: cluster, timeout: arvados.Duration(*timeout)}
	resp := agg.ClusterHealth()
	if *outputYAML {
		y, err := yaml.Marshal(resp)
		if err != nil {
			return err
		}
		stdout.Write(y)
		if resp.Health != "OK" {
			return errSilent
		}
		return nil
	}
	if resp.Health != "OK" {
		for _, msg := range resp.Errors {
			fmt.Fprintln(stdout, msg)
		}
		fmt.Fprintln(stderr, "health check failed")
		return errSilent
	}
	return nil
}

var reGoVersion = regexp.MustCompile(` \(go\d+([\d.])*\)$`)

// Return true if either a==b or the only difference is that one has a
// " (go1.2.3)" suffix and the other does not.
//
// This allows us to recognize a non-Go (rails) service as the same
// version as a Go service.
func sameVersion(a, b string) bool {
	if a == b {
		return true
	}
	anogo := reGoVersion.ReplaceAllLiteralString(a, "")
	bnogo := reGoVersion.ReplaceAllLiteralString(b, "")
	if (anogo == a) != (bnogo == b) {
		// only one of a/b has a (go1.2.3) suffix, so compare
		// without that part
		return anogo == bnogo
	}
	// both or neither has a (go1.2.3) suffix, and we already know
	// a!=b
	return false
}
