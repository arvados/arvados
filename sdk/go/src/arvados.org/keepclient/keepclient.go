package keepclient

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
)

type KeepDisk struct {
	Hostname string `json:"service_host"`
	Port     int    `json:"service_port"`
	SSL      bool   `json:"service_ssl_flag"`
}

func KeepDisks() (service_roots []string, err error) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	var req *http.Request
	if req, err = http.NewRequest("GET", "https://localhost:3001/arvados/v1/keep_disks", nil); err != nil {
		return nil, err
	}

	var resp *http.Response
	req.Header.Add("Authorization", "OAuth2 4axaw8zxe0qm22wa6urpp5nskcne8z88cvbupv653y1njyi05h")
	if resp, err = client.Do(req); err != nil {
		return nil, err
	}

	type SvcList struct {
		Items []KeepDisk `json:"items"`
	}
	dec := json.NewDecoder(resp.Body)
	var m SvcList
	if err := dec.Decode(&m); err != nil {
		return nil, err
	}

	service_roots = make([]string, len(m.Items))
	for index, element := range m.Items {
		n := ""
		if element.SSL {
			n = "s"
		}
		service_roots[index] = fmt.Sprintf("http%s://%s:%d",
			n, element.Hostname, element.Port)
	}
	sort.Strings(service_roots)
	return service_roots, nil
}

func ShuffledServiceRoots(service_roots []string, hash string) (pseq []string) {
	// Build an ordering with which to query the Keep servers based on the
	// contents of the hash.  "hash" is a hex-encoded number at least 8
	// digits (32 bits) long

	// seed used to calculate the next keep server from 'pool' to be added
	// to 'pseq'
	seed := hash

	// Keep servers still to be added to the ordering
	pool := service_roots[:]

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
		probe, _ := strconv.ParseInt(seed[0:8], 16, 32)
		probe %= int64(len(pool))

		// Append the selected server to the probe sequence and remove it
		// from the pool.
		pseq = append(pseq, pool[probe])
		pool = append(pool[:probe], pool[probe+1:]...)

		// Remove the digits just used from the seed
		seed = seed[8:]
	}
	return pseq
}
