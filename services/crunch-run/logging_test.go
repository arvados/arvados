package main

import (
	"fmt"
	. "gopkg.in/check.v1"
	"time"
)

type LoggingTestSuite struct{}

type TestTimestamper struct {
	count int
}

func (this *TestTimestamper) Timestamp(t time.Time) string {
	this.count += 1
	return fmt.Sprintf("2015-12-29T15:51:45.%09dZ", this.count)
}

// Gocheck boilerplate
var _ = Suite(&LoggingTestSuite{})

func (s *LoggingTestSuite) TestWriteLogs(c *C) {
	api := &ArvTestClient{}
	kc := &KeepTestClient{}
	cr := NewContainerRunner(api, kc, nil)
	cr.CrunchLog.Timestamper = (&TestTimestamper{}).Timestamp

	cr.CrunchLog.Print("Hello world!")
	cr.CrunchLog.Print("Goodbye")
	cr.CrunchLog.Close()

	c.Check(api.Calls, Equals, 1)

	mt, err := cr.LogCollection.ManifestText()
	c.Check(err, IsNil)
	c.Check(mt, Equals, ". 74561df9ae65ee9f35d5661d42454264+83 0:83:crunch-run.txt\n")

	logtext := "2015-12-29T15:51:45.000000001Z Hello world!\n" +
		"2015-12-29T15:51:45.000000002Z Goodbye\n"

	c.Check(api.Content["event_type"], Equals, "crunch-run")
	c.Check(api.Content["properties"].(map[string]string)["text"], Equals, logtext)
	c.Check(string(kc.Content), Equals, logtext)
}

func (s *LoggingTestSuite) TestWriteLogsLarge(c *C) {
	api := &ArvTestClient{}
	kc := &KeepTestClient{}
	cr := NewContainerRunner(api, kc, nil)
	cr.CrunchLog.Timestamper = (&TestTimestamper{}).Timestamp

	for i := 0; i < 2000000; i += 1 {
		cr.CrunchLog.Printf("Hello %d", i)
	}
	cr.CrunchLog.Print("Goodbye")
	cr.CrunchLog.Close()

	c.Check(api.Calls > 1, Equals, true)
	c.Check(api.Calls < 2000000, Equals, true)

	mt, err := cr.LogCollection.ManifestText()
	c.Check(err, IsNil)
	c.Check(mt, Equals, ". 9c2c05d1fae6aaa8af85113ba725716d+67108864 80b821383a07266c2a66a4566835e26e+21780065 0:88888929:crunch-run.txt\n")
}

func (s *LoggingTestSuite) TestWriteMultipleLogs(c *C) {
	api := &ArvTestClient{}
	kc := &KeepTestClient{}
	cr := NewContainerRunner(api, kc, nil)
	ts := &TestTimestamper{}
	cr.CrunchLog.Timestamper = ts.Timestamp
	stdout := NewThrottledLogger(cr.NewLogWriter("stdout"))
	stdout.Timestamper = ts.Timestamp

	cr.CrunchLog.Print("Hello world!")
	stdout.Print("Doing stuff")
	cr.CrunchLog.Print("Goodbye")
	stdout.Print("Blurb")

	cr.CrunchLog.Close()
	logtext1 := "2015-12-29T15:51:45.000000001Z Hello world!\n" +
		"2015-12-29T15:51:45.000000003Z Goodbye\n"
	c.Check(api.Content["event_type"], Equals, "crunch-run")
	c.Check(api.Content["properties"].(map[string]string)["text"], Equals, logtext1)

	stdout.Close()
	logtext2 := "2015-12-29T15:51:45.000000002Z Doing stuff\n" +
		"2015-12-29T15:51:45.000000004Z Blurb\n"
	c.Check(api.Content["event_type"], Equals, "stdout")
	c.Check(api.Content["properties"].(map[string]string)["text"], Equals, logtext2)

	mt, err := cr.LogCollection.ManifestText()
	c.Check(err, IsNil)
	c.Check(mt, Equals, ""+
		". 408672f5b5325f7d20edfbf899faee42+83 0:83:crunch-run.txt\n"+
		". c556a293010069fa79a6790a931531d5+80 0:80:stdout.txt\n")
}
