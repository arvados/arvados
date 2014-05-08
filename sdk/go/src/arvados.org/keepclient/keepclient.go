package keepclient

import (
	//"net/http"
	"os"
)

type KeepClient struct {
	apiServer   string
	apiToken    string
	apiInsecure bool
}

func InitKeepClient() *KeepClient {
	kc := KeepClient{os.Getenv("ARVADOS_API_HOST"),
		os.Getenv("ARVADOS_API_TOKEN"),
		os.Getenv("ARVADOS_API_HOST_INSECURE") != ""}

	return &kc
}
