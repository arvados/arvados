// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package localdb

import (
	"context"
	"errors"

	"git.arvados.org/arvados.git/lib/config"
	"git.arvados.org/arvados.git/lib/controller/rpc"
	"git.arvados.org/arvados.git/lib/ctrlctx"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"github.com/jmoiron/sqlx"
	check "gopkg.in/check.v1"
)

type localdbSuite struct {
	ctx         context.Context
	cancel      context.CancelFunc
	cluster     *arvados.Cluster
	db          *sqlx.DB
	dbConnector *ctrlctx.DBConnector
	tx          *sqlx.Tx
	txFinish    func(*error)
	userctx     context.Context // uses ActiveUser token
	localdb     *Conn
	railsSpy    *arvadostest.Proxy
}

func (s *localdbSuite) SetUpSuite(c *check.C) {
	arvadostest.StartKeep(2, true)
}

func (s *localdbSuite) TearDownSuite(c *check.C) {
	// Undo any changes/additions to the user database so they
	// don't affect subsequent tests.
	arvadostest.ResetEnv()
	c.Check(arvados.NewClientFromEnv().RequestAndDecode(nil, "POST", "database/reset", nil, nil), check.IsNil)
}

func (s *localdbSuite) SetUpTest(c *check.C) {
	*s = localdbSuite{}
	s.ctx, s.cancel = context.WithCancel(context.Background())
	cfg, err := config.NewLoader(nil, ctxlog.TestLogger(c)).Load()
	c.Assert(err, check.IsNil)
	s.cluster, err = cfg.GetCluster("")
	c.Assert(err, check.IsNil)
	s.dbConnector = &ctrlctx.DBConnector{PostgreSQL: s.cluster.PostgreSQL}
	s.db, err = s.dbConnector.GetDB(s.ctx)
	c.Assert(err, check.IsNil)
	s.ctx, s.txFinish = ctrlctx.New(s.ctx, s.dbConnector.GetDB)
	s.tx, err = ctrlctx.CurrentTx(s.ctx)
	c.Assert(err, check.IsNil)
	s.localdb = NewConn(s.ctx, s.cluster, s.dbConnector.GetDB)
	s.railsSpy = arvadostest.NewProxy(c, s.cluster.Services.RailsAPI)
	*s.localdb.railsProxy = *rpc.NewConn(s.cluster.ClusterID, s.railsSpy.URL, true, rpc.PassthroughTokenProvider)
	s.userctx = ctrlctx.NewWithToken(s.ctx, s.cluster, arvadostest.ActiveTokenV2)
}

var errRollbackAfterTest = errors.New("rollback after test")

func (s *localdbSuite) TearDownTest(c *check.C) {
	if s.tx != nil {
		s.tx.Rollback()
	}
	if s.txFinish != nil {
		s.txFinish(&errRollbackAfterTest)
	}
	if s.railsSpy != nil {
		s.railsSpy.Close()
	}
	if s.dbConnector != nil {
		s.dbConnector.Close()
	}
	s.cancel()
}

func (s *localdbSuite) setUpVocabulary(c *check.C, testVocabulary string) {
	if testVocabulary == "" {
		testVocabulary = `{
			"strict_tags": false,
			"tags": {
				"IDTAGIMPORTANCES": {
					"strict": true,
					"labels": [{"label": "Importance"}, {"label": "Priority"}],
					"values": {
						"IDVALIMPORTANCES1": { "labels": [{"label": "Critical"}, {"label": "Urgent"}, {"label": "High"}] },
						"IDVALIMPORTANCES2": { "labels": [{"label": "Normal"}, {"label": "Moderate"}] },
						"IDVALIMPORTANCES3": { "labels": [{"label": "Low"}] }
					}
				}
			}
		}`
	}
	voc, err := arvados.NewVocabulary([]byte(testVocabulary), []string{})
	c.Assert(err, check.IsNil)
	s.localdb.vocabularyCache = voc
	s.cluster.API.VocabularyPath = "foo"
}
