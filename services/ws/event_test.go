package main

import (
	"database/sql"

	"git.curoverse.com/arvados.git/sdk/go/config"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&eventSuite{})

type eventSuite struct{}

func (*eventSuite) TestDetail(c *check.C) {
	var railsDB struct {
		Test struct {
			Database string
			Username string
			Password string
			Host     string
		}
	}
	err := config.LoadFile(&railsDB, "../api/config/database.yml")
	c.Assert(err, check.IsNil)
	cfg := pgConfig{
		"dbname":   railsDB.Test.Database,
		"host":     railsDB.Test.Host,
		"password": railsDB.Test.Password,
		"user":     railsDB.Test.Username,
	}
	db, err := sql.Open("postgres", cfg.ConnectionString())
	c.Assert(err, check.IsNil)
	e := &event{
		LogID: 17,
		db:    db,
	}
	logRow := e.Detail()
	c.Assert(logRow, check.NotNil)
	c.Check(logRow, check.Equals, e.logRow)
	c.Check(logRow.UUID, check.Equals, "zzzzz-57u5n-containerlog006")
	c.Check(logRow.ObjectUUID, check.Equals, "zzzzz-dz642-logscontainer03")
	c.Check(logRow.EventType, check.Equals, "crunchstat")
	c.Check(logRow.Properties["text"], check.Equals, "2013-11-07_23:33:41 zzzzz-8i9sb-ahd7cie8jah9qui 29610 1 stderr crunchstat: cpu 1935.4300 user 59.4100 sys 8 cpus -- interval 10.0002 seconds 12.9900 user 0.9900 sys")
}
