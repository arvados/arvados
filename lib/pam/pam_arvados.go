// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

// To enable, add an entry in /etc/pam.d/common-auth where pam_unix.so
// would normally be. Examples:
//
// auth [success=1 default=ignore] /usr/lib/pam_arvados.so zzzzz.arvadosapi.com vmhostname.example
// auth [success=1 default=ignore] /usr/lib/pam_arvados.so zzzzz.arvadosapi.com vmhostname.example insecure debug
//
// Replace zzzzz.arvadosapi.com with your controller host or
// host:port.
//
// Replace vmhostname.example with the VM's name as it appears in the
// Arvados virtual_machine object.
//
// Use "insecure" if your API server certificate does not pass name
// verification.
//
// Use "debug" to enable debug log messages.

package main

import (
	"io/ioutil"
	"log/syslog"
	"os"

	"context"
	"errors"
	"fmt"
	"runtime"
	"syscall"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"github.com/sirupsen/logrus"
	lSyslog "github.com/sirupsen/logrus/hooks/syslog"
	"golang.org/x/sys/unix"
)

/*
#cgo LDFLAGS: -lpam -fPIC
#include <security/pam_ext.h>
char *stringindex(char** a, int i);
const char *get_user(pam_handle_t *pamh);
const char *get_authtoken(pam_handle_t *pamh);
*/
import "C"

func main() {}

func init() {
	if err := unix.Prctl(syscall.PR_SET_DUMPABLE, 0, 0, 0, 0); err != nil {
		newLogger(false).WithError(err).Warn("unable to disable ptrace")
	}
}

//export pam_sm_setcred
func pam_sm_setcred(pamh *C.pam_handle_t, flags, cArgc C.int, cArgv **C.char) C.int {
	return C.PAM_IGNORE
}

//export pam_sm_authenticate
func pam_sm_authenticate(pamh *C.pam_handle_t, flags, cArgc C.int, cArgv **C.char) C.int {
	runtime.GOMAXPROCS(1)
	logger := newLogger(flags&C.PAM_SILENT == 0)
	cUsername := C.get_user(pamh)
	if cUsername == nil {
		return C.PAM_USER_UNKNOWN
	}

	cToken := C.get_authtoken(pamh)
	if cToken == nil {
		return C.PAM_AUTH_ERR
	}

	argv := make([]string, cArgc)
	for i := 0; i < int(cArgc); i++ {
		argv[i] = C.GoString(C.stringindex(cArgv, C.int(i)))
	}

	err := authenticate(logger, C.GoString(cUsername), C.GoString(cToken), argv)
	if err != nil {
		logger.WithError(err).Error("authentication failed")
		return C.PAM_AUTH_ERR
	}
	return C.PAM_SUCCESS
}

func authenticate(logger *logrus.Logger, username, token string, argv []string) error {
	hostname := ""
	apiHost := ""
	insecure := false
	for idx, arg := range argv {
		if idx == 0 {
			apiHost = arg
		} else if idx == 1 {
			hostname = arg
		} else if arg == "insecure" {
			insecure = true
		} else if arg == "debug" {
			logger.SetLevel(logrus.DebugLevel)
		} else {
			logger.Warnf("unkown option: %s\n", arg)
		}
	}
	if hostname == "" || hostname == "-" {
		h, err := os.Hostname()
		if err != nil {
			logger.WithError(err).Warnf("cannot get hostname -- try using an explicit hostname in pam config")
			return fmt.Errorf("cannot get hostname: %w", err)
		}
		hostname = h
	}
	logger.Debugf("username=%q arvados_api_host=%q hostname=%q insecure=%t", username, apiHost, hostname, insecure)
	if apiHost == "" {
		logger.Warnf("cannot authenticate: config error: arvados_api_host and hostname must be non-empty")
		return errors.New("config error")
	}
	arv := &arvados.Client{
		Scheme:    "https",
		APIHost:   apiHost,
		AuthToken: token,
		Insecure:  insecure,
	}
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Minute))
	defer cancel()
	var vms arvados.VirtualMachineList
	err := arv.RequestAndDecodeContext(ctx, &vms, "GET", "arvados/v1/virtual_machines", nil, arvados.ListOptions{
		Limit: 2,
		Filters: []arvados.Filter{
			{"hostname", "=", hostname},
		},
	})
	if err != nil {
		return err
	}
	if len(vms.Items) == 0 {
		// It's possible there is no VM entry for the
		// configured hostname, but typically this just means
		// the user does not have permission to see (let alone
		// log in to) this VM.
		return errors.New("permission denied")
	} else if len(vms.Items) > 1 {
		return fmt.Errorf("multiple results for hostname %q", hostname)
	} else if vms.Items[0].Hostname != hostname {
		return fmt.Errorf("looked up hostname %q but controller returned record with hostname %q", hostname, vms.Items[0].Hostname)
	}
	var user arvados.User
	err = arv.RequestAndDecodeContext(ctx, &user, "GET", "arvados/v1/users/current", nil, nil)
	if err != nil {
		return err
	}
	var links arvados.LinkList
	err = arv.RequestAndDecodeContext(ctx, &links, "GET", "arvados/v1/links", nil, arvados.ListOptions{
		Limit: 1,
		Filters: []arvados.Filter{
			{"link_class", "=", "permission"},
			{"name", "=", "can_login"},
			{"tail_uuid", "=", user.UUID},
			{"head_uuid", "=", vms.Items[0].UUID},
			{"properties.username", "=", username},
		},
	})
	if err != nil {
		return err
	}
	if len(links.Items) < 1 || links.Items[0].Properties["username"] != username {
		return errors.New("permission denied")
	}
	logger.Debugf("permission granted based on link with UUID %s", links.Items[0].UUID)
	return nil
}

func newLogger(stderr bool) *logrus.Logger {
	logger := logrus.New()
	if !stderr {
		logger.Out = ioutil.Discard
	}
	if hook, err := lSyslog.NewSyslogHook("udp", "localhost:514", syslog.LOG_AUTH|syslog.LOG_INFO, "pam_arvados"); err != nil {
		logger.Hooks.Add(hook)
	}
	return logger
}
