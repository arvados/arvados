// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"bufio"
	"crypto/md5"
	"fmt"
	"regexp"
	"strings"
	"time"

	"git.arvados.org/arvados.git/sdk/go/blockdigest"
)

// Collection is an arvados#collection resource.
type Collection struct {
	UUID                      string                 `json:"uuid"`
	Etag                      string                 `json:"etag"`
	OwnerUUID                 string                 `json:"owner_uuid"`
	TrashAt                   *time.Time             `json:"trash_at"`
	ManifestText              string                 `json:"manifest_text"`
	UnsignedManifestText      string                 `json:"unsigned_manifest_text"`
	Name                      string                 `json:"name"`
	CreatedAt                 time.Time              `json:"created_at"`
	ModifiedAt                time.Time              `json:"modified_at"`
	ModifiedByClientUUID      string                 `json:"modified_by_client_uuid"`
	ModifiedByUserUUID        string                 `json:"modified_by_user_uuid"`
	PortableDataHash          string                 `json:"portable_data_hash"`
	ReplicationConfirmed      *int                   `json:"replication_confirmed"`
	ReplicationConfirmedAt    *time.Time             `json:"replication_confirmed_at"`
	ReplicationDesired        *int                   `json:"replication_desired"`
	StorageClassesDesired     []string               `json:"storage_classes_desired"`
	StorageClassesConfirmed   []string               `json:"storage_classes_confirmed"`
	StorageClassesConfirmedAt *time.Time             `json:"storage_classes_confirmed_at"`
	DeleteAt                  *time.Time             `json:"delete_at"`
	IsTrashed                 bool                   `json:"is_trashed"`
	Properties                map[string]interface{} `json:"properties"`
	WritableBy                []string               `json:"writable_by,omitempty"`
	FileCount                 int                    `json:"file_count"`
	FileSizeTotal             int64                  `json:"file_size_total"`
	Version                   int                    `json:"version"`
	PreserveVersion           bool                   `json:"preserve_version"`
	CurrentVersionUUID        string                 `json:"current_version_uuid"`
	Description               string                 `json:"description"`
}

func (c Collection) resourceName() string {
	return "collection"
}

// SizedDigests returns the hash+size part of each data block
// referenced by the collection.
//
// Zero-length blocks are not included.
func (c *Collection) SizedDigests() ([]SizedDigest, error) {
	manifestText := c.ManifestText
	if manifestText == "" {
		manifestText = c.UnsignedManifestText
	}
	if manifestText == "" && c.PortableDataHash != "d41d8cd98f00b204e9800998ecf8427e+0" {
		// TODO: Check more subtle forms of corruption, too
		return nil, fmt.Errorf("manifest is missing")
	}
	var sds []SizedDigest
	scanner := bufio.NewScanner(strings.NewReader(manifestText))
	scanner.Buffer(make([]byte, 1048576), len(manifestText))
	for scanner.Scan() {
		line := scanner.Text()
		tokens := strings.Split(line, " ")
		if len(tokens) < 3 {
			return nil, fmt.Errorf("Invalid stream (<3 tokens): %q", line)
		}
		for _, token := range tokens[1:] {
			if !blockdigest.LocatorPattern.MatchString(token) {
				// FIXME: ensure it's a file token
				break
			}
			if strings.HasPrefix(token, "d41d8cd98f00b204e9800998ecf8427e+0") {
				// Exclude "empty block" placeholder
				continue
			}
			// FIXME: shouldn't assume 32 char hash
			if i := strings.IndexRune(token[33:], '+'); i >= 0 {
				token = token[:33+i]
			}
			sds = append(sds, SizedDigest(token))
		}
	}
	return sds, scanner.Err()
}

type CollectionList struct {
	Items          []Collection `json:"items"`
	ItemsAvailable int          `json:"items_available"`
	Offset         int          `json:"offset"`
	Limit          int          `json:"limit"`
}

var (
	blkRe = regexp.MustCompile(`^ [0-9a-f]{32}\+\d+`)
	tokRe = regexp.MustCompile(` ?[^ ]*`)
)

// PortableDataHash computes the portable data hash of the given
// manifest.
func PortableDataHash(mt string) string {
	h := md5.New()
	size := 0
	_ = tokRe.ReplaceAllFunc([]byte(mt), func(tok []byte) []byte {
		if m := blkRe.Find(tok); m != nil {
			// write hash+size, ignore remaining block hints
			tok = m
		}
		n, err := h.Write(tok)
		if err != nil {
			panic(err)
		}
		size += n
		return nil
	})
	return fmt.Sprintf("%x+%d", h.Sum(nil), size)
}
