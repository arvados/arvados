package main

import (
	"database/sql"
	"log"
	"strings"
	"sync"

	_ "github.com/lib/pq"
)

type pgConfig map[string]string

func (c pgConfig) ConnectionString() string {
	s := ""
	for k, v := range c {
		s += k
		s += "='"
		s += strings.Replace(
			strings.Replace(v, `\`, `\\`, -1),
			`'`, `\'`, -1)
		s += "' "
	}
	return s
}

type pgEventSource struct {
	PgConfig  pgConfig
	QueueSize int

	db        *sql.DB
	setupOnce sync.Once
}

func (es *pgEventSource) setup() {
	db, err := sql.Open("postgres", es.PgConfig.ConnectionString())
	if err != nil {
		log.Fatal(err)
	}
	es.db = db
}

func (es *pgEventSource) EventSource() <-chan event {
	es.setupOnce.Do(es.setup)
	return nil
}
