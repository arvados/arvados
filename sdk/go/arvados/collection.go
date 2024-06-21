// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"regexp"
	"strings"
	"time"

	"git.arvados.org/arvados.git/sdk/go/blockdigest"
)

var (
	UUIDMatch = regexp.MustCompile(`^[a-z0-9]{5}-[a-z0-9]{5}-[a-z0-9]{15}$`).MatchString
	PDHMatch  = regexp.MustCompile(`^[0-9a-f]{32}\+\d+$`).MatchString
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
	manifestText := []byte(c.ManifestText)
	if len(manifestText) == 0 {
		manifestText = []byte(c.UnsignedManifestText)
	}
	if len(manifestText) == 0 && c.PortableDataHash != "d41d8cd98f00b204e9800998ecf8427e+0" {
		// TODO: Check more subtle forms of corruption, too
		return nil, fmt.Errorf("manifest is missing")
	}
	sds := make([]SizedDigest, 0, len(manifestText)/40)
	for _, line := range bytes.Split(manifestText, []byte{'\n'}) {
		if len(line) == 0 {
			continue
		}
		tokens := bytes.Split(line, []byte{' '})
		if len(tokens) < 3 {
			return nil, fmt.Errorf("Invalid stream (<3 tokens): %q", line)
		}
		for _, token := range tokens[1:] {
			if !blockdigest.LocatorPattern.Match(token) {
				// FIXME: ensure it's a file token
				break
			}
			if bytes.HasPrefix(token, []byte("d41d8cd98f00b204e9800998ecf8427e+0")) {
				// Exclude "empty block" placeholder
				continue
			}
			// FIXME: shouldn't assume 32 char hash
			if i := bytes.IndexRune(token[33:], '+'); i >= 0 {
				token = token[:33+i]
			}
			sds = append(sds, SizedDigest(string(token)))
		}
	}
	return sds, nil
}

type CollectionList struct {
	Items          []Collection `json:"items"`
	ItemsAvailable int          `json:"items_available"`
	Offset         int          `json:"offset"`
	Limit          int          `json:"limit"`
}

// PortableDataHash computes the portable data hash of the given
// manifest.
func PortableDataHash(mt string) string {
	// To calculate the PDH, we write the manifest to an md5 hash
	// func, except we skip the "extra" part of block tokens that
	// look like "abcdef0123456789abcdef0123456789+12345+extra".
	//
	// This code is simplified by the facts that (A) all block
	// tokens -- even the first and last in a stream -- are
	// preceded and followed by a space character; and (B) all
	// non-block tokens either start with '.'  or contain ':'.
	//
	// A regexp-based approach (like the one this replaced) would
	// be more readable, but very slow.
	h := md5.New()
	size := 0
	todo := []byte(mt)
	for len(todo) > 0 {
		// sp is the end of the current token (note that if
		// the current token is the last file token in a
		// stream, we'll also include the \n and the dirname
		// token on the next line, which is perfectly fine for
		// our purposes).
		sp := bytes.IndexByte(todo, ' ')
		if sp < 0 {
			// Last token of the manifest, which is never
			// a block token.
			n, _ := h.Write(todo)
			size += n
			break
		}
		if sp >= 34 && todo[32] == '+' && bytes.IndexByte(todo[:32], ':') == -1 && todo[0] != '.' {
			// todo[:sp] is a block token.
			sizeend := bytes.IndexByte(todo[33:sp], '+')
			if sizeend < 0 {
				// "hash+size"
				sizeend = sp
			} else {
				// "hash+size+extra"
				sizeend += 33
			}
			n, _ := h.Write(todo[:sizeend])
			h.Write([]byte{' '})
			size += n + 1
		} else {
			// todo[:sp] is not a block token.
			n, _ := h.Write(todo[:sp+1])
			size += n
		}
		todo = todo[sp+1:]
	}
	return fmt.Sprintf("%x+%d", h.Sum(nil), size)
}

// CollectionIDFromDNSName returns a UUID or PDH if s begins with a
// UUID or URL-encoded PDH; otherwise "".
func CollectionIDFromDNSName(s string) string {
	// Strip domain.
	if i := strings.IndexRune(s, '.'); i >= 0 {
		s = s[:i]
	}
	// Names like {uuid}--collections.example.com serve the same
	// purpose as {uuid}.collections.example.com but can reduce
	// cost/effort of using [additional] wildcard certificates.
	if i := strings.Index(s, "--"); i >= 0 {
		s = s[:i]
	}
	if UUIDMatch(s) {
		return s
	}
	if pdh := strings.Replace(s, "-", "+", 1); PDHMatch(pdh) {
		return pdh
	}
	return ""
}
