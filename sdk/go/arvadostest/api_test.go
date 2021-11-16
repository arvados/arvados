// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvadostest

import "git.arvados.org/arvados.git/sdk/go/arvados"

// Test that *APIStub implements arvados.API
var _ arvados.API = &APIStub{}
