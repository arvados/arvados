// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepweb

import "golang.org/x/net/webdav"

var _ webdav.FileSystem = &webdavFS{}
