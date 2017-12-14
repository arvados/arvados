package main

import (
	"github.com/billziss-gh/cgofuse/fuse"
)

var _ fuse.FileSystem = &keepFS{}
