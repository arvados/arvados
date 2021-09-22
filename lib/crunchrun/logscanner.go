// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package crunchrun

import (
	"bytes"
	"strings"
)

// logScanner is an io.Writer that calls ReportFunc(pattern) the first
// time one of the Patterns appears in the data. Patterns must not
// contain newlines.
type logScanner struct {
	Patterns   []string
	ReportFunc func(pattern, text string)
	reported   bool
	buf        bytes.Buffer
}

func (s *logScanner) Write(p []byte) (int, error) {
	if s.reported {
		// We only call reportFunc once. Once we've called it
		// there's no need to buffer/search subsequent writes.
		return len(p), nil
	}
	split := bytes.LastIndexByte(p, '\n')
	if split < 0 {
		return s.buf.Write(p)
	}
	s.buf.Write(p[:split+1])
	txt := s.buf.String()
	for _, pattern := range s.Patterns {
		if found := strings.Index(txt, pattern); found >= 0 {
			// Report the entire line where the pattern
			// was found.
			txt = txt[strings.LastIndexByte(txt[:found], '\n')+1:]
			if end := strings.IndexByte(txt, '\n'); end >= 0 {
				txt = txt[:end]
			}
			s.ReportFunc(pattern, txt)
			s.reported = true
			return len(p), nil
		}
	}
	s.buf.Reset()
	if split == len(p) {
		return len(p), nil
	}
	n, err := s.buf.Write(p[split+1:])
	return n + split + 1, err
}
