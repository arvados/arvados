package health

import (
	"context"
	"encoding/json"
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

	if agg.Config == nil {
		cfg, err := arvados.GetConfig()
		if err != nil {
			err = fmt.Errorf("arvados.GetConfig(): %s", err)
			sendErr(http.StatusInternalServerError, err)
			return
		}
		agg.Config = cfg
	}
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

type ServiceHealth struct {
	Health string `json:"health"`
	N      int    `json:"n"`
}

type ClusterHealthResponse struct {
	Health   string                   `json:"health"`
	Checks   map[string]CheckResponse `json:"checks"`
	Services map[string]ServiceHealth `json:"services"`
}

type CheckResponse struct {
	Status       int         `json:"status"`
	Health       string      `json:"health"`
	Error        string      `json:"error,omitempty"`
	ResponseTime json.Number `json:"responseTime"`
}

func (r *CheckResponse) OK() bool {
	return r.Health == "OK" && r.Status == http.StatusOK
}

func (agg *Aggregator) ClusterHealth(cluster *arvados.Cluster) ClusterHealthResponse {
	resp := ClusterHealthResponse{
		Health:   "OK",
		Checks:   make(map[string]CheckResponse),
		Services: make(map[string]ServiceHealth),
	}

	mtx := sync.Mutex{}
	wg := sync.WaitGroup{}
	for node, nodeConfig := range cluster.SystemNodes {
		for svc, addr := range map[string]string{
			"keepstore": nodeConfig.Keepstore.Listen,
		} {
			if addr == "" {
				continue
			}
			wg.Add(1)
			go func(node string) {
				defer wg.Done()
				pingResp := agg.ping(node, addr, cluster)

				mtx.Lock()
				defer mtx.Unlock()
				resp.Checks[node+"/"+svc+"/_health/ping"] = pingResp
				svHealth := resp.Services[svc]
				if pingResp.OK() {
					svHealth.N++
				} else {
					resp.Health = "ERROR"
				}
				resp.Services[svc] = svHealth
			}(node)
		}
	}
	wg.Wait()

	for svc, svHealth := range resp.Services {
		if svHealth.N > 0 {
			svHealth.Health = "OK"
		} else {
			svHealth.Health = "ERROR"
		}
		resp.Services[svc] = svHealth
	}

	return resp
}

func (agg *Aggregator) ping(node, addr string, cluster *arvados.Cluster) (result CheckResponse) {
	t0 := time.Now()

	var err error
	defer func() {
		result.ResponseTime = json.Number(fmt.Sprintf("%.6f", time.Since(t0).Seconds()))
		if err != nil {
			result.Health, result.Error = "ERROR", err.Error()
		}
	}()

	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return
	}
	req, err := http.NewRequest("GET", "http://"+node+":"+port+"/_health/ping", nil)
	if err != nil {
		return
	}
	req.Header.Set("Authorization", "Bearer "+cluster.ManagementToken)

	ctx, cancel := context.WithCancel(req.Context())
	go func() {
		select {
		case <-time.After(time.Duration(agg.timeout)):
			cancel()
		case <-ctx.Done():
		}
	}()
	req = req.WithContext(ctx)
	resp, err := agg.httpClient.Do(req)
	if err != nil {
		return
	}
	result.Status = resp.StatusCode
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		err = fmt.Errorf("cannot decode response: %s", err)
		return
	}
	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("HTTP %d %s", resp.StatusCode, resp.Status)
		return
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
