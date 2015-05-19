// Code for generating pull lists as described in https://arvados.org/projects/arvados/wiki/Keep_Design_Doc#Pull-List
package summary

import (
	"encoding/json"
	"fmt"
	"git.curoverse.com/arvados.git/sdk/go/blockdigest"
	"git.curoverse.com/arvados.git/sdk/go/keepclient"
	"git.curoverse.com/arvados.git/sdk/go/logger"
	"git.curoverse.com/arvados.git/services/datamanager/keep"
	"git.curoverse.com/arvados.git/services/datamanager/loggerutil"
	"log"
	"os"
	"strings"
)

type Locator struct {
	Digest blockdigest.BlockDigest
	// TODO(misha): Add size field to the Locator (and MarshalJSON() below)
}

func (l Locator) MarshalJSON() ([]byte, error) {
	return []byte("\"" + l.Digest.String() + "\""), nil
}

// One entry in the Pull List
type PullListEntry struct {
	Locator Locator  `json:"locator"`
	Servers []string `json:"servers"`
}

// The Pull List for a particular server
type PullList struct {
	Entries []PullListEntry `json:"blocks"`
}

// EntriesByDigest implements sort.Interface for []PullListEntry
// based on the Digest.
type EntriesByDigest []PullListEntry

func (a EntriesByDigest) Len() int      { return len(a) }
func (a EntriesByDigest) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a EntriesByDigest) Less(i, j int) bool {
	di, dj := a[i].Locator.Digest, a[j].Locator.Digest
	return di.H < dj.H || (di.H == dj.H && di.L < dj.L)
}

// For a given under-replicated block, this structure represents which
// servers should pull the specified block and which servers they can
// pull it from.
type PullServers struct {
	To   []string // Servers that should pull the specified block
	From []string // Servers that already contain the specified block
}

// Creates a map from block locator to PullServers with one entry for
// each under-replicated block.
func ComputePullServers(kc *keepclient.KeepClient,
	keepServerInfo *keep.ReadServers,
	blockToDesiredReplication map[blockdigest.BlockDigest]int,
	underReplicated BlockSet) (m map[Locator]PullServers) {
	m = map[Locator]PullServers{}
	// We use CanonicalString to avoid filling memory with dupicate
	// copies of the same string.
	var cs CanonicalString

	for block, _ := range underReplicated {
		serversStoringBlock := keepServerInfo.BlockToServers[block]
		numCopies := len(serversStoringBlock)
		numCopiesMissing := blockToDesiredReplication[block] - numCopies
		if numCopiesMissing > 0 {
			// We expect this to always be true, since the block was listed
			// in underReplicated.
			// TODO(misha): Consider asserting the above conditional.

			if numCopies > 0 {
				// I believe that we should expect this to always be true.

				// A server's host-port string appears as a key in this map
				// iff it contains the block.
				serverHasBlock := map[string]struct{}{}
				for _, info := range serversStoringBlock {
					sa := keepServerInfo.KeepServerIndexToAddress[info.ServerIndex]
					serverHasBlock[sa.HostPort()] = struct{}{}
				}

				roots := keepclient.NewRootSorter(kc.LocalRoots(),
					block.String()).GetSortedRoots()

				l := Locator{Digest: block}
				m[l] = CreatePullServers(cs, serverHasBlock, roots, numCopiesMissing)
			}
		}
	}
	return m
}

// Creates a pull list in which the To and From fields preserve the
// ordering of sorted servers and the contents are all canonical
// strings.
func CreatePullServers(cs CanonicalString,
	serverHasBlock map[string]struct{},
	sortedServers []string,
	maxToFields int) (ps PullServers) {

	ps = PullServers{
		To:   make([]string, 0, maxToFields),
		From: make([]string, 0, len(serverHasBlock)),
	}

	for _, host := range sortedServers {
		// Strip the protocol portion of the url.
		// Use the canonical copy of the string to avoid memory waste.
		server := cs.Get(RemoveProtocolPrefix(host))
		_, hasBlock := serverHasBlock[server]
		if hasBlock {
			ps.From = append(ps.From, server)
		} else if len(ps.To) < maxToFields {
			ps.To = append(ps.To, server)
		}
	}

	return
}

// Strips the protocol prefix from a url.
func RemoveProtocolPrefix(url string) string {
	return url[(strings.LastIndex(url, "/") + 1):]
}

// Produces a PullList for each keep server.
func BuildPullLists(lps map[Locator]PullServers) (spl map[string]PullList) {
	spl = map[string]PullList{}
	// We don't worry about canonicalizing our strings here, because we
	// assume lps was created by ComputePullServers() which already
	// canonicalized the strings for us.
	for locator, pullServers := range lps {
		for _, destination := range pullServers.To {
			pullList, pullListExists := spl[destination]
			if !pullListExists {
				pullList = PullList{Entries: []PullListEntry{}}
				spl[destination] = pullList
			}
			pullList.Entries = append(pullList.Entries,
				PullListEntry{Locator: locator, Servers: pullServers.From})
			spl[destination] = pullList
		}
	}
	return
}

// Writes each pull list to a file.
// The filename is based on the hostname.
//
// This is just a hack for prototyping, it is not expected to be used
// in production.
func WritePullLists(arvLogger *logger.Logger,
	pullLists map[string]PullList) {
	r := strings.NewReplacer(":", ".")
	for host, list := range pullLists {
		filename := fmt.Sprintf("pull_list.%s", r.Replace(host))
		pullListFile, err := os.Create(filename)
		if err != nil {
			loggerutil.FatalWithMessage(arvLogger,
				fmt.Sprintf("Failed to open %s: %v", filename, err))
		}
		defer pullListFile.Close()

		enc := json.NewEncoder(pullListFile)
		err = enc.Encode(list)
		if err != nil {
			loggerutil.FatalWithMessage(arvLogger,
				fmt.Sprintf("Failed to write pull list to %s: %v", filename, err))
		}
		log.Printf("Wrote pull list to %s.", filename)
	}
}
