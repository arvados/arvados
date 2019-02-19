// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package test

import (
	"os"

	"github.com/sirupsen/logrus"
)

func Logger() logrus.FieldLogger {
	logger := logrus.StandardLogger()
	if os.Getenv("ARVADOS_DEBUG") != "" {
		logger.SetLevel(logrus.DebugLevel)
	}
	return logger
}
