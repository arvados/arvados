// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package crunchrun

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"

	check "gopkg.in/check.v1"
)

var _ = check.Suite(&containerExecSuite{})

type containerExecSuite struct {
}

func downloadFile(filepath string, url string) (err error) {
	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// Writer the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

func (s *containerExecSuite) SetUpTest(c *check.C) {
	// Download to ./fixtures/busybox_uclibc.tar from
	// http://cache.arvados.org/busybox_uclibc.tar if it's not there.

	_, err := os.Stat("./fixtures/busybox_uclibc.tar")
	if err != nil {
		err := downloadFile("./fixtures/busybox_uclibc.tar", "http://cache.arvados.org/busybox_uclibc.tar")
		c.Check(err, check.IsNil)
	}
}

func (s *containerExecSuite) TearDownTest(c *check.C) {
}

func (s *containerExecSuite) TestContainerExecLoadImage(c *check.C) {
	ce, err := NewSingularityClient()
	c.Check(err, check.IsNil)

	c.Check(ce.CheckImageIsLoaded("InvalidImageID"), check.Equals, false)
	f, err := os.Open("./fixtures/hello.tar")
	c.Check(err, check.IsNil)
	defer f.Close()
	imageID, err := ce.LoadImage(f)
	c.Check(err, check.IsNil)
	loaded := ce.CheckImageIsLoaded(imageID)
	c.Check(loaded, check.Equals, true)
	err = ce.ImageRemove(imageID)
	c.Check(err, check.IsNil)
	loaded = ce.CheckImageIsLoaded(imageID)
	c.Check(loaded, check.Equals, false)
}

func (s *containerExecSuite) TestContainerExecRunContainer(c *check.C) {
	// tempfiles c.MkDir() NOTE...
	// This file will be a random file with known content
	file, err := ioutil.TempFile("/tmp", "arvados_test")
	c.Check(err, check.IsNil)
	defer os.Remove(file.Name())
	io.Copy(file, bytes.NewBufferString("known content from a file in the host"))
	err = file.Close()
	c.Check(err, check.IsNil)
	localKnownFile := file.Name()

	c.Check(err, check.IsNil)
	for _, trial := range []struct {
		env              []string
		cmd              []string
		mounts           []volumeMounts
		enableNetworking bool
		expectedStdout   string
		expectedStderr   string
	}{

		{
			// trying environment variable replacement
			env:            []string{"FOO=BAR"},
			cmd:            []string{"echo", "hello", "$FOO"},
			mounts:         []volumeMounts{},
			expectedStdout: "hello BAR\n",
			expectedStderr: "",
		},

		{
			env: []string{},
			cmd: []string{"cat", "/testFile.txt"},
			mounts: []volumeMounts{
				{
					hostPath:      localKnownFile,
					containerPath: "/testFile.txt",
					readonly:      true,
				},
			},

			expectedStdout: "known content from a file in the host",
			expectedStderr: "",
		},
	} {
		containerCfg := &ContainerConfig{
			Env:              trial.env,
			Cmd:              trial.cmd,
			Mounts:           trial.mounts,
			EnableNetworking: trial.enableNetworking,
		}
		f, err := os.Open("./fixtures/busybox_uclibc.tar")
		c.Check(err, check.IsNil)
		defer f.Close()

		ce, err := NewSingularityClient()
		c.Check(err, check.IsNil)

		imageID, err := ce.LoadImage(f)
		fmt.Printf("IMAGE %s", string(imageID))
		defer ce.ImageRemove(imageID)

		err = ce.CreateContainer(*containerCfg)
		c.Check(err, check.IsNil)
		stdout, err := ce.StdoutPipe()
		c.Check(err, check.IsNil)
		defer stdout.Close()

		stderr, err := ce.StderrPipe()
		c.Check(err, check.IsNil)
		defer stderr.Close()

		err = ce.StartContainer()
		c.Check(err, check.IsNil)

		buf := new(bytes.Buffer)
		buf.ReadFrom(stdout)
		c.Check(buf.String(), check.Equals, trial.expectedStdout)

		buf2 := new(bytes.Buffer)
		buf2.ReadFrom(stderr)
		c.Check(buf2.String(), check.Equals, trial.expectedStderr)

		err = ce.Wait()
		c.Check(err, check.IsNil)
	}
}
