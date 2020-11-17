// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package keepclient

import (
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"git.arvados.org/arvados.git/sdk/go/arvadosclient"
)

// DebugPrintf emits debug messages. The easiest way to enable
// keepclient debug messages in your application is to assign
// log.Printf to DebugPrintf.
var DebugPrintf = func(string, ...interface{}) {}

func init() {
	if arvadosclient.StringBool(os.Getenv("ARVADOS_DEBUG")) {
		DebugPrintf = log.Printf
	}
}

type keepService struct {
	Uuid     string `json:"uuid"`
	Hostname string `json:"service_host"`
	Port     int    `json:"service_port"`
	SSL      bool   `json:"service_ssl_flag"`
	SvcType  string `json:"service_type"`
	ReadOnly bool   `json:"read_only"`
}

// Md5String returns md5 hash for the bytes in the given string
func Md5String(s string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(s)))
}

type svcList struct {
	Items []keepService `json:"items"`
}

type uploadStatus struct {
	err            error
	url            string
	statusCode     int
	replicasStored int
	response       string
}

func (this *KeepClient) uploadToKeepServer(host string, hash string, body io.Reader,
	uploadStatusChan chan<- uploadStatus, expectedLength int64, reqid string) {

	var req *http.Request
	var err error
	var url = fmt.Sprintf("%s/%s", host, hash)
	if req, err = http.NewRequest("PUT", url, nil); err != nil {
		DebugPrintf("DEBUG: [%s] Error creating request PUT %v error: %v", reqid, url, err.Error())
		uploadStatusChan <- uploadStatus{err, url, 0, 0, ""}
		return
	}

	req.ContentLength = expectedLength
	if expectedLength > 0 {
		req.Body = ioutil.NopCloser(body)
	} else {
		// "For client requests, a value of 0 means unknown if
		// Body is not nil."  In this case we do want the body
		// to be empty, so don't set req.Body.
	}

	req.Header.Add("X-Request-Id", reqid)
	req.Header.Add("Authorization", "OAuth2 "+this.Arvados.ApiToken)
	req.Header.Add("Content-Type", "application/octet-stream")
	req.Header.Add(XKeepDesiredReplicas, fmt.Sprint(this.Want_replicas))
	if len(this.StorageClasses) > 0 {
		req.Header.Add("X-Keep-Storage-Classes", strings.Join(this.StorageClasses, ", "))
	}

	var resp *http.Response
	if resp, err = this.httpClient().Do(req); err != nil {
		DebugPrintf("DEBUG: [%s] Upload failed %v error: %v", reqid, url, err.Error())
		uploadStatusChan <- uploadStatus{err, url, 0, 0, err.Error()}
		return
	}

	rep := 1
	if xr := resp.Header.Get(XKeepReplicasStored); xr != "" {
		fmt.Sscanf(xr, "%d", &rep)
	}

	defer resp.Body.Close()
	defer io.Copy(ioutil.Discard, resp.Body)

	respbody, err2 := ioutil.ReadAll(&io.LimitedReader{R: resp.Body, N: 4096})
	response := strings.TrimSpace(string(respbody))
	if err2 != nil && err2 != io.EOF {
		DebugPrintf("DEBUG: [%s] Upload %v error: %v response: %v", reqid, url, err2.Error(), response)
		uploadStatusChan <- uploadStatus{err2, url, resp.StatusCode, rep, response}
	} else if resp.StatusCode == http.StatusOK {
		DebugPrintf("DEBUG: [%s] Upload %v success", reqid, url)
		uploadStatusChan <- uploadStatus{nil, url, resp.StatusCode, rep, response}
	} else {
		if resp.StatusCode >= 300 && response == "" {
			response = resp.Status
		}
		DebugPrintf("DEBUG: [%s] Upload %v error: %v response: %v", reqid, url, resp.StatusCode, response)
		uploadStatusChan <- uploadStatus{errors.New(resp.Status), url, resp.StatusCode, rep, response}
	}
}

func (this *KeepClient) putReplicas(
	hash string,
	getReader func() io.Reader,
	expectedLength int64) (locator string, replicas int, err error) {

	reqid := this.getRequestID()

	// Calculate the ordering for uploading to servers
	sv := NewRootSorter(this.WritableLocalRoots(), hash).GetSortedRoots()

	// The next server to try contacting
	nextServer := 0

	// The number of active writers
	active := 0

	// Used to communicate status from the upload goroutines
	uploadStatusChan := make(chan uploadStatus)
	defer func() {
		// Wait for any abandoned uploads (e.g., we started
		// two uploads and the first replied with replicas=2)
		// to finish before closing the status channel.
		go func() {
			for active > 0 {
				<-uploadStatusChan
			}
			close(uploadStatusChan)
		}()
	}()

	replicasDone := 0
	replicasTodo := this.Want_replicas

	replicasPerThread := this.replicasPerService
	if replicasPerThread < 1 {
		// unlimited or unknown
		replicasPerThread = replicasTodo
	}

	retriesRemaining := 1 + this.Retries
	var retryServers []string

	lastError := make(map[string]string)

	for retriesRemaining > 0 {
		retriesRemaining--
		nextServer = 0
		retryServers = []string{}
		for replicasTodo > 0 {
			for active*replicasPerThread < replicasTodo {
				// Start some upload requests
				if nextServer < len(sv) {
					DebugPrintf("DEBUG: [%s] Begin upload %s to %s", reqid, hash, sv[nextServer])
					go this.uploadToKeepServer(sv[nextServer], hash, getReader(), uploadStatusChan, expectedLength, reqid)
					nextServer++
					active++
				} else {
					if active == 0 && retriesRemaining == 0 {
						msg := "Could not write sufficient replicas: "
						for _, resp := range lastError {
							msg += resp + "; "
						}
						msg = msg[:len(msg)-2]
						return locator, replicasDone, InsufficientReplicasError(errors.New(msg))
					}
					break
				}
			}
			DebugPrintf("DEBUG: [%s] Replicas remaining to write: %v active uploads: %v",
				reqid, replicasTodo, active)

			// Now wait for something to happen.
			if active > 0 {
				status := <-uploadStatusChan
				active--

				if status.statusCode == 200 {
					// good news!
					replicasDone += status.replicasStored
					replicasTodo -= status.replicasStored
					locator = status.response
					delete(lastError, status.url)
				} else {
					msg := fmt.Sprintf("[%d] %s", status.statusCode, status.response)
					if len(msg) > 100 {
						msg = msg[:100]
					}
					lastError[status.url] = msg
				}

				if status.statusCode == 0 || status.statusCode == 408 || status.statusCode == 429 ||
					(status.statusCode >= 500 && status.statusCode != 503) {
					// Timeout, too many requests, or other server side failure
					// Do not retry when status code is 503, which means the keep server is full
					retryServers = append(retryServers, status.url[0:strings.LastIndex(status.url, "/")])
				}
			} else {
				break
			}
		}

		sv = retryServers
	}

	return locator, replicasDone, nil
}
