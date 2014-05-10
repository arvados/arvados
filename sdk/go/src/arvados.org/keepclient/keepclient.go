package keepclient

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
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

/*
func ProbeSequence(service_roots []string) (pseq []string) {
	pseq = make([]string, 0, len(disks))
	pool := disks[:]

}
*/
