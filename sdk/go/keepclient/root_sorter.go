package keepclient

import (
	"crypto/md5"
	"fmt"
	"sort"
)

type RootSorter struct {
	root         []string
	weight       []string
	order        []int
}

func NewRootSorter(serviceRoots map[string]string, hash string) (*RootSorter) {
	rs := new(RootSorter)
	rs.root = make([]string, len(serviceRoots))
	rs.weight = make([]string, len(serviceRoots))
	rs.order = make([]int, len(serviceRoots))
	i := 0
	for uuid, root := range serviceRoots {
		rs.root[i] = root
		rs.weight[i] = rs.getWeight(hash, uuid)
		rs.order[i] = i
		i++
	}
	sort.Sort(rs)
	return rs
}

func (rs RootSorter) getWeight(hash string, uuid string) (string) {
	var service_key []byte
	if len(uuid) == 27 {
		service_key = []byte(hash + uuid[12:])
	} else {
		// Only useful for testing, a set of one service root, etc.
		service_key = []byte(hash + uuid)
	}
	return fmt.Sprintf("%x", md5.Sum(service_key))
}

func (rs RootSorter) GetSortedRoots() ([]string) {
	sorted := make([]string, len(rs.order))
	for i := range rs.order {
		sorted[i] = rs.root[rs.order[i]]
	}
	return sorted
}

// Less is really More here: the heaviest root will be at the front of the list.
func (rs RootSorter) Less(i, j int) bool {
	return rs.weight[rs.order[j]] < rs.weight[rs.order[i]]
}

func (rs RootSorter) Len() int {
	return len(rs.order)
}

func (rs RootSorter) Swap(i, j int) {
	sort.IntSlice(rs.order).Swap(i, j)
}
