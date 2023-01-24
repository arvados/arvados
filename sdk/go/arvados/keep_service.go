// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"bufio"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// KeepService is an arvados#keepService record
type KeepService struct {
	UUID           string    `json:"uuid"`
	ServiceHost    string    `json:"service_host"`
	ServicePort    int       `json:"service_port"`
	ServiceSSLFlag bool      `json:"service_ssl_flag"`
	ServiceType    string    `json:"service_type"`
	ReadOnly       bool      `json:"read_only"`
	CreatedAt      time.Time `json:"created_at"`
	ModifiedAt     time.Time `json:"modified_at"`
}

type KeepMount struct {
	UUID           string          `json:"uuid"`
	DeviceID       string          `json:"device_id"`
	ReadOnly       bool            `json:"read_only"`
	Replication    int             `json:"replication"`
	StorageClasses map[string]bool `json:"storage_classes"`
}

// KeepServiceList is an arvados#keepServiceList record
type KeepServiceList struct {
	Items          []KeepService `json:"items"`
	ItemsAvailable int           `json:"items_available"`
	Offset         int           `json:"offset"`
	Limit          int           `json:"limit"`
}

// KeepServiceIndexEntry is what a keep service's index response tells
// us about a stored block.
type KeepServiceIndexEntry struct {
	SizedDigest
	// Time of last write, in nanoseconds since Unix epoch
	Mtime int64
}

// EachKeepService calls f once for every readable
// KeepService. EachKeepService stops if it encounters an
// error, such as f returning a non-nil error.
func (c *Client) EachKeepService(f func(KeepService) error) error {
	params := ResourceListParams{}
	for {
		var page KeepServiceList
		err := c.RequestAndDecode(&page, "GET", "arvados/v1/keep_services", nil, params)
		if err != nil {
			return err
		}
		for _, item := range page.Items {
			err = f(item)
			if err != nil {
				return err
			}
		}
		params.Offset = params.Offset + len(page.Items)
		if params.Offset >= page.ItemsAvailable {
			return nil
		}
	}
}

func (s *KeepService) url(path string) string {
	var f string
	if s.ServiceSSLFlag {
		f = "https://%s:%d/%s"
	} else {
		f = "http://%s:%d/%s"
	}
	return fmt.Sprintf(f, s.ServiceHost, s.ServicePort, path)
}

// String implements fmt.Stringer
func (s *KeepService) String() string {
	return s.UUID
}

func (s *KeepService) Mounts(c *Client) ([]KeepMount, error) {
	url := s.url("mounts")
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	var mounts []KeepMount
	err = c.DoAndDecode(&mounts, req)
	if err != nil {
		return nil, fmt.Errorf("GET %v: %v", url, err)
	}
	return mounts, nil
}

// Touch updates the timestamp on the given block.
func (s *KeepService) Touch(ctx context.Context, c *Client, blk string) error {
	req, err := http.NewRequest("TOUCH", s.url(blk), nil)
	if err != nil {
		return err
	}
	resp, err := c.Do(req.WithContext(ctx))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("%s %s: %s", resp.Proto, resp.Status, body)
	}
	return nil
}

// Untrash moves/copies the given block out of trash.
func (s *KeepService) Untrash(ctx context.Context, c *Client, blk string) error {
	req, err := http.NewRequest("PUT", s.url("untrash/"+blk), nil)
	if err != nil {
		return err
	}
	resp, err := c.Do(req.WithContext(ctx))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("%s %s: %s", resp.Proto, resp.Status, body)
	}
	return nil
}

// IndexMount returns an unsorted list of blocks at the given mount point.
func (s *KeepService) IndexMount(ctx context.Context, c *Client, mountUUID string, prefix string) ([]KeepServiceIndexEntry, error) {
	return s.index(ctx, c, prefix, s.url("mounts/"+mountUUID+"/blocks?prefix="+prefix))
}

// Index returns an unsorted list of blocks that can be retrieved from
// this server.
func (s *KeepService) Index(ctx context.Context, c *Client, prefix string) ([]KeepServiceIndexEntry, error) {
	return s.index(ctx, c, prefix, s.url("index/"+prefix))
}

func (s *KeepService) index(ctx context.Context, c *Client, prefix, url string) ([]KeepServiceIndexEntry, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("NewRequestWithContext(%v): %v", url, err)
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Do(%v): %v", url, err)
	} else if resp.StatusCode != 200 {
		return nil, fmt.Errorf("%v: %d %v", url, resp.StatusCode, resp.Status)
	}
	defer resp.Body.Close()

	var entries []KeepServiceIndexEntry
	scanner := bufio.NewScanner(resp.Body)
	sawEOF := false
	for scanner.Scan() {
		if scanner.Err() != nil {
			// If we encounter a read error (timeout,
			// connection failure), stop now and return it
			// below, so it doesn't get masked by the
			// ensuing "badly formatted response" error.
			break
		}
		if sawEOF {
			return nil, fmt.Errorf("Index response contained non-terminal blank line")
		}
		line := scanner.Text()
		if line == "" {
			sawEOF = true
			continue
		}
		fields := strings.Split(line, " ")
		if len(fields) != 2 {
			return nil, fmt.Errorf("Malformed index line %q: %d fields", line, len(fields))
		}
		if !strings.HasPrefix(fields[0], prefix) {
			return nil, fmt.Errorf("Index response included block %q despite asking for prefix %q", fields[0], prefix)
		}
		mtime, err := strconv.ParseInt(fields[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("Malformed index line %q: mtime: %v", line, err)
		}
		if mtime < 1e12 {
			// An old version of keepstore is giving us
			// timestamps in seconds instead of
			// nanoseconds. (This threshold correctly
			// handles all times between 1970-01-02 and
			// 33658-09-27.)
			mtime = mtime * 1e9
		}
		entries = append(entries, KeepServiceIndexEntry{
			SizedDigest: SizedDigest(fields[0]),
			Mtime:       mtime,
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("Error scanning index response: %v", err)
	}
	if !sawEOF {
		return nil, fmt.Errorf("Index response had no EOF marker")
	}
	return entries, nil
}
