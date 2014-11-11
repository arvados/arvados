package keepclient

import (
	"fmt"
	. "gopkg.in/check.v1"
	"strconv"
	"strings"
)

type RootSorterSuite struct{}
var _ = Suite(&RootSorterSuite{})

func FakeSvcRoot(i uint64) (string) {
	return fmt.Sprintf("https://%x.svc/", i)
}

func FakeSvcUuid(i uint64) (string) {
	return fmt.Sprintf("zzzzz-bi6l4-%015x", i)
}

func FakeServiceRoots(n uint64) (map[string]string) {
	sr := map[string]string{}
	for i := uint64(0); i < n; i ++ {
		sr[FakeSvcUuid(i)] = FakeSvcRoot(i)
	}
	return sr
}

func (*RootSorterSuite) EmptyRoots(c *C) {
	rs := NewRootSorter(map[string]string{}, Md5String("foo"))
	c.Check(rs.GetSortedRoots(), Equals, []string{})
}

func (*RootSorterSuite) JustOneRoot(c *C) {
	rs := NewRootSorter(FakeServiceRoots(1), Md5String("foo"))
	c.Check(rs.GetSortedRoots(), Equals, []string{FakeSvcRoot(0)})
}

func (*RootSorterSuite) ReferenceSet(c *C) {
	fakeroots := FakeServiceRoots(16)
	// These reference probe orders are explained further in
	// ../../python/arvados/keep.py:
	expected_orders := []string{
		"3eab2d5fc9681074",
		"097dba52e648f1c3",
		"c5b4e023f8a7d691",
		"9d81c02e76a3bf54",
	}
	for h, expected_order := range expected_orders {
		hash := Md5String(fmt.Sprintf("%064x", h))
		roots := NewRootSorter(fakeroots, hash).GetSortedRoots()
		for i, svc_id_s := range strings.Split(expected_order, "") {
			svc_id, err := strconv.ParseUint(svc_id_s, 16, 64)
			c.Assert(err, Equals, nil)
			c.Check(roots[i], Equals, FakeSvcRoot(svc_id))
		}
	}
}
