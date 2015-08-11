package main

import (
	"encoding/json"
)

// We don't have isolated unit tests for /status.json yet, but we do
// check (e.g., in pull_worker_test.go) that /status.json reports
// specific statistics correctly at the appropriate times.

// getStatusItem("foo","bar","baz") retrieves /status.json, decodes
// the response body into resp, and returns resp["foo"]["bar"]["baz"].
func getStatusItem(keys ...string) interface{} {
	resp := IssueRequest(&RequestTester{"/status.json", "", "GET", nil})
	var s interface{}
	json.NewDecoder(resp.Body).Decode(&s)
	for _, k := range keys {
		s = s.(map[string]interface{})[k]
	}
	return s
}
