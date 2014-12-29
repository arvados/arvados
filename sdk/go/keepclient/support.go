/* Internal methods to support keepclient.go */
package keepclient

import (
	"crypto/md5"
	"errors"
	"fmt"
	"git.curoverse.com/arvados.git/sdk/go/streamer"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type keepDisk struct {
	Uuid     string `json:"uuid"`
	Hostname string `json:"service_host"`
	Port     int    `json:"service_port"`
	SSL      bool   `json:"service_ssl_flag"`
	SvcType  string `json:"service_type"`
}

func Md5String(s string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(s)))
}

func (this *KeepClient) DiscoverKeepServers() error {
	if prx := os.Getenv("ARVADOS_KEEP_PROXY"); prx != "" {
		sr := map[string]string{"proxy": prx}
		this.SetServiceRoots(sr)
		this.Using_proxy = true
		if this.Client.Timeout == 0 {
			this.Client.Timeout = 10 * time.Minute
		}
		return nil
	}

	if this.Client.Timeout == 0 {
		this.Client.Timeout = 15 * time.Second
	}

	type svcList struct {
		Items []keepDisk `json:"items"`
	}
	var m svcList

	err := this.Arvados.Call("GET", "keep_services", "", "accessible", nil, &m)

	if err != nil {
		if err := this.Arvados.List("keep_disks", nil, &m); err != nil {
			return err
		}
	}

	listed := make(map[string]bool)
	service_roots := make(map[string]string)

	for _, element := range m.Items {
		n := ""

		if element.SSL {
			n = "s"
		}

		// Construct server URL
		url := fmt.Sprintf("http%s://%s:%d", n, element.Hostname, element.Port)

		// Skip duplicates
		if !listed[url] {
			listed[url] = true
			service_roots[element.Uuid] = url
		}
		if element.SvcType == "proxy" {
			this.Using_proxy = true
		}
	}

	this.SetServiceRoots(service_roots)

	return nil
}

type uploadStatus struct {
	err             error
	url             string
	statusCode      int
	replicas_stored int
	response        string
}

func (this KeepClient) uploadToKeepServer(host string, hash string, body io.ReadCloser,
	upload_status chan<- uploadStatus, expectedLength int64, requestId string) {

	var req *http.Request
	var err error
	var url = fmt.Sprintf("%s/%s", host, hash)
	if req, err = http.NewRequest("PUT", url, nil); err != nil {
		log.Printf("[%v] Error creating request PUT %v error: %v", requestId, url, err.Error())
		upload_status <- uploadStatus{err, url, 0, 0, ""}
		body.Close()
		return
	}

	if expectedLength > -1 {
		req.ContentLength = expectedLength
	}
	if expectedLength == 0 {
		defer body.Close()
	}

	req.Header.Add("Authorization", fmt.Sprintf("OAuth2 %s", this.Arvados.ApiToken))
	req.Header.Add("Content-Type", "application/octet-stream")

	if this.Using_proxy {
		req.Header.Add(X_Keep_Desired_Replicas, fmt.Sprint(this.Want_replicas))
	}

	req.Body = body

	var resp *http.Response
	if resp, err = this.Client.Do(req); err != nil {
		log.Printf("[%v] Upload failed %v error: %v", requestId, url, err.Error())
		upload_status <- uploadStatus{err, url, 0, 0, ""}
		return
	}

	rep := 1
	if xr := resp.Header.Get(X_Keep_Replicas_Stored); xr != "" {
		fmt.Sscanf(xr, "%d", &rep)
	}

	defer resp.Body.Close()
	defer io.Copy(ioutil.Discard, resp.Body)

	respbody, err2 := ioutil.ReadAll(&io.LimitedReader{resp.Body, 4096})
	response := strings.TrimSpace(string(respbody))
	if err2 != nil && err2 != io.EOF {
		log.Printf("[%v] Upload %v error: %v response: %v", requestId, url, err2.Error(), response)
		upload_status <- uploadStatus{err2, url, resp.StatusCode, rep, response}
	} else if resp.StatusCode == http.StatusOK {
		log.Printf("[%v] Upload %v success", requestId, url)
		upload_status <- uploadStatus{nil, url, resp.StatusCode, rep, response}
	} else {
		log.Printf("[%v] Upload %v error: %v response: %v", requestId, url, resp.StatusCode, response)
		upload_status <- uploadStatus{errors.New(resp.Status), url, resp.StatusCode, rep, response}
	}
}

func (this KeepClient) putReplicas(
	hash string,
	tr *streamer.AsyncStream,
	expectedLength int64) (locator string, replicas int, err error) {

	// Take the hash of locator and timestamp in order to identify this
	// specific transaction in log statements.
	requestId := fmt.Sprintf("%x", md5.Sum([]byte(locator+time.Now().String())))[0:8]

	// Calculate the ordering for uploading to servers
	sv := NewRootSorter(this.ServiceRoots(), hash).GetSortedRoots()

	// The next server to try contacting
	next_server := 0

	// The number of active writers
	active := 0

	// Used to communicate status from the upload goroutines
	upload_status := make(chan uploadStatus)
	defer close(upload_status)

	// Desired number of replicas
	remaining_replicas := this.Want_replicas

	for remaining_replicas > 0 {
		for active < remaining_replicas {
			// Start some upload requests
			if next_server < len(sv) {
				log.Printf("[%v] Begin upload %s to %s", requestId, hash, sv[next_server])
				go this.uploadToKeepServer(sv[next_server], hash, tr.MakeStreamReader(), upload_status, expectedLength, requestId)
				next_server += 1
				active += 1
			} else {
				if active == 0 {
					return locator, (this.Want_replicas - remaining_replicas), InsufficientReplicasError
				} else {
					break
				}
			}
		}
		log.Printf("[%v] Replicas remaining to write: %v active uploads: %v",
			requestId, remaining_replicas, active)

		// Now wait for something to happen.
		status := <-upload_status
		active -= 1

		if status.statusCode == 200 {
			// good news!
			remaining_replicas -= status.replicas_stored
			locator = status.response
		}
	}

	return locator, this.Want_replicas, nil
}
