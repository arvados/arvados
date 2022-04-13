// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package health

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/auth"
)

const defaultTimeout = arvados.Duration(2 * time.Second)

// Aggregator implements http.Handler. It handles "GET /_health/all"
// by checking the health of all configured services on the cluster
// and responding 200 if everything is healthy.
type Aggregator struct {
	setupOnce  sync.Once
	httpClient *http.Client
	timeout    arvados.Duration

	Cluster *arvados.Cluster

	// If non-nil, Log is called after handling each request.
	Log func(*http.Request, error)
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
	Health string `json:"health"`

	// An entry for each known health check of each known instance
	// of each needed component: "instance of service S on node N
	// reports health-check C is OK."
	Checks map[string]CheckResult `json:"checks"`

	// An entry for each service type: "service S is OK." This
	// exposes problems that can't be expressed in Checks, like
	// "service S is needed, but isn't configured to run
	// anywhere."
	Services map[arvados.ServiceName]ServiceHealth `json:"services"`
}

type CheckResult struct {
	Health         string                 `json:"health"`
	Error          string                 `json:"error,omitempty"`
	HTTPStatusCode int                    `json:",omitempty"`
	HTTPStatusText string                 `json:",omitempty"`
	Response       map[string]interface{} `json:"response"`
	ResponseTime   json.Number            `json:"responseTime"`
}

type ServiceHealth struct {
	Health string `json:"health"`
	N      int    `json:"n"`
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
			resp.Services[svcName] = ServiceHealth{Health: "ERROR"}
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
				}

				mtx.Lock()
				defer mtx.Unlock()
				resp.Checks[fmt.Sprintf("%s+%s", svcName, pingURL)] = result
				if result.Health == "OK" {
					h := resp.Services[svcName]
					h.N++
					h.Health = "OK"
					resp.Services[svcName] = h
				} else {
					resp.Health = "ERROR"
				}
			}(svcName, addr)
		}
	}
	wg.Wait()

	// Report ERROR if a needed service didn't fail any checks
	// merely because it isn't configured to run anywhere.
	for _, sh := range resp.Services {
		if sh.Health != "OK" {
			resp.Health = "ERROR"
			break
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
		result.ResponseTime = json.Number(fmt.Sprintf("%.6f", time.Since(t0).Seconds()))
	}()
	result.Health = "ERROR"

	req, err := http.NewRequest("GET", target.String(), nil)
	if err != nil {
		result.Error = err.Error()
		return
	}
	req.Header.Set("Authorization", "Bearer "+agg.Cluster.ManagementToken)

	// Avoid workbench1's redirect-http-to-https feature
	req.Header.Set("X-Forwarded-Proto", "https")

	ctx, cancel := context.WithTimeout(req.Context(), time.Duration(agg.timeout))
	defer cancel()
	req = req.WithContext(ctx)
	resp, err := agg.httpClient.Do(req)
	if err != nil {
		result.Error = err.Error()
		return
	}
	result.HTTPStatusCode = resp.StatusCode
	result.HTTPStatusText = resp.Status
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
	return
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
