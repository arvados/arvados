// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package keepclient

import (
	"errors"
	"os"

	"git.arvados.org/arvados.git/sdk/go/arvados"
)

// ErrNoManifest indicates the given collection has no manifest
// information (e.g., manifest_text was excluded by a "select"
// parameter when retrieving the collection record).
var ErrNoManifest = errors.New("Collection has no manifest")

// CollectionFileReader returns a Reader that reads content from a single file
// in the collection. The filename must be relative to the root of the
// collection.  A leading prefix of "/" or "./" in the filename is ignored.
func (kc *KeepClient) CollectionFileReader(collection map[string]interface{}, filename string) (arvados.File, error) {
	mText, ok := collection["manifest_text"].(string)
	if !ok {
		return nil, ErrNoManifest
	}
	fs, err := (&arvados.Collection{ManifestText: mText}).FileSystem(nil, kc)
	if err != nil {
		return nil, err
	}
	return fs.OpenFile(filename, os.O_RDONLY, 0)
}
