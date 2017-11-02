// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/manifest"
)

type File interface {
	io.Reader
	io.Closer
	io.Seeker
	Size() int64
}

type keepClient interface {
	ManifestFileReader(manifest.Manifest, string) (File, error)
}

type collectionFile struct {
	File
	collection *Collection
	name       string
	size       int64
}

func (cf *collectionFile) Size() int64 {
	return cf.size
}

func (cf *collectionFile) Readdir(count int) ([]os.FileInfo, error) {
	return nil, io.EOF
}

func (cf *collectionFile) Stat() (os.FileInfo, error) {
	return collectionDirent{
		collection: cf.collection,
		name:       cf.name,
		size:       cf.size,
		isDir:      false,
	}, nil
}

type collectionDir struct {
	collection *Collection
	stream     string
	dirents    []os.FileInfo
}

// Readdir implements os.File.
func (cd *collectionDir) Readdir(count int) ([]os.FileInfo, error) {
	ret := cd.dirents
	if count <= 0 {
		cd.dirents = nil
		return ret, nil
	} else if len(ret) == 0 {
		return nil, io.EOF
	}
	var err error
	if count >= len(ret) {
		count = len(ret)
		err = io.EOF
	}
	cd.dirents = cd.dirents[count:]
	return ret[:count], err
}

// Stat implements os.File.
func (cd *collectionDir) Stat() (os.FileInfo, error) {
	return collectionDirent{
		collection: cd.collection,
		name:       path.Base(cd.stream),
		isDir:      true,
		size:       int64(len(cd.dirents)),
	}, nil
}

// Close implements os.File.
func (cd *collectionDir) Close() error {
	return nil
}

// Read implements os.File.
func (cd *collectionDir) Read([]byte) (int, error) {
	return 0, nil
}

// Seek implements os.File.
func (cd *collectionDir) Seek(int64, int) (int64, error) {
	return 0, nil
}

// collectionDirent implements os.FileInfo.
type collectionDirent struct {
	collection *Collection
	name       string
	isDir      bool
	mode       os.FileMode
	size       int64
}

// Name implements os.FileInfo.
func (e collectionDirent) Name() string {
	return e.name
}

// ModTime implements os.FileInfo.
func (e collectionDirent) ModTime() time.Time {
	if e.collection.ModifiedAt == nil {
		return time.Now()
	}
	return *e.collection.ModifiedAt
}

// Mode implements os.FileInfo.
func (e collectionDirent) Mode() os.FileMode {
	if e.isDir {
		return 0555
	} else {
		return 0444
	}
}

// IsDir implements os.FileInfo.
func (e collectionDirent) IsDir() bool {
	return e.isDir
}

// Size implements os.FileInfo.
func (e collectionDirent) Size() int64 {
	return e.size
}

// Sys implements os.FileInfo.
func (e collectionDirent) Sys() interface{} {
	return nil
}

// A CollectionFileSystem is an http.Filesystem with an added Stat() method.
type CollectionFileSystem interface {
	http.FileSystem
	Stat(name string) (os.FileInfo, error)
}

// collectionFS implements CollectionFileSystem.
type collectionFS struct {
	collection *Collection
	client     *Client
	kc         keepClient
	sizes      map[string]int64
	sizesOnce  sync.Once
}

// FileSystem returns a CollectionFileSystem for the collection.
func (c *Collection) FileSystem(client *Client, kc keepClient) CollectionFileSystem {
	return &collectionFS{
		collection: c,
		client:     client,
		kc:         kc,
	}
}

func (c *collectionFS) Stat(name string) (os.FileInfo, error) {
	name = canonicalName(name)
	if name == "." {
		return collectionDirent{
			collection: c.collection,
			name:       "/",
			isDir:      true,
		}, nil
	}
	if size, ok := c.fileSizes()[name]; ok {
		return collectionDirent{
			collection: c.collection,
			name:       path.Base(name),
			size:       size,
			isDir:      false,
		}, nil
	}
	for fnm := range c.fileSizes() {
		if !strings.HasPrefix(fnm, name+"/") {
			continue
		}
		return collectionDirent{
			collection: c.collection,
			name:       path.Base(name),
			isDir:      true,
		}, nil
	}
	return nil, os.ErrNotExist
}

func (c *collectionFS) Open(name string) (http.File, error) {
	// Ensure name looks the way it does in a manifest.
	name = canonicalName(name)

	m := manifest.Manifest{Text: c.collection.ManifestText}

	// Return a file if it exists.
	if size, ok := c.fileSizes()[name]; ok {
		reader, err := c.kc.ManifestFileReader(m, name)
		if err != nil {
			return nil, err
		}
		return &collectionFile{
			File:       reader,
			collection: c.collection,
			name:       path.Base(name),
			size:       size,
		}, nil
	}

	// Return a directory if it's the root dir or there are file
	// entries below it.
	children := map[string]collectionDirent{}
	for fnm, size := range c.fileSizes() {
		if !strings.HasPrefix(fnm, name+"/") {
			continue
		}
		isDir := false
		ent := fnm[len(name)+1:]
		if i := strings.Index(ent, "/"); i >= 0 {
			ent = ent[:i]
			isDir = true
		}
		e := children[ent]
		e.collection = c.collection
		e.isDir = isDir
		e.name = ent
		e.size = size
		children[ent] = e
	}
	if len(children) == 0 && name != "." {
		return nil, os.ErrNotExist
	}
	dirents := make([]os.FileInfo, 0, len(children))
	for _, ent := range children {
		dirents = append(dirents, ent)
	}
	return &collectionDir{
		collection: c.collection,
		stream:     name,
		dirents:    dirents,
	}, nil
}

// fileSizes returns a map of files that can be opened. Each key
// starts with "./".
func (c *collectionFS) fileSizes() map[string]int64 {
	c.sizesOnce.Do(func() {
		c.sizes = map[string]int64{}
		m := manifest.Manifest{Text: c.collection.ManifestText}
		for ms := range m.StreamIter() {
			for _, fss := range ms.FileStreamSegments {
				c.sizes[ms.StreamName+"/"+fss.Name] += int64(fss.SegLen)
			}
		}
	})
	return c.sizes
}

func canonicalName(name string) string {
	name = path.Clean("/" + name)
	if name == "/" || name == "./" {
		name = "."
	} else if strings.HasPrefix(name, "/") {
		name = "." + name
	}
	return name
}
