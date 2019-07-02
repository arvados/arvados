// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
)

// ExportJSON writes a JSON object with the safe (non-secret) portions
// of the cluster config to w.
func ExportJSON(w io.Writer, cluster *arvados.Cluster) error {
	buf, err := json.Marshal(cluster)
	if err != nil {
		return err
	}
	var m map[string]interface{}
	err = json.Unmarshal(buf, &m)
	if err != nil {
		return err
	}
	err = redactUnsafe(m, "", "")
	if err != nil {
		return err
	}
	return json.NewEncoder(w).Encode(m)
}

// whitelist classifies configs as safe/unsafe to reveal to
// unauthenticated clients.
//
// Every config entry must either be listed explicitly here along with
// all of its parent keys (e.g., "API" + "API.RequestTimeout"), or
// have an ancestor listed as false (e.g.,
// "PostgreSQL.Connection.password" has an ancestor
// "PostgreSQL.Connection" with a false value). Otherwise, it is a bug
// which should be caught by tests.
//
// Example: API.RequestTimeout is safe because whitelist["API"] == and
// whitelist["API.RequestTimeout"] == true.
//
// Example: PostgreSQL.Connection.password is not safe because
// whitelist["PostgreSQL.Connection"] == false.
//
// Example: PostgreSQL.BadKey would cause an error because
// whitelist["PostgreSQL"] isn't false, and neither
// whitelist["PostgreSQL.BadKey"] nor whitelist["PostgreSQL.*"]
// exists.
var whitelist = map[string]bool{
	// | sort -t'"' -k2,2
	"API":                                         true,
	"API.AsyncPermissionsUpdateInterval":          false,
	"API.DisabledAPIs":                            false,
	"API.MaxIndexDatabaseRead":                    false,
	"API.MaxItemsPerResponse":                     true,
	"API.MaxRequestAmplification":                 false,
	"API.MaxRequestSize":                          true,
	"API.RailsSessionSecretToken":                 false,
	"API.RequestTimeout":                          true,
	"AuditLogs":                                   false,
	"AuditLogs.MaxAge":                            false,
	"AuditLogs.MaxDeleteBatch":                    false,
	"AuditLogs.UnloggedAttributes":                false,
	"Collections":                                 true,
	"Collections.BlobSigning":                     true,
	"Collections.BlobSigningKey":                  false,
	"Collections.BlobSigningTTL":                  true,
	"Collections.CollectionVersioning":            false,
	"Collections.DefaultReplication":              true,
	"Collections.DefaultTrashLifetime":            true,
	"Collections.ManagedProperties":               true,
	"Collections.ManagedProperties.*":             true,
	"Collections.ManagedProperties.*.*":           true,
	"Collections.PreserveVersionIfIdle":           true,
	"Collections.TrashSweepInterval":              false,
	"Containers":                                  true,
	"Containers.CloudVMs":                         false,
	"Containers.DefaultKeepCacheRAM":              true,
	"Containers.DispatchPrivateKey":               false,
	"Containers.JobsAPI":                          true,
	"Containers.JobsAPI.CrunchJobUser":            false,
	"Containers.JobsAPI.CrunchJobWrapper":         false,
	"Containers.JobsAPI.CrunchRefreshTrigger":     false,
	"Containers.JobsAPI.DefaultDockerImage":       false,
	"Containers.JobsAPI.Enable":                   true,
	"Containers.JobsAPI.GitInternalDir":           false,
	"Containers.JobsAPI.ReuseJobIfOutputsDiffer":  false,
	"Containers.Logging":                          false,
	"Containers.LogReuseDecisions":                false,
	"Containers.MaxComputeVMs":                    false,
	"Containers.MaxDispatchAttempts":              false,
	"Containers.MaxRetryAttempts":                 true,
	"Containers.SLURM":                            false,
	"Containers.StaleLockTimeout":                 false,
	"Containers.SupportedDockerImageFormats":      true,
	"Containers.UsePreemptibleInstances":          true,
	"EnableBetaController14287":                   false,
	"Git":                                         false,
	"InstanceTypes":                               true,
	"InstanceTypes.*":                             true,
	"InstanceTypes.*.*":                           true,
	"Login":                                       false,
	"Mail":                                        false,
	"ManagementToken":                             false,
	"PostgreSQL":                                  false,
	"RemoteClusters":                              true,
	"RemoteClusters.*":                            true,
	"RemoteClusters.*.ActivateUsers":              true,
	"RemoteClusters.*.Host":                       true,
	"RemoteClusters.*.Insecure":                   true,
	"RemoteClusters.*.Proxy":                      true,
	"RemoteClusters.*.Scheme":                     true,
	"Services":                                    true,
	"Services.*":                                  true,
	"Services.*.ExternalURL":                      true,
	"Services.*.InternalURLs":                     false,
	"SystemLogs":                                  false,
	"SystemRootToken":                             false,
	"TLS":                                         false,
	"Users":                                       true,
	"Users.AnonymousUserToken":                    true,
	"Users.AdminNotifierEmailFrom":                false,
	"Users.AutoAdminFirstUser":                    false,
	"Users.AutoAdminUserWithEmail":                false,
	"Users.AutoSetupNewUsers":                     false,
	"Users.AutoSetupNewUsersWithRepository":       false,
	"Users.AutoSetupNewUsersWithVmUUID":           false,
	"Users.AutoSetupUsernameBlacklist":            false,
	"Users.EmailSubjectPrefix":                    false,
	"Users.NewInactiveUserNotificationRecipients": false,
	"Users.NewUserNotificationRecipients":         false,
	"Users.NewUsersAreActive":                     false,
	"Users.UserNotifierEmailFrom":                 false,
	"Users.UserProfileNotificationAddress":        false,
	"Workbench":                                   true,
	"Workbench.ActivationContactLink":             false,
	"Workbench.APIClientConnectTimeout":           true,
	"Workbench.APIClientReceiveTimeout":           true,
	"Workbench.APIResponseCompression":            false,
	"Workbench.ApplicationMimetypesWithViewIcon":  false,
	"Workbench.ArvadosDocsite":                    true,
	"Workbench.ArvadosPublicDataDocURL":           true,
	"Workbench.EnableGettingStartedPopup":         true,
	"Workbench.EnablePublicProjectsPage":          true,
	"Workbench.FileViewersConfigURL":              true,
	"Workbench.LogViewerMaxBytes":                 true,
	"Workbench.MultiSiteSearch":                   true,
	"Workbench.Repositories":                      false,
	"Workbench.RepositoryCache":                   false,
	"Workbench.RunningJobLogRecordsToFetch":       true,
	"Workbench.SecretKeyBase":                     false,
	"Workbench.ShowRecentCollectionsOnDashboard":  true,
	"Workbench.ShowUserAgreementInline":           false,
	"Workbench.ShowUserNotifications":             false,
	"Workbench.SiteName":                          true,
	"Workbench.Theme":                             true,
	"Workbench.TrustAllContent":                   false,
	"Workbench.UserProfileFormFields":             true,
	"Workbench.UserProfileFormMessage":            true,
	"Workbench.VocabularyURL":                     true,
}

func redactUnsafe(m map[string]interface{}, mPrefix, lookupPrefix string) error {
	var errs []string
	for k, v := range m {
		lookupKey := k
		safe, ok := whitelist[lookupPrefix+k]
		if !ok {
			lookupKey = "*"
			safe, ok = whitelist[lookupPrefix+"*"]
		}
		if !ok {
			errs = append(errs, fmt.Sprintf("config bug: key %q not in whitelist map", lookupPrefix+k))
			continue
		}
		if !safe {
			delete(m, k)
			continue
		}
		if v, ok := v.(map[string]interface{}); ok {
			err := redactUnsafe(v, mPrefix+k+".", lookupPrefix+lookupKey+".")
			if err != nil {
				errs = append(errs, err.Error())
			}
		}
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n"))
	}
	return nil
}
