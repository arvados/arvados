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
	"API":                                             true,
	"API.AsyncPermissionsUpdateInterval":              true,
	"API.DisabledAPIs":                                true,
	"API.MaxIndexDatabaseRead":                        true,
	"API.MaxItemsPerResponse":                         true,
	"API.MaxRequestAmplification":                     true,
	"API.MaxRequestSize":                              true,
	"API.RailsSessionSecretToken":                     false,
	"API.RequestTimeout":                              true,
	"AuditLogs":                                       true,
	"AuditLogs.MaxAge":                                true,
	"AuditLogs.MaxDeleteBatch":                        true,
	"AuditLogs.UnloggedAttributes":                    true,
	"Collections":                                     true,
	"Collections.BlobSigning":                         true,
	"Collections.BlobSigningKey":                      false,
	"Collections.BlobSigningTTL":                      true,
	"Collections.CollectionVersioning":                true,
	"Collections.DefaultReplication":                  true,
	"Collections.DefaultTrashLifetime":                true,
	"Collections.PreserveVersionIfIdle":               true,
	"Collections.TrashSweepInterval":                  true,
	"Containers":                                      true,
	"Containers.CloudVMs":                             true,
	"Containers.CloudVMs.BootProbeCommand":            true,
	"Containers.CloudVMs.Driver":                      true,
	"Containers.CloudVMs.DriverParameters":            false,
	"Containers.CloudVMs.Enable":                      true,
	"Containers.CloudVMs.ImageID":                     true,
	"Containers.CloudVMs.MaxCloudOpsPerSecond":        true,
	"Containers.CloudVMs.MaxProbesPerSecond":          true,
	"Containers.CloudVMs.PollInterval":                true,
	"Containers.CloudVMs.ProbeInterval":               true,
	"Containers.CloudVMs.ResourceTags":                true,
	"Containers.CloudVMs.ResourceTags.*":              true,
	"Containers.CloudVMs.SSHPort":                     true,
	"Containers.CloudVMs.SyncInterval":                true,
	"Containers.CloudVMs.TagKeyPrefix":                true,
	"Containers.CloudVMs.TimeoutBooting":              true,
	"Containers.CloudVMs.TimeoutIdle":                 true,
	"Containers.CloudVMs.TimeoutProbe":                true,
	"Containers.CloudVMs.TimeoutShutdown":             true,
	"Containers.CloudVMs.TimeoutSignal":               true,
	"Containers.CloudVMs.TimeoutTERM":                 true,
	"Containers.DefaultKeepCacheRAM":                  true,
	"Containers.DispatchPrivateKey":                   true,
	"Containers.JobsAPI":                              true,
	"Containers.JobsAPI.CrunchJobUser":                true,
	"Containers.JobsAPI.CrunchJobWrapper":             true,
	"Containers.JobsAPI.CrunchRefreshTrigger":         true,
	"Containers.JobsAPI.DefaultDockerImage":           true,
	"Containers.JobsAPI.Enable":                       true,
	"Containers.JobsAPI.GitInternalDir":               true,
	"Containers.JobsAPI.ReuseJobIfOutputsDiffer":      true,
	"Containers.Logging":                              true,
	"Containers.Logging.LimitLogBytesPerJob":          true,
	"Containers.Logging.LogBytesPerEvent":             true,
	"Containers.Logging.LogPartialLineThrottlePeriod": true,
	"Containers.Logging.LogSecondsBetweenEvents":      true,
	"Containers.Logging.LogThrottleBytes":             true,
	"Containers.Logging.LogThrottleLines":             true,
	"Containers.Logging.LogThrottlePeriod":            true,
	"Containers.Logging.LogUpdatePeriod":              true,
	"Containers.Logging.LogUpdateSize":                true,
	"Containers.Logging.MaxAge":                       true,
	"Containers.LogReuseDecisions":                    true,
	"Containers.MaxComputeVMs":                        true,
	"Containers.MaxDispatchAttempts":                  true,
	"Containers.MaxRetryAttempts":                     true,
	"Containers.SLURM":                                true,
	"Containers.SLURM.Managed":                        true,
	"Containers.SLURM.Managed.AssignNodeHostname":     true,
	"Containers.SLURM.Managed.ComputeNodeDomain":      false,
	"Containers.SLURM.Managed.ComputeNodeNameservers": false,
	"Containers.SLURM.Managed.DNSServerConfDir":       true,
	"Containers.SLURM.Managed.DNSServerConfTemplate":  true,
	"Containers.SLURM.Managed.DNSServerReloadCommand": false,
	"Containers.SLURM.Managed.DNSServerUpdateCommand": false,
	"Containers.StaleLockTimeout":                     true,
	"Containers.SupportedDockerImageFormats":          true,
	"Containers.UsePreemptibleInstances":              true,
	"Git":                                             true,
	"Git.Repositories":                                true,
	"InstanceTypes":                                   true,
	"InstanceTypes.*":                                 true,
	"InstanceTypes.*.*":                               true,
	"Login":                                           true,
	"Login.ProviderAppID":                             false,
	"Login.ProviderAppSecret":                         false,
	"Mail":                                            true,
	"Mail.EmailFrom":                                  true,
	"Mail.IssueReporterEmailFrom":                     true,
	"Mail.IssueReporterEmailTo":                       true,
	"Mail.MailchimpAPIKey":                            false,
	"Mail.MailchimpListID":                            false,
	"Mail.SendUserSetupNotificationEmail":             true,
	"Mail.SupportEmailAddress":                        true,
	"ManagementToken":                                 false,
	"PostgreSQL":                                      true,
	"PostgreSQL.Connection":                           false,
	"PostgreSQL.ConnectionPool":                       true,
	"RemoteClusters":                                  true,
	"RemoteClusters.*":                                true,
	"RemoteClusters.*.ActivateUsers":                  true,
	"RemoteClusters.*.Host":                           true,
	"RemoteClusters.*.Insecure":                       true,
	"RemoteClusters.*.Proxy":                          true,
	"RemoteClusters.*.Scheme":                         true,
	"Services":                                        true,
	"Services.*":                                      true,
	"Services.*.ExternalURL":                          true,
	"Services.*.InternalURLs":                         true,
	"Services.*.InternalURLs.*":                       true,
	"Services.*.InternalURLs.*.*":                     true,
	"SystemLogs":                                      true,
	"SystemLogs.Format":                               true,
	"SystemLogs.LogLevel":                             true,
	"SystemLogs.MaxRequestLogParamsSize":              true,
	"SystemRootToken":                                 false,
	"TLS":                                             true,
	"TLS.Certificate":                                 true,
	"TLS.Insecure":                                    true,
	"TLS.Key":                                         false,
	"Users":                                           true,
	"Users.AdminNotifierEmailFrom":                    true,
	"Users.AutoAdminFirstUser":                        false,
	"Users.AutoAdminUserWithEmail":                    false,
	"Users.AutoSetupNewUsers":                         true,
	"Users.AutoSetupNewUsersWithRepository":           true,
	"Users.AutoSetupNewUsersWithVmUUID":               true,
	"Users.AutoSetupUsernameBlacklist":                false,
	"Users.EmailSubjectPrefix":                        true,
	"Users.NewInactiveUserNotificationRecipients":     false,
	"Users.NewUserNotificationRecipients":             false,
	"Users.NewUsersAreActive":                         true,
	"Users.UserNotifierEmailFrom":                     true,
	"Users.UserProfileNotificationAddress":            true,
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
