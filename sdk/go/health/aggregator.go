// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package health

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/auth"
)

const defaultTimeout = arvados.Duration(2 * time.Second)

// Aggregator implements http.Handler. It handles "GET /_health/all"
// by checking the health of all configured services on the cluster
// and responding 200 if everything is healthy.
type Aggregator struct {
	setupOnce  sync.Once
	httpClient *http.Client
	timeout    arvados.Duration

	Config *arvados.Config

	// If non-nil, Log is called after handling each request.
	Log func(*http.Request, error)
}

func (agg *Aggregator) setup() {
	agg.httpClient = http.DefaultClient
	if agg.timeout == 0 {
		// this is always the case, except in the test suite
		agg.timeout = defaultTimeout
	}
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

	cluster, err := agg.Config.GetCluster("")
	if err != nil {
		err = fmt.Errorf("arvados.GetCluster(): %s", err)
		sendErr(http.StatusInternalServerError, err)
		return
	}
	if !agg.checkAuth(req, cluster) {
		sendErr(http.StatusUnauthorized, errUnauthorized)
		return
	}
	if req.URL.Path != "/_health/all" {
		sendErr(http.StatusNotFound, errNotFound)
		return
	}
	json.NewEncoder(resp).Encode(agg.ClusterHealth(cluster))
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

func (agg *Aggregator) ClusterHealth(cluster *arvados.Cluster) ClusterHealthResponse {
	resp := ClusterHealthResponse{
		Health:   "OK",
		Checks:   make(map[string]CheckResult),
		Services: make(map[arvados.ServiceName]ServiceHealth),
	}

	mtx := sync.Mutex{}
	wg := sync.WaitGroup{}
	for profileName, profile := range cluster.NodeProfiles {
		for svc, addr := range profile.ServicePorts() {
			// Ensure svc is listed in resp.Services.
			mtx.Lock()
			if _, ok := resp.Services[svc]; !ok {
				resp.Services[svc] = ServiceHealth{Health: "ERROR"}
			}
			mtx.Unlock()

			if addr == "" {
				// svc is not expected on this node.
				continue
			}

			wg.Add(1)
			go func(profileName string, svc arvados.ServiceName, addr string) {
				defer wg.Done()
				var result CheckResult
				url, err := agg.pingURL(profileName, addr)
				if err != nil {
					result = CheckResult{
						Health: "ERROR",
						Error:  err.Error(),
					}
				} else {
					result = agg.ping(url, cluster)
				}

				mtx.Lock()
				defer mtx.Unlock()
				resp.Checks[fmt.Sprintf("%s+%s", svc, url)] = result
				if result.Health == "OK" {
					h := resp.Services[svc]
					h.N++
					h.Health = "OK"
					resp.Services[svc] = h
				} else {
					resp.Health = "ERROR"
				}
			}(profileName, svc, addr)
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

func (agg *Aggregator) pingURL(node, addr string) (string, error) {
	_, port, err := net.SplitHostPort(addr)
	return "http://" + node + ":" + port + "/_health/ping", err
}

func (agg *Aggregator) ping(url string, cluster *arvados.Cluster) (result CheckResult) {
	t0 := time.Now()

	var err error
	defer func() {
		result.ResponseTime = json.Number(fmt.Sprintf("%.6f", time.Since(t0).Seconds()))
		if err != nil {
			result.Health, result.Error = "ERROR", err.Error()
		} else {
			result.Health = "OK"
		}
	}()

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return
	}
	req.Header.Set("Authorization", "Bearer "+cluster.ManagementToken)

	ctx, cancel := context.WithTimeout(req.Context(), time.Duration(agg.timeout))
	defer cancel()
	req = req.WithContext(ctx)
	resp, err := agg.httpClient.Do(req)
	if err != nil {
		return
	}
	result.HTTPStatusCode = resp.StatusCode
	result.HTTPStatusText = resp.Status
	err = json.NewDecoder(resp.Body).Decode(&result.Response)
	if err != nil {
		err = fmt.Errorf("cannot decode response: %s", err)
	} else if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("HTTP %d %s", resp.StatusCode, resp.Status)
	} else if h, _ := result.Response["health"].(string); h != "OK" {
		if e, ok := result.Response["error"].(string); ok && e != "" {
			err = errors.New(e)
		} else {
			err = fmt.Errorf("health=%q in ping response", h)
		}
	}
	return
}

func (agg *Aggregator) checkAuth(req *http.Request, cluster *arvados.Cluster) bool {
	creds := auth.NewCredentialsFromHTTPRequest(req)
	for _, token := range creds.Tokens {
		if token != "" && token == cluster.ManagementToken {
			return true
		}
	}
	return false
}
