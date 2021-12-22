// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvadostest

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"

	"gopkg.in/check.v1"
)

var authSettings = make(map[string]string)

// ResetEnv resets ARVADOS_* env vars to whatever they were the first
// time this func was called.
//
// Call it from your SetUpTest or SetUpSuite func if your tests modify
// env vars.
func ResetEnv() {
	if len(authSettings) == 0 {
		for _, e := range os.Environ() {
			e := strings.SplitN(e, "=", 2)
			if len(e) == 2 {
				authSettings[e[0]] = e[1]
			}
		}
	} else {
		for k, v := range authSettings {
			os.Setenv(k, v)
		}
	}
}

var pythonTestDir string

func chdirToPythonTests() {
	if pythonTestDir != "" {
		if err := os.Chdir(pythonTestDir); err != nil {
			log.Fatalf("chdir %s: %s", pythonTestDir, err)
		}
		return
	}
	for {
		if err := os.Chdir("sdk/python/tests"); err == nil {
			pythonTestDir, err = os.Getwd()
			if err != nil {
				log.Fatal(err)
			}
			return
		}
		if parent, err := os.Getwd(); err != nil || parent == "/" {
			log.Fatalf("sdk/python/tests/ not found in any ancestor")
		}
		if err := os.Chdir(".."); err != nil {
			log.Fatal(err)
		}
	}
}

func ResetDB(c *check.C) {
	hc := http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}}
	req, err := http.NewRequest("POST", "https://"+os.Getenv("ARVADOS_TEST_API_HOST")+"/database/reset", nil)
	c.Assert(err, check.IsNil)
	req.Header.Set("Authorization", "Bearer "+AdminToken)
	resp, err := hc.Do(req)
	c.Assert(err, check.IsNil)
	defer resp.Body.Close()
	c.Check(resp.StatusCode, check.Equals, http.StatusOK)
}

// StartKeep starts the given number of keep servers,
// optionally with --keep-blob-signing enabled.
// Use numKeepServers = 2 and blobSigning = false under all normal circumstances.
func StartKeep(numKeepServers int, blobSigning bool) {
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	chdirToPythonTests()

	cmdArgs := []string{"run_test_server.py", "start_keep", "--num-keep-servers", strconv.Itoa(numKeepServers)}
	if blobSigning {
		cmdArgs = append(cmdArgs, "--keep-blob-signing")
	}

	bgRun(exec.Command("python", cmdArgs...))
}

// StopKeep stops keep servers that were started with StartKeep.
// numkeepServers should be the same value that was passed to StartKeep,
// which is 2 under all normal circumstances.
func StopKeep(numKeepServers int) {
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	chdirToPythonTests()

	cmd := exec.Command("python", "run_test_server.py", "stop_keep", "--num-keep-servers", strconv.Itoa(numKeepServers))
	bgRun(cmd)
	// Without Wait, "go test" in go1.10.1 tends to hang. https://github.com/golang/go/issues/24050
	cmd.Wait()
}

// Start cmd, with stderr and stdout redirected to our own
// stderr. Return when the process exits, but do not wait for its
// stderr and stdout to close: any grandchild processes will continue
// writing to our stderr.
func bgRun(cmd *exec.Cmd) {
	cmd.Stdin = nil
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stderr
	if err := cmd.Start(); err != nil {
		log.Fatalf("%+v: %s", cmd.Args, err)
	}
	if _, err := cmd.Process.Wait(); err != nil {
		log.Fatalf("%+v: %s", cmd.Args, err)
	}
}

// CreateBadPath creates a tmp dir, appends given string and returns that path
// This will guarantee that the path being returned does not exist
func CreateBadPath() (badpath string, err error) {
	tempdir, err := ioutil.TempDir("", "bad")
	if err != nil {
		return "", fmt.Errorf("Could not create temporary directory for bad path: %v", err)
	}
	badpath = path.Join(tempdir, "bad")
	return badpath, nil
}

// DestroyBadPath deletes the tmp dir created by the previous CreateBadPath call
func DestroyBadPath(badpath string) error {
	tempdir := path.Join(badpath, "..")
	err := os.Remove(tempdir)
	if err != nil {
		return fmt.Errorf("Could not remove bad path temporary directory %v: %v", tempdir, err)
	}
	return nil
}
