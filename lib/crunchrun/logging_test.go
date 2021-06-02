// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package crunchrun

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadosclient"
	. "gopkg.in/check.v1"
)

type LoggingTestSuite struct {
	client *arvados.Client
}

type TestTimestamper struct {
	count int
}

func (stamper *TestTimestamper) Timestamp(t time.Time) string {
	stamper.count++
	t, err := time.ParseInLocation(time.RFC3339Nano, fmt.Sprintf("2015-12-29T15:51:45.%09dZ", stamper.count), t.Location())
	if err != nil {
		panic(err)
	}
	return RFC3339Timestamp(t)
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
	cr, err := NewContainerRunner(s.client, api, kc, "zzzzz-zzzzzzzzzzzzzzz")
	c.Assert(err, IsNil)
	cr.CrunchLog.Timestamper = (&TestTimestamper{}).Timestamp

	cr.CrunchLog.Print("Hello world!")
	cr.CrunchLog.Print("Goodbye")
	cr.CrunchLog.Close()

	c.Check(api.Calls, Equals, 1)

	mt, err := cr.LogCollection.MarshalManifest(".")
	c.Check(err, IsNil)
	c.Check(mt, Equals, ". 74561df9ae65ee9f35d5661d42454264+83 0:83:crunch-run.txt\n")

	logtext := "2015-12-29T15:51:45.000000001Z Hello world!\n" +
		"2015-12-29T15:51:45.000000002Z Goodbye\n"

	c.Check(api.Content[0]["log"].(arvadosclient.Dict)["event_type"], Equals, "crunch-run")
	c.Check(api.Content[0]["log"].(arvadosclient.Dict)["properties"].(map[string]string)["text"], Equals, logtext)
	c.Check(string(kc.Content), Equals, logtext)
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
	cr.CrunchLog.Timestamper = (&TestTimestamper{}).Timestamp
	cr.CrunchLog.Immediate = nil

	for i := 0; i < 2000000; i++ {
		cr.CrunchLog.Printf("Hello %d", i)
	}
	cr.CrunchLog.Print("Goodbye")
	cr.CrunchLog.Close()

	c.Check(api.Calls > 0, Equals, true)
	c.Check(api.Calls < 2000000, Equals, true)

	mt, err := cr.LogCollection.MarshalManifest(".")
	c.Check(err, IsNil)
	c.Check(mt, Equals, ". 9c2c05d1fae6aaa8af85113ba725716d+67108864 80b821383a07266c2a66a4566835e26e+21780065 0:88888929:crunch-run.txt\n")
}

func (s *LoggingTestSuite) TestWriteMultipleLogs(c *C) {
	api := &ArvTestClient{}
	kc := &KeepTestClient{}
	defer kc.Close()
	cr, err := NewContainerRunner(s.client, api, kc, "zzzzz-zzzzzzzzzzzzzzz")
	c.Assert(err, IsNil)
	ts := &TestTimestamper{}
	cr.CrunchLog.Timestamper = ts.Timestamp
	w, err := cr.NewLogWriter("stdout")
	c.Assert(err, IsNil)
	stdout := NewThrottledLogger(w)
	stdout.Timestamper = ts.Timestamp

	cr.CrunchLog.Print("Hello world!")
	stdout.Print("Doing stuff")
	cr.CrunchLog.Print("Goodbye")
	stdout.Print("Blurb")
	cr.CrunchLog.Close()
	stdout.Close()

	logText := make(map[string]string)
	for _, content := range api.Content {
		log := content["log"].(arvadosclient.Dict)
		logText[log["event_type"].(string)] += log["properties"].(map[string]string)["text"]
	}

	c.Check(logText["crunch-run"], Equals, `2015-12-29T15:51:45.000000001Z Hello world!
2015-12-29T15:51:45.000000003Z Goodbye
`)
	c.Check(logText["stdout"], Equals, `2015-12-29T15:51:45.000000002Z Doing stuff
2015-12-29T15:51:45.000000004Z Blurb
`)

	mt, err := cr.LogCollection.MarshalManifest(".")
	c.Check(err, IsNil)
	c.Check(mt, Equals, ". 48f9023dc683a850b1c9b482b14c4b97+163 0:83:crunch-run.txt 83:80:stdout.txt\n")
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
		cr, err := NewContainerRunner(s.client, api, kc, "zzzzz-zzzzzzzzzzzzzzz")
		c.Assert(err, IsNil)
		ts := &TestTimestamper{}
		cr.CrunchLog.Timestamper = ts.Timestamp
		w, err := cr.NewLogWriter("stdout")
		c.Assert(err, IsNil)
		stdout := NewThrottledLogger(w)
		stdout.Timestamper = ts.Timestamp

		c.Check(cr.logUUID, Equals, "")
		cr.CrunchLog.Printf("Hello %1000s", "space")
		for i, t := 0, time.NewTicker(time.Millisecond); i < 5000 && cr.logUUID == ""; i++ {
			<-t.C
		}
		c.Check(cr.logUUID, Not(Equals), "")
		cr.CrunchLog.Print("Goodbye")
		fmt.Fprint(stdout, "Goodbye\n")
		cr.CrunchLog.Close()
		stdout.Close()
		w.Close()

		mt, err := cr.LogCollection.MarshalManifest(".")
		c.Check(err, IsNil)
		// Block packing depends on whether there's an update
		// between the two Goodbyes -- either way the first
		// block will be 4dc76.
		c.Check(mt, Matches, `. 4dc76e0a212bfa30c39d76d8c16da0c0\+1038 (afc503bc1b9a828b4bb543cb629e936c\+78|90699dc22545cd74a0664303f70bc05a\+39 276b49339fd5203d15a93ff3de11bfb9\+39) 0:1077:crunch-run.txt 1077:39:stdout.txt\n`)
	}
}

func (s *LoggingTestSuite) TestWriteLogsWithRateLimitThrottleBytes(c *C) {
	s.testWriteLogsWithRateLimit(c, "crunchLogThrottleBytes", 50, 65536, "Exceeded rate 50 bytes per 60 seconds")
}

func (s *LoggingTestSuite) TestWriteLogsWithRateLimitThrottleLines(c *C) {
	s.testWriteLogsWithRateLimit(c, "crunchLogThrottleLines", 1, 1024, "Exceeded rate 1 lines per 60 seconds")
}

func (s *LoggingTestSuite) TestWriteLogsWithRateLimitThrottleBytesPerEvent(c *C) {
	s.testWriteLogsWithRateLimit(c, "crunchLimitLogBytesPerJob", 50, 67108864, "Exceeded log limit 50 bytes (crunch_limit_log_bytes_per_job)")
}

func (s *LoggingTestSuite) testWriteLogsWithRateLimit(c *C, throttleParam string, throttleValue int, throttleDefault int, expected string) {
	discoveryMap[throttleParam] = float64(throttleValue)
	defer func() {
		discoveryMap[throttleParam] = float64(throttleDefault)
	}()

	api := &ArvTestClient{}
	kc := &KeepTestClient{}
	defer kc.Close()
	cr, err := NewContainerRunner(s.client, api, kc, "zzzzz-zzzzzzzzzzzzzzz")
	c.Assert(err, IsNil)
	cr.CrunchLog.Timestamper = (&TestTimestamper{}).Timestamp

	cr.CrunchLog.Print("Hello world!")
	cr.CrunchLog.Print("Goodbye")
	cr.CrunchLog.Close()

	c.Check(api.Calls, Equals, 1)

	mt, err := cr.LogCollection.MarshalManifest(".")
	c.Check(err, IsNil)
	c.Check(mt, Equals, ". 74561df9ae65ee9f35d5661d42454264+83 0:83:crunch-run.txt\n")

	logtext := "2015-12-29T15:51:45.000000001Z Hello world!\n" +
		"2015-12-29T15:51:45.000000002Z Goodbye\n"

	c.Check(api.Content[0]["log"].(arvadosclient.Dict)["event_type"], Equals, "crunch-run")
	stderrLog := api.Content[0]["log"].(arvadosclient.Dict)["properties"].(map[string]string)["text"]
	c.Check(true, Equals, strings.Contains(stderrLog, expected))
	c.Check(string(kc.Content), Equals, logtext)
}
