// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package killcollection

import (
	"flag"
	"fmt"
	"io"
	"os"

	"git.arvados.org/arvados.git/lib/config"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadosclient"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"git.arvados.org/arvados.git/sdk/go/keepclient"
	"github.com/sirupsen/logrus"
)

var Command command

type command struct{}

func (command) RunCommand(prog string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	var err error
	logger := ctxlog.New(stderr, "text", "info")
	defer func() {
		if err != nil {
			logger.WithError(err).Error("fatal")
		}
		logger.Info("exiting")
	}()

	loader := config.NewLoader(stdin, logger)

	flags := flag.NewFlagSet("", flag.ContinueOnError)
	flags.SetOutput(stderr)
	loader.SetupFlags(flags)
	projectName := flags.String("project-name", "placeholder collections with lost data", "name of project to move collections into")
	placeholderFilename := flags.String("placeholder-filename", ".contents_removed", "name of empty file in replacement collection")
	loglevel := flags.String("log-level", "info", "logging level (debug, info, ...)")
	err = flags.Parse(args)
	if err == flag.ErrHelp {
		err = nil
		return 0
	} else if err != nil {
		return 2
	}

	if len(flags.Args()) == 0 {
		fmt.Fprintf(stderr, "Usage: %s [options] uuid ...\n", prog)
		flags.PrintDefaults()
		return 2
	}

	lvl, err := logrus.ParseLevel(*loglevel)
	if err != nil {
		return 2
	}
	logger.SetLevel(lvl)

	cfg, err := loader.Load()
	if err != nil {
		return 1
	}
	cluster, err := cfg.GetCluster("")
	if err != nil {
		return 1
	}
	client, err := arvados.NewClientFromConfig(cluster)
	if err != nil {
		return 1
	}
	client.AuthToken = cluster.SystemRootToken

	arv, err := arvadosclient.New(client)
	if err != nil {
		return 1
	}
	kc, err := keepclient.MakeKeepClient(arv)
	if err != nil {
		return 1
	}
	fs, err := (&arvados.Collection{}).FileSystem(client, kc)
	if err != nil {
		return 1
	}
	f, err := fs.OpenFile(*placeholderFilename, os.O_CREATE|os.O_WRONLY, 0777)
	if err != nil {
		return 1
	}
	err = f.Close()
	if err != nil {
		return 1
	}
	manifest, err := fs.MarshalManifest(".")
	if err != nil {
		return 1
	}
	logger.WithField("manifest_text", manifest).Debug("replacement manifest")

	var systemUser arvados.User
	err = client.RequestAndDecode(&systemUser, "GET", "arvados/v1/users/current", nil, nil)
	if err != nil {
		return 1
	}
	logger.Printf("system user uuid is %s", systemUser.UUID)
	var projectList arvados.GroupList
	err = client.RequestAndDecode(&projectList, "GET", "arvados/v1/groups", nil, arvados.ListOptions{
		Limit: 1,
		Filters: []arvados.Filter{
			{"name", "=", *projectName},
			{"owner_uuid", "=", systemUser.UUID},
		},
	})
	if err != nil {
		return 1
	}
	var project arvados.Group
	if len(projectList.Items) > 0 {
		project = projectList.Items[0]
		logger.WithField("UUID", project.UUID).Info("using existing project")
	} else {
		logger.Info("creating new project")
		err = client.RequestAndDecode(&project, "POST", "arvados/v1/groups", nil, map[string]interface{}{
			"group": map[string]interface{}{
				"name":        *projectName,
				"owner_uuid":  systemUser.UUID,
				"group_class": "project",
			},
		})
		if err != nil {
			return 1
		}
		logger.WithField("UUID", project.UUID).Info("created new project")
	}

	for _, uuid := range flags.Args() {
		logger := logger.WithField("UUID", uuid)
		var coll arvados.Collection
		err := client.RequestAndDecode(&coll, "PATCH", "arvados/v1/collections/"+uuid, nil, map[string]interface{}{
			"collection": map[string]interface{}{
				"owner_uuid":    project.UUID,
				"manifest_text": manifest,
				"name":          "placeholder for collection " + uuid,
			},
		})
		if err != nil {
			logger.WithError(err).Error("error updating collection")
			return 1
		}
		logger.Info("done")
	}
	return 0
}
