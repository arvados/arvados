package keepclient

import (
	"crypto/tls"
	"encoding/json"
	"net/http"
)

type KeepDisk struct {
	Hostname string `json:"service_host"`
	Port     int    `json:"service_port"`
}

func KeepDisks() (disks []KeepDisk, err error) {
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

	return m.Items, nil
}
