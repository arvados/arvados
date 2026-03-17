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
	"net/url"
	"strings"

	"git.arvados.org/arvados.git/lib/cmd"
	"git.arvados.org/arvados.git/lib/config"
	"git.arvados.org/arvados.git/lib/dispatchcloud/worker"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
)

var InstanceCommand = cmd.Multi(map[string]cmd.Handler{
	"list":  instanceList{},
	"kill":  instanceAction{action: "kill", reason: true},
	"drain": instanceAction{action: "drain"},
	"hold":  instanceAction{action: "hold"},
	"run":   instanceAction{action: "run"},
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
		fmt.Fprintln(stderr, err)
		return 1
	}
	cluster, err := cfg.GetCluster("")
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	client := http.DefaultClient
	if len(cluster.Services.DispatchCloud.InternalURLs) == 0 {
		fmt.Fprintf(stderr, "no Services.DispatchCloud.InternalURLs configured\n")
		return 1
	}
	if *header {
		fmt.Fprint(stdout, "instance\taddress\tstate\tidle-behavior\tconfig-type\tprovider-type\tprice\trunning-containers\n")
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
			if inst.Address == "" {
				inst.Address = "-"
			}
			running := "-"
			if len(inst.RunningContainerUUIDs) > 0 {
				running = strings.Join(inst.RunningContainerUUIDs, ",")
			}
			fmt.Fprintf(stdout, "%s\t%s\t%s\t%s\t%s\t%s\t%f\t%s\n", inst.Instance, inst.Address, inst.WorkerState, inst.IdleBehavior, inst.ArvadosInstanceType, inst.ProviderInstanceType, inst.Price, running)
		}
	}
	return 0
}

type instanceAction struct {
	action string
	reason bool // accept "reason" flag
}

func (ia instanceAction) RunCommand(prog string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	logger := ctxlog.New(stderr, "text", "info")
	loader := config.NewLoader(stdin, logger)
	loader.SkipLegacy = true
	flags := flag.NewFlagSet(prog, flag.ContinueOnError)
	loader.SetupFlags(flags)
	reason := new(string)
	if ia.reason {
		reason = flags.String("reason", "", "reason to write in dispatch system logs")
	}
	if ok, code := cmd.ParseFlags(flags, prog, args, "instance-id [...]", stderr); !ok {
		return code
	}
	if len(flags.Args()) == 0 {
		fmt.Fprintln(stderr, "usage error: no instance IDs provided")
		return 2
	}
	todo := map[string]bool{}
	for _, id := range flags.Args() {
		todo[id] = true
	}
	cfg, err := loader.Load()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	cluster, err := cfg.GetCluster("")
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	client := http.DefaultClient
	if len(cluster.Services.DispatchCloud.InternalURLs) == 0 {
		fmt.Fprintf(stderr, "no Services.DispatchCloud.InternalURLs configured")
		return 1
	}
	for u := range cluster.Services.DispatchCloud.InternalURLs {
		u.Path = "/arvados/v1/dispatch/instances/" + ia.action
		for id := range todo {
			u.RawQuery = url.Values{
				"instance_id": []string{id},
				"reason":      []string{*reason},
			}.Encode()
			req, err := http.NewRequest(http.MethodPost, u.String(), nil)
			if err != nil {
				fmt.Fprintf(stderr, "%s: error setting up API request: %s\n", id, err)
				continue
			}
			req.Header.Set("Authorization", "Bearer "+cluster.ManagementToken)
			resp, err := client.Do(req)
			if err != nil {
				fmt.Fprintf(stderr, "%s: error doing API request: %s\n", id, err)
				continue
			}
			fmt.Fprintf(stderr, "%s: %s (%s)\n", id, resp.Status, u.Host)
			if resp.StatusCode == http.StatusOK {
				delete(todo, id)
			}
		}
	}
	if len(todo) > 0 {
		return 1
	}
	return 0
}
