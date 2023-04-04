// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"git.arvados.org/arvados.git/lib/cmd"
	"git.arvados.org/arvados.git/lib/controller/rpc"
	"git.arvados.org/arvados.git/sdk/go/arvados"
)

// logsCommand displays logs from a running container.
type logsCommand struct {
	ac *arvados.Client
}

func (lc logsCommand) RunCommand(prog string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	f := flag.NewFlagSet(prog, flag.ContinueOnError)
	pollInterval := f.Duration("poll", time.Second*2, "minimum duration to wait before polling for new data")
	if ok, code := cmd.ParseFlags(f, prog, args, "container-uuid", stderr); !ok {
		return code
	} else if f.NArg() < 1 {
		fmt.Fprintf(stderr, "missing required argument: container-uuid (try -help)\n")
		return 2
	} else if f.NArg() > 1 {
		fmt.Fprintf(stderr, "encountered extra arguments after container-uuid (try -help)\n")
		return 2
	}
	target := f.Args()[0]

	lc.ac = arvados.NewClientFromEnv()
	lc.ac.Client = &http.Client{}
	if lc.ac.Insecure {
		lc.ac.Client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true}}
	}

	err := lc.tailf(target, stdout, stderr, *pollInterval)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func (lc *logsCommand) tailf(target string, stdout, stderr io.Writer, pollInterval time.Duration) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rpcconn := rpcFromEnv()
	ctrUUID, err := resolveToContainerUUID(rpcconn, target)
	if err != nil {
		return err
	}
	fmt.Fprintln(stderr, "connecting to container", ctrUUID)

	var (
		// files to display
		watching = []string{"crunch-run.txt", "stderr.txt"}
		// fnm => file offset of next byte to display
		mark = map[string]int64{}
		// fnm => current size of file reported by api
		size = map[string]int64{}
		// exit after fetching next log chunk
		containerFinished = false
	)

poll:
	for delay := pollInterval; ; time.Sleep(delay) {
		// When /arvados/v1/containers/{uuid}/log_events is
		// implemented, we'll wait here for the next
		// server-sent event to tell us some updated file
		// sizes. For now, we poll.
		for _, fnm := range watching {
			currentsize, _, err := lc.copyRange(ctx, ctrUUID, fnm, "0-0", nil)
			if err != nil {
				fmt.Fprintln(stderr, err)
				delay = pollInterval
				continue poll
			}
			size[fnm] = currentsize
			if oldsize, seen := mark[fnm]; !seen && currentsize > 10000 {
				mark[fnm] = currentsize - 10000
			} else if !seen {
				mark[fnm] = 0
			} else if currentsize < oldsize {
				// Log collection must have been
				// emptied and reset.
				fmt.Fprintln(stderr, "--- log restarted ---")
				for fnm := range mark {
					delete(mark, fnm)
				}
				delay = pollInterval
				continue poll
			}
		}
		newData := map[string]*bytes.Buffer{}
		for _, fnm := range watching {
			if size[fnm] > mark[fnm] {
				newData[fnm] = &bytes.Buffer{}
				_, n, err := lc.copyRange(ctx, ctrUUID, fnm, fmt.Sprintf("%d-", mark[fnm]), newData[fnm])
				if err != nil {
					fmt.Fprintln(stderr, err)
				}
				mark[fnm] += n
			}
		}
		checkState := lc.display(stdout, stderr, watching, newData)
		if containerFinished {
			// If the caller specified a container request
			// UUID and the container we were watching has
			// been replaced by a new one, start watching
			// logs from the new one. Otherwise, we're
			// done.
			if target == ctrUUID {
				// caller specified container UUID
				return nil
			}
			newUUID, err := resolveToContainerUUID(rpcconn, target)
			if err != nil {
				return err
			}
			if newUUID == ctrUUID {
				// no further attempts
				return nil
			}
			ctrUUID = newUUID
			containerFinished = false
			delay = 0
			continue
		}
		if len(newData) > 0 {
			delay = pollInterval
		}
		if len(newData) == 0 || checkState {
			delay = delay * 2
			if delay > pollInterval*5 {
				delay = pollInterval * 5
			}
			ctr, err := rpcconn.ContainerGet(ctx, arvados.GetOptions{UUID: ctrUUID, Select: []string{"state"}})
			if err != nil {
				fmt.Fprintln(stderr, err)
				delay = pollInterval
				continue
			}
			if ctr.State == arvados.ContainerStateCancelled || ctr.State == arvados.ContainerStateComplete {
				containerFinished = true
				delay = 0
			}
		}
	}
	return nil
}

// Retrieve specified byte range (e.g., "12-34", "1234-") from given
// fnm and write to out.
//
// If range is empty ("0-0"), out can be nil.
//
// Return values are current file size, bytes copied, error.
//
// If the file does not exist, return values are 0, 0, nil.
func (lc *logsCommand) copyRange(ctx context.Context, uuid, fnm, byterange string, out io.Writer) (int64, int64, error) {
	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(20*time.Second))
	defer cancel()
	srcURL := url.URL{
		Scheme: "https",
		Host:   lc.ac.APIHost,
		Path:   "/arvados/v1/containers/" + uuid + "/log/" + fnm,
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srcURL.String(), nil)
	if err != nil {
		return 0, 0, err
	}
	req.Header.Set("Range", "bytes="+byterange)
	req.Header.Set("Authorization", "Bearer "+lc.ac.AuthToken)
	resp, err := lc.ac.Client.Do(req)
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return 0, 0, nil
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return 0, 0, fmt.Errorf("error getting %s: %s", fnm, resp.Status)
	}
	var rstart, rend, rsize int64
	_, err = fmt.Sscanf(resp.Header.Get("Content-Range"), "bytes %d-%d/%d", &rstart, &rend, &rsize)
	if err != nil {
		return 0, 0, fmt.Errorf("error parsing Content-Range header %q: %s", resp.Header.Get("Content-Range"), err)
	}
	if out == nil {
		return rsize, 0, nil
	}
	n, err := io.Copy(out, resp.Body)
	return rsize, n, err
}

// display some log data, formatted as desired (prefixing each line
// with a tag indicating which file it came from, etc.).
//
// Return value is true if the log data contained a hint that it's a
// good time to check whether the container is finished so we can
// exit.
func (lc *logsCommand) display(out, stderr io.Writer, watching []string, received map[string]*bytes.Buffer) bool {
	checkState := false
	for _, fnm := range watching {
		buf := received[fnm]
		if buf == nil || buf.Len() == 0 {
			continue
		}
		for _, line := range bytes.Split(bytes.TrimSuffix(buf.Bytes(), []byte{'\n'}), []byte{'\n'}) {
			_, err := fmt.Fprintf(out, "%-14s %s\n", fnm, line)
			if err != nil {
				fmt.Fprintln(stderr, err)
			}
			checkState = checkState ||
				bytes.HasSuffix(line, []byte("Complete")) ||
				bytes.HasSuffix(line, []byte("Cancelled")) ||
				bytes.HasSuffix(line, []byte("Queued"))
		}
	}
	return checkState
}

// shellCommand connects the terminal to an interactive shell on a
// running container.
type shellCommand struct{}

func (shellCommand) RunCommand(prog string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	f := flag.NewFlagSet(prog, flag.ContinueOnError)
	detachKeys := f.String("detach-keys", "ctrl-],ctrl-]", "set detach key sequence, as in docker-attach(1)")
	if ok, code := cmd.ParseFlags(f, prog, args, "[username@]container-uuid [ssh-options] [remote-command [args...]]", stderr); !ok {
		return code
	} else if f.NArg() < 1 {
		fmt.Fprintf(stderr, "missing required argument: container-uuid (try -help)\n")
		return 2
	}
	target := f.Args()[0]
	if !strings.Contains(target, "@") {
		target = "root@" + target
	}
	sshargs := f.Args()[1:]

	// Try setting up a tunnel, and exit right away if it
	// fails. This tunnel won't get used -- we'll set up a new
	// tunnel when running as SSH client's ProxyCommand child --
	// but in most cases where the real tunnel setup would fail,
	// we catch the problem earlier here. This makes it less
	// likely that an error message about tunnel setup will get
	// hidden behind noisy errors from SSH client like this:
	//
	// [useful tunnel setup error message here]
	// kex_exchange_identification: Connection closed by remote host
	// Connection closed by UNKNOWN port 65535
	// exit status 255
	//
	// In case our target is a container request, the probe also
	// resolves it to a container, so we don't connect to two
	// different containers in a race.
	var probetarget bytes.Buffer
	exitcode := connectSSHCommand{}.RunCommand(
		"arvados-client connect-ssh",
		[]string{"-detach-keys=" + *detachKeys, "-probe-only=true", target},
		&bytes.Buffer{}, &probetarget, stderr)
	if exitcode != 0 {
		return exitcode
	}
	target = strings.Trim(probetarget.String(), "\n")

	selfbin, err := os.Readlink("/proc/self/exe")
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	sshargs = append([]string{
		"ssh",
		"-o", "ProxyCommand " + selfbin + " connect-ssh -detach-keys=" + shellescape(*detachKeys) + " " + shellescape(target),
		"-o", "StrictHostKeyChecking no",
		target},
		sshargs...)
	sshbin, err := exec.LookPath("ssh")
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	err = syscall.Exec(sshbin, sshargs, os.Environ())
	fmt.Fprintf(stderr, "exec(%q) failed: %s\n", sshbin, err)
	return 1
}

// connectSSHCommand connects stdin/stdout to a container's gateway
// server (see lib/crunchrun/ssh.go).
//
// It is intended to be invoked with OpenSSH client's ProxyCommand
// config.
type connectSSHCommand struct{}

func (connectSSHCommand) RunCommand(prog string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	f := flag.NewFlagSet(prog, flag.ContinueOnError)
	f.SetOutput(stderr)
	f.Usage = func() {
		_, prog := filepath.Split(prog)
		fmt.Fprint(stderr, prog+`: connect to the gateway service for a running container.

NOTE: You almost certainly don't want to use this command directly. It
is meant to be used internally. Use "arvados-client shell" instead.

Usage: `+prog+` [options] [username@]container-uuid

Options:
`)
		f.PrintDefaults()
	}
	probeOnly := f.Bool("probe-only", false, "do not transfer IO, just setup tunnel, print target UUID, and exit")
	detachKeys := f.String("detach-keys", "", "set detach key sequence, as in docker-attach(1)")
	if ok, code := cmd.ParseFlags(f, prog, args, "[username@]container-uuid", stderr); !ok {
		return code
	} else if f.NArg() != 1 {
		fmt.Fprintf(stderr, "missing required argument: [username@]container-uuid\n")
		return 2
	}
	targetUUID := f.Args()[0]
	loginUsername := "root"
	if i := strings.Index(targetUUID, "@"); i >= 0 {
		loginUsername = targetUUID[:i]
		targetUUID = targetUUID[i+1:]
	}
	if os.Getenv("ARVADOS_API_HOST") == "" || os.Getenv("ARVADOS_API_TOKEN") == "" {
		fmt.Fprintln(stderr, "fatal: ARVADOS_API_HOST and ARVADOS_API_TOKEN environment variables are not set")
		return 1
	}
	rpcconn := rpcFromEnv()
	targetUUID, err := resolveToContainerUUID(rpcconn, targetUUID)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintln(stderr, "connecting to container", targetUUID)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sshconn, err := rpcconn.ContainerSSH(ctx, arvados.ContainerSSHOptions{
		UUID:          targetUUID,
		DetachKeys:    *detachKeys,
		LoginUsername: loginUsername,
	})
	if err != nil {
		fmt.Fprintln(stderr, "error setting up tunnel:", err)
		return 1
	}
	defer sshconn.Conn.Close()

	if *probeOnly {
		fmt.Fprintln(stdout, targetUUID)
		return 0
	}

	go func() {
		defer cancel()
		_, err := io.Copy(stdout, sshconn.Conn)
		if err != nil && ctx.Err() == nil {
			fmt.Fprintf(stderr, "receive: %v\n", err)
		}
	}()
	go func() {
		defer cancel()
		_, err := io.Copy(sshconn.Conn, stdin)
		if err != nil && ctx.Err() == nil {
			fmt.Fprintf(stderr, "send: %v\n", err)
		}
	}()
	<-ctx.Done()
	return 0
}

func shellescape(s string) string {
	return "'" + strings.Replace(s, "'", "'\\''", -1) + "'"
}

func rpcFromEnv() *rpc.Conn {
	insecure := os.Getenv("ARVADOS_API_HOST_INSECURE")
	return rpc.NewConn("",
		&url.URL{
			Scheme: "https",
			Host:   os.Getenv("ARVADOS_API_HOST"),
		},
		insecure == "1" || insecure == "yes" || insecure == "true",
		func(context.Context) ([]string, error) {
			return []string{os.Getenv("ARVADOS_API_TOKEN")}, nil
		})
}

func resolveToContainerUUID(rpcconn *rpc.Conn, targetUUID string) (string, error) {
	switch {
	case strings.Contains(targetUUID, "-dz642-"):
		return targetUUID, nil
	case strings.Contains(targetUUID, "-xvhdp-"):
		crs, err := rpcconn.ContainerRequestList(context.TODO(), arvados.ListOptions{Limit: -1, Filters: []arvados.Filter{{"uuid", "=", targetUUID}}})
		if err != nil {
			return "", err
		}
		if len(crs.Items) < 1 {
			return "", fmt.Errorf("container request %q not found\n", targetUUID)
		}
		cr := crs.Items[0]
		if cr.ContainerUUID == "" {
			return "", fmt.Errorf("no container assigned, container request state is %s\n", strings.ToLower(string(cr.State)))
		}
		return cr.ContainerUUID, nil
	default:
		return "", fmt.Errorf("target UUID is not a container or container request UUID: %s\n", targetUUID)
	}
}
