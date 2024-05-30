// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package crunchrun

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	. "gopkg.in/check.v1"
	check "gopkg.in/check.v1"
)

// newTestTimestamper wraps an io.Writer, inserting a predictable
// RFC3339NanoFixed timestamp at the beginning of each line.
func newTestTimestamper(w io.Writer) *prefixer {
	count := 0
	return &prefixer{
		writer: w,
		prefixFunc: func() string {
			count++
			return fmt.Sprintf("2015-12-29T15:51:45.%09dZ ", count)
		},
	}
}

type LoggingTestSuite struct {
	client *arvados.Client
}

// Gocheck boilerplate
var _ = Suite(&LoggingTestSuite{})

func (s *LoggingTestSuite) SetUpTest(c *C) {
	s.client = arvados.NewClientFromEnv()
	crunchLogUpdatePeriod = time.Hour * 24 * 365
	crunchLogUpdateSize = 1 << 50
}

func (s *LoggingTestSuite) TestWriteLogs(c *C) {
	api := &ArvTestClient{}
	kc := &KeepTestClient{}
	defer kc.Close()
	cr, err := NewContainerRunner(s.client, api, kc, "zzzzz-dz642-zzzzzzzzzzzzzzz")
	c.Assert(err, IsNil)
	f, err := cr.openLogFile("crunch-run")
	c.Assert(err, IsNil)
	cr.CrunchLog = newLogWriter(newTestTimestamper(f))

	cr.CrunchLog.Print("Hello world!")
	cr.CrunchLog.Print("Goodbye")

	c.Check(api.Calls, Equals, 0)

	logs := logFileContent(c, cr, "crunch-run.txt")
	c.Check(logs, Matches, `....-..-..T..:..:..\..........Z Hello world!\n`+
		`....-..-..T..:..:..\..........Z Goodbye\n`)
}

func (s *LoggingTestSuite) TestWriteLogsLarge(c *C) {
	if testing.Short() {
		return
	}
	api := &ArvTestClient{}
	kc := &KeepTestClient{}
	defer kc.Close()
	cr, err := NewContainerRunner(s.client, api, kc, "zzzzz-zzzzzzzzzzzzzzz")
	c.Assert(err, IsNil)
	f, err := cr.openLogFile("crunch-run")
	c.Assert(err, IsNil)
	cr.CrunchLog = newLogWriter(newTestTimestamper(f))
	for i := 0; i < 2000000; i++ {
		cr.CrunchLog.Printf("Hello %d", i)
	}
	cr.CrunchLog.Print("Goodbye")

	logs := logFileContent(c, cr, "crunch-run.txt")
	c.Check(strings.Count(logs, "\n"), Equals, 2000001)
	// Redact most of the logs except the start/end for the regexp
	// match -- otherwise, when the regexp fails, gocheck spams
	// the test logs with tens of megabytes of quoted strings.
	c.Assert(len(logs) > 10000, Equals, true)
	c.Check(logs[:500]+"\n...\n"+logs[len(logs)-500:], Matches, `(?ms)2015-12-29T15:51:45.000000001Z Hello 0
2015-12-29T15:51:45.000000002Z Hello 1
2015-12-29T15:51:45.000000003Z Hello 2
2015-12-29T15:51:45.000000004Z Hello 3
.*
2015-12-29T15:51:45.001999998Z Hello 1999997
2015-12-29T15:51:45.001999999Z Hello 1999998
2015-12-29T15:51:45.002000000Z Hello 1999999
2015-12-29T15:51:45.002000001Z Goodbye
`)

	mt, err := cr.LogCollection.MarshalManifest(".")
	c.Check(err, IsNil)
	c.Check(mt, Equals, ". 9c2c05d1fae6aaa8af85113ba725716d+67108864 80b821383a07266c2a66a4566835e26e+21780065 0:88888929:crunch-run.txt\n")
}

func (s *LoggingTestSuite) TestLogUpdate(c *C) {
	for _, trial := range []struct {
		maxBytes    int64
		maxDuration time.Duration
	}{
		{1000, 10 * time.Second},
		{1000000, time.Millisecond},
	} {
		c.Logf("max %d bytes, %s", trial.maxBytes, trial.maxDuration)
		crunchLogUpdateSize = trial.maxBytes
		crunchLogUpdatePeriod = trial.maxDuration

		api := &ArvTestClient{}
		kc := &KeepTestClient{}
		defer kc.Close()
		cr, err := NewContainerRunner(s.client, api, kc, "zzzzz-dz642-zzzzzzzzzzzzzzz")
		c.Assert(err, IsNil)
		f, err := cr.openLogFile("crunch-run")
		c.Assert(err, IsNil)
		cr.CrunchLog = newLogWriter(newTestTimestamper(f))
		stdout, err := cr.openLogFile("stdout")
		c.Assert(err, IsNil)

		c.Check(cr.logUUID, Equals, "")
		cr.CrunchLog.Printf("Hello %1000s", "space")
		for i, t := 0, time.NewTicker(time.Millisecond); i < 5000 && cr.logUUID == ""; i++ {
			<-t.C
		}
		c.Check(cr.logUUID, Not(Equals), "")
		cr.CrunchLog.Print("Goodbye")
		fmt.Fprintln(stdout, "Goodbye")

		c.Check(logFileContent(c, cr, "crunch-run.txt"), Matches, `....-..-..T..:..:............Z Hello  {995}space\n`+
			`....-..-..T..:..:............Z Goodbye\n`)
		c.Check(logFileContent(c, cr, "stdout.txt"), Matches, `Goodbye\n`)

		mt, err := cr.LogCollection.MarshalManifest(".")
		c.Check(err, IsNil)
		c.Check(mt, Matches, `. 4dc76e0a212bfa30c39d76d8c16da0c0\+1038 5be52044a8c51e7b62dd62be07872968\+47 0:1077:crunch-run.txt 1077:8:stdout.txt\n`)
	}
}

type filterSuite struct{}

var _ = Suite(&filterSuite{})

func (*filterSuite) TestFilterKeepstoreErrorsOnly(c *check.C) {
	var buf bytes.Buffer
	f := filterKeepstoreErrorsOnly{WriteCloser: nopCloser{&buf}}
	for _, s := range []string{
		"not j",
		"son\n" + `{"msg":"foo"}` + "\n{}\n" + `{"msg":"request"}` + "\n" + `{"msg":1234}` + "\n\n",
		"\n[\n",
		`{"msg":"response","respStatusCode":404,"foo": "bar"}` + "\n",
		`{"msg":"response","respStatusCode":206}` + "\n",
	} {
		f.Write([]byte(s))
	}
	c.Check(buf.String(), check.Equals, `not json
{"msg":"foo"}
{}
{"msg":1234}
[
{"msg":"response","respStatusCode":404,"foo": "bar"}
`)
}

type nopCloser struct {
	io.Writer
}

func (nopCloser) Close() error { return nil }
