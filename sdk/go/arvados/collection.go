package arvados

import (
	"bufio"
	"fmt"
	"strings"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/manifest"
)

// Collection is an arvados#collection resource.
type Collection struct {
	UUID                   string     `json:"uuid,omitempty"`
	ExpiresAt              *time.Time `json:"expires_at,omitempty"`
	ManifestText           string     `json:"manifest_text,omitempty"`
	UnsignedManifestText   string     `json:"unsigned_manifest_text,omitempty"`
	CreatedAt              *time.Time `json:"created_at,omitempty"`
	ModifiedAt             *time.Time `json:"modified_at,omitempty"`
	PortableDataHash       string     `json:"portable_data_hash,omitempty"`
	ReplicationConfirmed   *int       `json:"replication_confirmed,omitempty"`
	ReplicationConfirmedAt *time.Time `json:"replication_confirmed_at,omitempty"`
	ReplicationDesired     *int       `json:"replication_desired,omitempty"`
}

// SizedDigests returns the hash+size part of each data block
// referenced by the collection.
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
			if !manifest.LocatorPattern.MatchString(token) {
				// FIXME: ensure it's a file token
				break
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

// CollectionList is an arvados#collectionList resource.
type CollectionList struct {
	Items          []Collection `json:"items"`
	ItemsAvailable int          `json:"items_available"`
	Offset         int          `json:"offset"`
	Limit          int          `json:"limit"`
}
