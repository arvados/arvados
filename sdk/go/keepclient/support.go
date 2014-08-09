/* Internal methods to support keepclient.go */
package keepclient

import (
	"git.curoverse.com/arvados.git/sdk/go/streamer"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

type keepDisk struct {
	Hostname string `json:"service_host"`
	Port     int    `json:"service_port"`
	SSL      bool   `json:"service_ssl_flag"`
	SvcType  string `json:"service_type"`
}

func (this *KeepClient) DiscoverKeepServers() error {
	if prx := os.Getenv("ARVADOS_KEEP_PROXY"); prx != "" {
		this.SetServiceRoots([]string{prx})
		this.Using_proxy = true
		return nil
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
	service_roots := make([]string, 0, len(m.Items))

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
			service_roots = append(service_roots, url)
		}
		if element.SvcType == "proxy" {
			this.Using_proxy = true
		}
	}

	this.SetServiceRoots(service_roots)

	return nil
}

func (this KeepClient) shuffledServiceRoots(hash string) (pseq []string) {
	// Build an ordering with which to query the Keep servers based on the
	// contents of the hash.  "hash" is a hex-encoded number at least 8
	// digits (32 bits) long

	// seed used to calculate the next keep server from 'pool' to be added
	// to 'pseq'
	seed := hash

	// Keep servers still to be added to the ordering
	service_roots := this.ServiceRoots()
	pool := make([]string, len(service_roots))
	copy(pool, service_roots)

	// output probe sequence
	pseq = make([]string, 0, len(service_roots))

	// iterate while there are servers left to be assigned
	for len(pool) > 0 {

		if len(seed) < 8 {
			// ran out of digits in the seed
			if len(pseq) < (len(hash) / 4) {
				// the number of servers added to the probe
				// sequence is less than the number of 4-digit
				// slices in 'hash' so refill the seed with the
				// last 4 digits.
				seed = hash[len(hash)-4:]
			}
			seed += hash
		}

		// Take the next 8 digits (32 bytes) and interpret as an integer,
		// then modulus with the size of the remaining pool to get the next
		// selected server.
		probe, _ := strconv.ParseUint(seed[0:8], 16, 32)
		probe %= uint64(len(pool))

		// Append the selected server to the probe sequence and remove it
		// from the pool.
		pseq = append(pseq, pool[probe])
		pool = append(pool[:probe], pool[probe+1:]...)

		// Remove the digits just used from the seed
		seed = seed[8:]
	}
	return pseq
}

type uploadStatus struct {
	err             error
	url             string
	statusCode      int
	replicas_stored int
	response        string
}

func (this KeepClient) uploadToKeepServer(host string, hash string, body io.ReadCloser,
	upload_status chan<- uploadStatus, expectedLength int64) {

	log.Printf("Uploading %s to %s", hash, host)

	var req *http.Request
	var err error
	var url = fmt.Sprintf("%s/%s", host, hash)
	if req, err = http.NewRequest("PUT", url, nil); err != nil {
		upload_status <- uploadStatus{err, url, 0, 0, ""}
		body.Close()
		return
	}

	if expectedLength > 0 {
		req.ContentLength = expectedLength
	}

	req.Header.Add("Authorization", fmt.Sprintf("OAuth2 %s", this.Arvados.ApiToken))
	req.Header.Add("Content-Type", "application/octet-stream")

	if this.Using_proxy {
		req.Header.Add(X_Keep_Desired_Replicas, fmt.Sprint(this.Want_replicas))
	}

	req.Body = body

	var resp *http.Response
	if resp, err = this.Client.Do(req); err != nil {
		upload_status <- uploadStatus{err, url, 0, 0, ""}
		body.Close()
		return
	}

	rep := 1
	if xr := resp.Header.Get(X_Keep_Replicas_Stored); xr != "" {
		fmt.Sscanf(xr, "%d", &rep)
	}

	defer resp.Body.Close()
	defer io.Copy(ioutil.Discard, resp.Body)

	respbody, err2 := ioutil.ReadAll(&io.LimitedReader{resp.Body, 4096})
	if err2 != nil && err2 != io.EOF {
		upload_status <- uploadStatus{err2, url, resp.StatusCode, rep, string(respbody)}
		return
	}

	locator := strings.TrimSpace(string(respbody))

	if resp.StatusCode == http.StatusOK {
		upload_status <- uploadStatus{nil, url, resp.StatusCode, rep, locator}
	} else {
		upload_status <- uploadStatus{errors.New(resp.Status), url, resp.StatusCode, rep, locator}
	}
}

func (this KeepClient) putReplicas(
	hash string,
	tr *streamer.AsyncStream,
	expectedLength int64) (locator string, replicas int, err error) {

	// Calculate the ordering for uploading to servers
	sv := this.shuffledServiceRoots(hash)

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
				go this.uploadToKeepServer(sv[next_server], hash, tr.MakeStreamReader(), upload_status, expectedLength)
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

		// Now wait for something to happen.
		status := <-upload_status
		if status.statusCode == 200 {
			// good news!
			remaining_replicas -= status.replicas_stored
			locator = status.response
		} else {
			// writing to keep server failed for some reason
			log.Printf("Keep server put to %v failed with '%v'",
				status.url, status.err)
		}
		active -= 1
		log.Printf("Upload to %v status code: %v remaining replicas: %v active: %v", status.url, status.statusCode, remaining_replicas, active)
	}

	return locator, this.Want_replicas, nil
}
