package arvados

import (
	"bufio"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// KeepService is an arvados#keepService record
type KeepService struct {
	UUID           string `json:"uuid"`
	ServiceHost    string `json:"service_host"`
	ServicePort    int    `json:"service_port"`
	ServiceSSLFlag bool   `json:"service_ssl_flag"`
	ServiceType    string `json:"service_type"`
	ReadOnly       bool   `json:"read_only"`
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

// Index returns an unsorted list of blocks that can be retrieved from
// this server.
func (s *KeepService) Index(c *Client, prefix string) ([]KeepServiceIndexEntry, error) {
	url := s.url("index/" + prefix)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("NewRequest(%v): %v", url, err)
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Do(%v): %v", url, err)
	} else if resp.StatusCode != 200 {
		return nil, fmt.Errorf("%v: %v", url, resp.Status)
	}
	defer resp.Body.Close()

	var entries []KeepServiceIndexEntry
	scanner := bufio.NewScanner(resp.Body)
	sawEOF := false
	for scanner.Scan() {
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
