// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package mount

import (
	"github.com/curoverse/cgofuse/fuse"
)

var _ fuse.FileSystem = &keepFS{}
