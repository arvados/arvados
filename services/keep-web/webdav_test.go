package main

import "golang.org/x/net/webdav"

var _ webdav.FileSystem = &webdavFS{}
