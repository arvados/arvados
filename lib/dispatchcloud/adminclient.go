// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package dispatchcloud

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"

	"git.arvados.org/arvados.git/lib/cmd"
	"git.arvados.org/arvados.git/lib/config"
	"git.arvados.org/arvados.git/lib/dispatchcloud/worker"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
)

var InstanceCommand = cmd.Multi(map[string]cmd.Handler{
	"list": instanceList{},
})

type instanceList struct{}

func (instanceList) RunCommand(prog string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	logger := ctxlog.New(stderr, "text", "info")
	loader := config.NewLoader(stdin, logger)
	loader.SkipLegacy = true
	flags := flag.NewFlagSet(prog, flag.ContinueOnError)
	loader.SetupFlags(flags)
	header := flags.Bool("header", false, "print column headings")
	if ok, code := cmd.ParseFlags(flags, prog, args, "", stderr); !ok {
		return code
	}
	cfg, err := loader.Load()
	if err != nil {
		return 1
	}
	cluster, err := cfg.GetCluster("")
	if err != nil {
		return 1
	}
	client := http.DefaultClient
	if len(cluster.Services.DispatchCloud.InternalURLs) == 0 {
		fmt.Fprintf(stderr, "no Services.DispatchCloud.InternalURLs configured")
		return 1
	}
	if *header {
		fmt.Fprint(stdout, "instance\taddress\tstate\tidle-behavior\tconfig-type\tprovider-type\tprice\tlast-container\n")
	}
	for url := range cluster.Services.DispatchCloud.InternalURLs {
		req, err := http.NewRequest(http.MethodGet, url.String()+"/arvados/v1/dispatch/instances", nil)
		if err != nil {
			fmt.Fprintf(stderr, "error setting up API request: %s\n", err)
			return 1
		}
		req.Header.Set("Authorization", "Bearer "+cluster.ManagementToken)
		resp, err := client.Do(req)
		if err != nil {
			fmt.Fprintf(stderr, "error doing API request: %s\n", err)
			return 1
		}
		var instances struct {
			Items []worker.InstanceView
		}
		err = json.NewDecoder(resp.Body).Decode(&instances)
		if err != nil {
			fmt.Fprintf(stderr, "error decoding API response: %s\n", err)
			return 1
		}
		for _, inst := range instances.Items {
			if inst.Instance == "" {
				inst.Instance = "-"
			}
			if inst.LastContainerUUID == "" {
				inst.LastContainerUUID = "-"
			}
			fmt.Fprintf(stdout, "%s\t%s\t%s\t%s\t%s\t%s\t%f\t%s\n", inst.Instance, inst.Address, inst.WorkerState, inst.IdleBehavior, inst.ArvadosInstanceType, inst.ProviderInstanceType, inst.Price, inst.LastContainerUUID)
		}
	}
	return 0
}
