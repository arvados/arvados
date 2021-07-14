// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package keepclient

import (
	"bytes"
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadosclient"
	"git.arvados.org/arvados.git/sdk/go/asyncbuf"
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
	classesStored  map[string]int
	response       string
}

func (kc *KeepClient) uploadToKeepServer(host string, hash string, classesTodo []string, body io.Reader,
	uploadStatusChan chan<- uploadStatus, expectedLength int, reqid string) {

	var req *http.Request
	var err error
	var url = fmt.Sprintf("%s/%s", host, hash)
	if req, err = http.NewRequest("PUT", url, nil); err != nil {
		DebugPrintf("DEBUG: [%s] Error creating request PUT %v error: %v", reqid, url, err.Error())
		uploadStatusChan <- uploadStatus{err, url, 0, 0, nil, ""}
		return
	}

	req.ContentLength = int64(expectedLength)
	if expectedLength > 0 {
		req.Body = ioutil.NopCloser(body)
	} else {
		// "For client requests, a value of 0 means unknown if
		// Body is not nil."  In this case we do want the body
		// to be empty, so don't set req.Body.
	}

	req.Header.Add("X-Request-Id", reqid)
	req.Header.Add("Authorization", "OAuth2 "+kc.Arvados.ApiToken)
	req.Header.Add("Content-Type", "application/octet-stream")
	req.Header.Add(XKeepDesiredReplicas, fmt.Sprint(kc.Want_replicas))
	if len(classesTodo) > 0 {
		req.Header.Add(XKeepStorageClasses, strings.Join(classesTodo, ", "))
	}

	var resp *http.Response
	if resp, err = kc.httpClient().Do(req); err != nil {
		DebugPrintf("DEBUG: [%s] Upload failed %v error: %v", reqid, url, err.Error())
		uploadStatusChan <- uploadStatus{err, url, 0, 0, nil, err.Error()}
		return
	}

	rep := 1
	if xr := resp.Header.Get(XKeepReplicasStored); xr != "" {
		fmt.Sscanf(xr, "%d", &rep)
	}
	scc := resp.Header.Get(XKeepStorageClassesConfirmed)
	classesStored, err := parseStorageClassesConfirmedHeader(scc)
	if err != nil {
		DebugPrintf("DEBUG: [%s] Ignoring invalid %s header %q: %s", reqid, XKeepStorageClassesConfirmed, scc, err)
	}

	defer resp.Body.Close()
	defer io.Copy(ioutil.Discard, resp.Body)

	respbody, err2 := ioutil.ReadAll(&io.LimitedReader{R: resp.Body, N: 4096})
	response := strings.TrimSpace(string(respbody))
	if err2 != nil && err2 != io.EOF {
		DebugPrintf("DEBUG: [%s] Upload %v error: %v response: %v", reqid, url, err2.Error(), response)
		uploadStatusChan <- uploadStatus{err2, url, resp.StatusCode, rep, classesStored, response}
	} else if resp.StatusCode == http.StatusOK {
		DebugPrintf("DEBUG: [%s] Upload %v success", reqid, url)
		uploadStatusChan <- uploadStatus{nil, url, resp.StatusCode, rep, classesStored, response}
	} else {
		if resp.StatusCode >= 300 && response == "" {
			response = resp.Status
		}
		DebugPrintf("DEBUG: [%s] Upload %v error: %v response: %v", reqid, url, resp.StatusCode, response)
		uploadStatusChan <- uploadStatus{errors.New(resp.Status), url, resp.StatusCode, rep, classesStored, response}
	}
}

func (kc *KeepClient) BlockWrite(ctx context.Context, req arvados.BlockWriteOptions) (arvados.BlockWriteResponse, error) {
	var resp arvados.BlockWriteResponse
	var getReader func() io.Reader
	if req.Data == nil && req.Reader == nil {
		return resp, errors.New("invalid BlockWriteOptions: Data and Reader are both nil")
	}
	if req.DataSize < 0 {
		return resp, fmt.Errorf("invalid BlockWriteOptions: negative DataSize %d", req.DataSize)
	}
	if req.DataSize > BLOCKSIZE || len(req.Data) > BLOCKSIZE {
		return resp, ErrOversizeBlock
	}
	if req.Data != nil {
		if req.DataSize > len(req.Data) {
			return resp, errors.New("invalid BlockWriteOptions: DataSize > len(Data)")
		}
		if req.DataSize == 0 {
			req.DataSize = len(req.Data)
		}
		getReader = func() io.Reader { return bytes.NewReader(req.Data[:req.DataSize]) }
	} else {
		buf := asyncbuf.NewBuffer(make([]byte, 0, req.DataSize))
		go func() {
			_, err := io.Copy(buf, HashCheckingReader{req.Reader, md5.New(), req.Hash})
			buf.CloseWithError(err)
		}()
		getReader = buf.NewReader
	}
	if req.Hash == "" {
		m := md5.New()
		_, err := io.Copy(m, getReader())
		if err != nil {
			return resp, err
		}
		req.Hash = fmt.Sprintf("%x", m.Sum(nil))
	}
	if req.StorageClasses == nil {
		req.StorageClasses = kc.StorageClasses
	}
	if req.Replicas == 0 {
		req.Replicas = kc.Want_replicas
	}
	if req.RequestID == "" {
		req.RequestID = kc.getRequestID()
	}
	if req.Attempts == 0 {
		req.Attempts = 1 + kc.Retries
	}

	// Calculate the ordering for uploading to servers
	sv := NewRootSorter(kc.WritableLocalRoots(), req.Hash).GetSortedRoots()

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

	replicasTodo := map[string]int{}
	for _, c := range req.StorageClasses {
		replicasTodo[c] = req.Replicas
	}

	replicasPerThread := kc.replicasPerService
	if replicasPerThread < 1 {
		// unlimited or unknown
		replicasPerThread = req.Replicas
	}

	retriesRemaining := req.Attempts
	var retryServers []string

	lastError := make(map[string]string)
	trackingClasses := len(replicasTodo) > 0

	for retriesRemaining > 0 {
		retriesRemaining--
		nextServer = 0
		retryServers = []string{}
		for {
			var classesTodo []string
			var maxConcurrency int
			for sc, r := range replicasTodo {
				classesTodo = append(classesTodo, sc)
				if maxConcurrency == 0 || maxConcurrency > r {
					// Having more than r
					// writes in flight
					// would overreplicate
					// class sc.
					maxConcurrency = r
				}
			}
			if !trackingClasses {
				maxConcurrency = req.Replicas - resp.Replicas
			}
			if maxConcurrency < 1 {
				// If there are no non-zero entries in
				// replicasTodo, we're done.
				break
			}
			for active*replicasPerThread < maxConcurrency {
				// Start some upload requests
				if nextServer < len(sv) {
					DebugPrintf("DEBUG: [%s] Begin upload %s to %s", req.RequestID, req.Hash, sv[nextServer])
					go kc.uploadToKeepServer(sv[nextServer], req.Hash, classesTodo, getReader(), uploadStatusChan, req.DataSize, req.RequestID)
					nextServer++
					active++
				} else {
					if active == 0 && retriesRemaining == 0 {
						msg := "Could not write sufficient replicas: "
						for _, resp := range lastError {
							msg += resp + "; "
						}
						msg = msg[:len(msg)-2]
						return resp, InsufficientReplicasError(errors.New(msg))
					}
					break
				}
			}

			DebugPrintf("DEBUG: [%s] Replicas remaining to write: %v active uploads: %v", req.RequestID, replicasTodo, active)
			if active < 1 {
				break
			}

			// Wait for something to happen.
			status := <-uploadStatusChan
			active--

			if status.statusCode == http.StatusOK {
				delete(lastError, status.url)
				resp.Replicas += status.replicasStored
				if len(status.classesStored) == 0 {
					// Server doesn't report
					// storage classes. Give up
					// trying to track which ones
					// are satisfied; just rely on
					// total # replicas.
					trackingClasses = false
				}
				for className, replicas := range status.classesStored {
					if replicasTodo[className] > replicas {
						replicasTodo[className] -= replicas
					} else {
						delete(replicasTodo, className)
					}
				}
				resp.Locator = status.response
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
		}

		sv = retryServers
	}

	return resp, nil
}

func parseStorageClassesConfirmedHeader(hdr string) (map[string]int, error) {
	if hdr == "" {
		return nil, nil
	}
	classesStored := map[string]int{}
	for _, cr := range strings.Split(hdr, ",") {
		cr = strings.TrimSpace(cr)
		if cr == "" {
			continue
		}
		fields := strings.SplitN(cr, "=", 2)
		if len(fields) != 2 {
			return nil, fmt.Errorf("expected exactly one '=' char in entry %q", cr)
		}
		className := fields[0]
		if className == "" {
			return nil, fmt.Errorf("empty class name in entry %q", cr)
		}
		replicas, err := strconv.Atoi(fields[1])
		if err != nil || replicas < 1 {
			return nil, fmt.Errorf("invalid replica count %q", fields[1])
		}
		classesStored[className] = replicas
	}
	return classesStored, nil
}
