package arvadostest

import (
	"net/http"
)

// StatusAndBody struct with response status and body
type StatusAndBody struct {
	ResponseStatus int
	ResponseBody   string
}

// APIStub with Data map of path and StatusAndBody
// Ex:  /arvados/v1/keep_services = arvadostest.StatusAndBody{200, string(`{}`)}
type APIStub struct {
	Data map[string]StatusAndBody
}

func (stub *APIStub) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	if req.URL.Path == "/redirect-loop" {
		http.Redirect(resp, req, "/redirect-loop", http.StatusFound)
		return
	}

	pathResponse := stub.Data[req.URL.Path]
	if pathResponse.ResponseBody != "" {
		if pathResponse.ResponseStatus == -1 {
			http.Redirect(resp, req, "/redirect-loop", http.StatusFound)
		} else {
			resp.WriteHeader(pathResponse.ResponseStatus)
			resp.Write([]byte(pathResponse.ResponseBody))
		}
	} else {
		resp.WriteHeader(500)
		resp.Write([]byte(``))
	}
}

// KeepServerStub with Data map of path and StatusAndBody
// Ex:  /status.json = arvadostest.StatusAndBody{200, string(`{}`)}
type KeepServerStub struct {
	Data map[string]StatusAndBody
}

func (stub *KeepServerStub) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	if req.URL.Path == "/redirect-loop" {
		http.Redirect(resp, req, "/redirect-loop", http.StatusFound)
		return
	}

	pathResponse := stub.Data[req.URL.Path]
	if pathResponse.ResponseBody != "" {
		if pathResponse.ResponseStatus == -1 {
			http.Redirect(resp, req, "/redirect-loop", http.StatusFound)
		} else {
			resp.WriteHeader(pathResponse.ResponseStatus)
			resp.Write([]byte(pathResponse.ResponseBody))
		}
	} else {
		resp.WriteHeader(500)
		resp.Write([]byte(``))
	}
}
