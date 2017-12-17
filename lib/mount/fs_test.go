package mount

import (
	"github.com/curoverse/cgofuse/fuse"
)

var _ fuse.FileSystem = &keepFS{}
