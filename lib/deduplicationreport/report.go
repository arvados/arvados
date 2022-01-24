// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package deduplicationreport

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"git.arvados.org/arvados.git/lib/cmd"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadosclient"
	"git.arvados.org/arvados.git/sdk/go/manifest"

	"github.com/dustin/go-humanize"
	"github.com/sirupsen/logrus"
)

func deDuplicate(inputs []string) (trimmed []string) {
	seen := make(map[string]bool)
	for _, uuid := range inputs {
		if !seen[uuid] {
			seen[uuid] = true
			trimmed = append(trimmed, uuid)
		}
	}
	return
}

// parseFlags returns either some inputs to process, or (if there are
// no inputs to process) a nil slice and a suitable exit code.
func parseFlags(prog string, args []string, logger *logrus.Logger, stderr io.Writer) (inputs []string, exitcode int) {
	flags := flag.NewFlagSet(prog, flag.ContinueOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), `
Usage:
  %s [options ...] <collection-uuid> <collection-uuid> ...

  %s [options ...] <collection-pdh>,<collection-uuid> \
     <collection-pdh>,<collection-uuid> ...

  This program analyzes the overlap in blocks used by 2 or more collections. It
  prints a deduplication report that shows the nominal space used by the
  collections, as well as the actual size and the amount of space that is saved
  by Keep's deduplication.

  The list of collections may be provided in two ways. A list of collection
  uuids is sufficient. Alternatively, the PDH for each collection may also be
  provided. This is will greatly speed up operation when the list contains
  multiple collections with the same PDH.

  Exit status will be zero if there were no errors generating the report.

Example:

  Use the 'arv' and 'jq' commands to get the list of the 100
  largest collections and generate the deduplication report:

  arv collection list --order 'file_size_total desc' --limit 100 | \
    jq -r '.items[] | [.portable_data_hash,.uuid] |@csv' | \
    sed -e 's/"//g'|tr '\n' ' ' | \
    xargs %s

Options:
`, prog, prog, prog)
		flags.PrintDefaults()
	}
	loglevel := flags.String("log-level", "info", "logging level (debug, info, ...)")
	if ok, code := cmd.ParseFlags(flags, prog, args, "collection-uuid [...]", stderr); !ok {
		return nil, code
	}

	inputs = deDuplicate(flags.Args())

	if len(inputs) < 1 {
		fmt.Fprintf(stderr, "Error: no collections provided\n")
		return nil, 2
	}

	lvl, err := logrus.ParseLevel(*loglevel)
	if err != nil {
		fmt.Fprintf(stderr, "Error: cannot parse log level: %s\n", err)
		return nil, 2
	}
	logger.SetLevel(lvl)
	return inputs, 0
}

func blockList(collection arvados.Collection) (blocks map[string]int) {
	blocks = make(map[string]int)
	m := manifest.Manifest{Text: collection.ManifestText}
	blockChannel := m.BlockIterWithDuplicates()
	for b := range blockChannel {
		blocks[b.Digest.String()] = b.Size
	}
	return
}

func collectionsInProject(ac *arvados.Client, UUID string) ([]string, error) {
	var UUIDs []string
	if !strings.Contains(UUID, "-j7d0g-") {
		return nil, fmt.Errorf("Error: UUID must refer to project object")
	}

	var collections, projects map[string]interface{}
	filterSet := []arvados.Filter{
		{
			Attr:     "owner_uuid",
			Operator: "=",
			Operand:  UUID,
		},
	}

	// Recursively add the collections of any subprojects
	// FIXME handle paging
	err := ac.RequestAndDecode(&projects, "GET", "arvados/v1/groups", nil, map[string]interface{}{
		"filters": filterSet,
		"limit":   10000,
	})
	if err != nil {
		return nil, fmt.Errorf("error querying subprojects: %s", err.Error())
	}
	if value, ok := projects["items"]; ok {
		items := value.([]interface{})
		for _, item := range items {
			itemMap := item.(map[string]interface{})
			if _, ok := itemMap["uuid"]; !ok {
				return nil, fmt.Errorf("Error: no uuid key in itemMap (projects)")
			}
			subUUIDs, err := collectionsInProject(ac, itemMap["uuid"].(string))
			if err != nil {
				return nil, err
			}
			UUIDs = append(UUIDs, subUUIDs...)
		}
	}

	// Add collections in this project
	// FIXME handle paging
	err = ac.RequestAndDecode(&collections, "GET", "arvados/v1/collections", nil, map[string]interface{}{
		"filters": filterSet,
		"limit":   10000,
	})
	if err != nil {
		return nil, fmt.Errorf("error querying collections: %s", err.Error())
	}
	if value, ok := collections["items"]; ok {
		items := value.([]interface{})
		for _, item := range items {
			itemMap := item.(map[string]interface{})
			if _, ok := itemMap["uuid"]; !ok {
				return nil, fmt.Errorf("Error: no uuid key in itemMap (collections)")
			}
			if _, ok := itemMap["portable_data_hash"]; !ok {
				return nil, fmt.Errorf("Error: no portable_data_hash key in itemMap (collections)")
			}
			UUIDs = append(UUIDs, itemMap["portable_data_hash"].(string)+","+itemMap["uuid"].(string))
		}
	}
	fmt.Fprintf(os.Stderr, ".")

	return UUIDs, nil
}

func report(prog string, args []string, logger *logrus.Logger, stdout, stderr io.Writer) (exitcode int) {
	var inputs []string

	inputs, exitcode = parseFlags(prog, args, logger, stderr)
	if inputs == nil {
		return
	}

	// Arvados Client setup
	arv, err := arvadosclient.MakeArvadosClient()
	if err != nil {
		logger.Errorf("Error creating Arvados object: %s", err)
		exitcode = 1
		return
	}

	type Col struct {
		FileSizeTotal int64
		FileCount     int64
	}

	blocks := make(map[string]map[string]int)
	pdhs := make(map[string]Col)
	var nominalSize int64
	ac := arvados.NewClientFromEnv()
	var extra []string
	for _, input := range inputs {
		if !strings.Contains(input, ",") && strings.Contains(input, "-j7d0g-") {
			fmt.Fprintf(os.Stderr, "Getting collections in %s and its sub-projects", input)
			tmp, err := collectionsInProject(ac, input)
			if err != nil {
				logger.Errorf("Error: %s", err.Error())
				exitcode = 1
				return
			}
			extra = append(extra, tmp...)
			fmt.Fprintf(os.Stderr, "\n")
		}
	}

	inputs = append(inputs, extra...)

	for _, input := range inputs {
		var uuid string
		var pdh string
		if strings.Contains(input, ",") {
			// The input is in the format pdh,uuid. This will allow us to save time on duplicate pdh's
			tmp := strings.Split(input, ",")
			pdh = tmp[0]
			uuid = tmp[1]
		} else {
			// The input must be a plain uuid
			uuid = input
		}

		if strings.Contains(uuid, "-j7d0g-") {
			// Already expanded to the list of collections in the project, above
			continue
		}
		if !strings.Contains(uuid, "-4zz18-") {
			logger.Errorf("Error: uuid must refer to collection object")
			exitcode = 1
			return
		}
		if _, ok := pdhs[pdh]; ok {
			// We've processed a collection with this pdh already. Simply add its
			// size to the totals and move on to the next one.
			// Note that we simply trust the PDH matches the collection UUID here,
			// in other words, we use it over the UUID. If they don't match, the report
			// will be wrong.
			nominalSize += pdhs[pdh].FileSizeTotal
		} else {
			var collection arvados.Collection
			err = arv.Get("collections", uuid, nil, &collection)
			if err != nil {
				logger.Errorf("Error: unable to retrieve collection: %s", err)
				exitcode = 1
				return
			}
			blocks[uuid] = make(map[string]int)
			blocks[uuid] = blockList(collection)
			if pdh != "" && collection.PortableDataHash != pdh {
				logger.Errorf("Error: the collection with UUID %s has PDH %s, but a different PDH was provided in the arguments: %s", uuid, collection.PortableDataHash, pdh)
				exitcode = 1
				return
			}
			if pdh == "" {
				pdh = collection.PortableDataHash
			}

			col := Col{}
			if collection.FileSizeTotal != 0 || collection.FileCount != 0 {
				nominalSize += collection.FileSizeTotal
				col.FileSizeTotal = collection.FileSizeTotal
				col.FileCount = int64(collection.FileCount)
			} else {
				// Collections created with old Arvados versions do not always have the total file size and count cached in the collections object
				var collSize int64
				for _, size := range blocks[uuid] {
					collSize += int64(size)
				}
				nominalSize += collSize
				col.FileSizeTotal = collSize
			}
			pdhs[pdh] = col
		}

		if pdhs[pdh].FileCount != 0 {
			fmt.Fprintf(stdout, "Collection %s: pdh %s; nominal size %d (%s); file count %d\n", uuid, pdh, pdhs[pdh].FileSizeTotal, humanize.IBytes(uint64(pdhs[pdh].FileSizeTotal)), pdhs[pdh].FileCount)
		} else {
			fmt.Fprintf(stdout, "Collection %s: pdh %s; nominal size %d (%s)\n", uuid, pdh, pdhs[pdh].FileSizeTotal, humanize.IBytes(uint64(pdhs[pdh].FileSizeTotal)))
		}
	}

	var totalSize int64
	seen := make(map[string]bool)
	for _, v := range blocks {
		for pdh, size := range v {
			if !seen[pdh] {
				seen[pdh] = true
				totalSize += int64(size)
			}
		}
	}
	fmt.Fprintln(stdout)
	fmt.Fprintf(stdout, "Collections:                 %15d\n", len(inputs))
	fmt.Fprintf(stdout, "Nominal size of stored data: %15d bytes (%s)\n", nominalSize, humanize.IBytes(uint64(nominalSize)))
	fmt.Fprintf(stdout, "Actual size of stored data:  %15d bytes (%s)\n", totalSize, humanize.IBytes(uint64(totalSize)))
	fmt.Fprintf(stdout, "Saved by Keep deduplication: %15d bytes (%s)\n", nominalSize-totalSize, humanize.IBytes(uint64(nominalSize-totalSize)))

	return exitcode
}
