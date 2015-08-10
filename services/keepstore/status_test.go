package main

import (
	"encoding/json"
)

func getStatusItem(keys ...string) interface{} {
	resp := IssueRequest(&RequestTester{"/status.json", "", "GET", nil})
	var s interface{}
	json.NewDecoder(resp.Body).Decode(&s)
	for _, k := range keys {
		s = s.(map[string]interface{})[k]
	}
	return s
}
