package summary

import (
	"encoding/json"
	"git.curoverse.com/arvados.git/sdk/go/blockdigest"
	. "gopkg.in/check.v1"
	"sort"
	"testing"
)

// Gocheck boilerplate
func Test(t *testing.T) {
	TestingT(t)
}

type MySuite struct{}

var _ = Suite(&MySuite{})

// Helper method to declare string sets more succinctly
// Could be placed somewhere more general.
func stringSet(slice ...string) (m map[string]struct{}) {
	m = map[string]struct{}{}
	for _, s := range slice {
		m[s] = struct{}{}
	}
	return
}

func (s *MySuite) TestPullListPrintsJSONCorrectly(c *C) {
	pl := PullList{PullRequest{
		Locator: Locator(blockdigest.MakeTestDigestSpecifySize(0xBadBeef, 56789)),
		Servers: []string{"keep0.qr1hi.arvadosapi.com:25107",
			"keep1.qr1hi.arvadosapi.com:25108"}}}

	b, err := json.Marshal(pl)
	c.Assert(err, IsNil)
	expectedOutput := `[{"locator":"0000000000000000000000000badbeef+56789",` +
		`"servers":["keep0.qr1hi.arvadosapi.com:25107",` +
		`"keep1.qr1hi.arvadosapi.com:25108"]}]`
	c.Check(string(b), Equals, expectedOutput)
}

func (s *MySuite) TestCreatePullServers(c *C) {
	var cs CanonicalString
	c.Check(
		CreatePullServers(cs,
			stringSet(),
			stringSet(),
			[]string{},
			5),
		DeepEquals,
		PullServers{To: []string{}, From: []string{}})

	c.Check(
		CreatePullServers(cs,
			stringSet("https://keep0:25107", "https://keep1:25108"),
			stringSet(),
			[]string{},
			5),
		DeepEquals,
		PullServers{To: []string{}, From: []string{}})

	c.Check(
		CreatePullServers(cs,
			stringSet("https://keep0:25107", "https://keep1:25108"),
			stringSet("https://keep0:25107"),
			[]string{"https://keep0:25107"},
			5),
		DeepEquals,
		PullServers{To: []string{}, From: []string{"https://keep0:25107"}})

	c.Check(
		CreatePullServers(cs,
			stringSet("https://keep0:25107", "https://keep1:25108"),
			stringSet("https://keep3:25110", "https://keep2:25109", "https://keep1:25108", "https://keep0:25107"),
			[]string{"https://keep3:25110", "https://keep2:25109", "https://keep1:25108", "https://keep0:25107"},
			5),
		DeepEquals,
		PullServers{To: []string{"https://keep3:25110", "https://keep2:25109"},
			From: []string{"https://keep1:25108", "https://keep0:25107"}})

	c.Check(
		CreatePullServers(cs,
			stringSet("https://keep0:25107", "https://keep1:25108"),
			stringSet("https://keep3:25110", "https://keep1:25108", "https://keep0:25107"),
			[]string{"https://keep3:25110", "https://keep2:25109", "https://keep1:25108", "https://keep0:25107"},
			5),
		DeepEquals,
		PullServers{To: []string{"https://keep3:25110"},
			From: []string{"https://keep1:25108", "https://keep0:25107"}})

	c.Check(
		CreatePullServers(cs,
			stringSet("https://keep0:25107", "https://keep1:25108"),
			stringSet("https://keep3:25110", "https://keep2:25109", "https://keep1:25108", "https://keep0:25107"),
			[]string{"https://keep3:25110", "https://keep2:25109", "https://keep1:25108", "https://keep0:25107"},
			1),
		DeepEquals,
		PullServers{To: []string{"https://keep3:25110"},
			From: []string{"https://keep1:25108", "https://keep0:25107"}})

	c.Check(
		CreatePullServers(cs,
			stringSet("https://keep0:25107", "https://keep1:25108"),
			stringSet("https://keep3:25110", "https://keep2:25109",
				"https://keep1:25108", "https://keep0:25107"),
			[]string{"https://keep3:25110", "https://keep2:25109",
				"https://keep1:25108", "https://keep0:25107"},
			1),
		DeepEquals,
		PullServers{To: []string{"https://keep3:25110"},
			From: []string{"https://keep1:25108", "https://keep0:25107"}})

	c.Check(
		CreatePullServers(cs,
			stringSet("https://keep0:25107", "https://keep1:25108"),
			stringSet("https://keep3:25110", "https://keep2:25109", "https://keep1:25108", "https://keep0:25107"),
			[]string{"https://keep3:25110", "https://keep2:25109", "https://keep1:25108", "https://keep0:25107"},
			0),
		DeepEquals,
		PullServers{To: []string{},
			From: []string{"https://keep1:25108", "https://keep0:25107"}})
}

// Checks whether two pull list maps are equal. Since pull lists are
// ordered arbitrarily, we need to sort them by digest before
// comparing them for deep equality.
type pullListMapEqualsChecker struct {
	*CheckerInfo
}

func (c *pullListMapEqualsChecker) Check(params []interface{}, names []string) (result bool, error string) {
	obtained, ok := params[0].(map[string]PullList)
	if !ok {
		return false, "First parameter is not a PullList map"
	}
	expected, ok := params[1].(map[string]PullList)
	if !ok {
		return false, "Second parameter is not a PullList map"
	}

	for _, v := range obtained {
		sort.Sort(PullListByLocator(v))
	}
	for _, v := range expected {
		sort.Sort(PullListByLocator(v))
	}

	return DeepEquals.Check(params, names)
}

var PullListMapEquals Checker = &pullListMapEqualsChecker{&CheckerInfo{
	Name:   "PullListMapEquals",
	Params: []string{"obtained", "expected"},
}}

func (s *MySuite) TestBuildPullLists(c *C) {
	c.Check(
		BuildPullLists(map[Locator]PullServers{}),
		PullListMapEquals,
		map[string]PullList{})

	locator1 := Locator{Digest: blockdigest.MakeTestBlockDigest(0xBadBeef)}
	c.Check(
		BuildPullLists(map[Locator]PullServers{
			locator1: PullServers{To: []string{}, From: []string{}}}),
		PullListMapEquals,
		map[string]PullList{})

	c.Check(
		BuildPullLists(map[Locator]PullServers{
			locator1: PullServers{To: []string{}, From: []string{"f1", "f2"}}}),
		PullListMapEquals,
		map[string]PullList{})

	c.Check(
		BuildPullLists(map[Locator]PullServers{
			locator1: PullServers{To: []string{"t1"}, From: []string{"f1", "f2"}}}),
		PullListMapEquals,
		map[string]PullList{
			"t1": PullList{PullRequest{locator1, []string{"f1", "f2"}}}})

	c.Check(
		BuildPullLists(map[Locator]PullServers{
			locator1: PullServers{To: []string{"t1"}, From: []string{}}}),
		PullListMapEquals,
		map[string]PullList{"t1": PullList{
			PullRequest{locator1, []string{}}}})

	c.Check(
		BuildPullLists(map[Locator]PullServers{
			locator1: PullServers{
				To:   []string{"t1", "t2"},
				From: []string{"f1", "f2"},
			}}),
		PullListMapEquals,
		map[string]PullList{
			"t1": PullList{PullRequest{locator1, []string{"f1", "f2"}}},
			"t2": PullList{PullRequest{locator1, []string{"f1", "f2"}}},
		})

	locator2 := Locator{Digest: blockdigest.MakeTestBlockDigest(0xCabbed)}
	c.Check(
		BuildPullLists(map[Locator]PullServers{
			locator1: PullServers{To: []string{"t1"}, From: []string{"f1", "f2"}},
			locator2: PullServers{To: []string{"t2"}, From: []string{"f3", "f4"}}}),
		PullListMapEquals,
		map[string]PullList{
			"t1": PullList{PullRequest{locator1, []string{"f1", "f2"}}},
			"t2": PullList{PullRequest{locator2, []string{"f3", "f4"}}},
		})

	c.Check(
		BuildPullLists(map[Locator]PullServers{
			locator1: PullServers{
				To:   []string{"t1"},
				From: []string{"f1", "f2"}},
			locator2: PullServers{
				To:   []string{"t2", "t1"},
				From: []string{"f3", "f4"}},
		}),
		PullListMapEquals,
		map[string]PullList{
			"t1": PullList{
				PullRequest{locator1, []string{"f1", "f2"}},
				PullRequest{locator2, []string{"f3", "f4"}},
			},
			"t2": PullList{
				PullRequest{locator2, []string{"f3", "f4"}},
			},
		})

	locator3 := Locator{Digest: blockdigest.MakeTestBlockDigest(0xDeadBeef)}
	locator4 := Locator{Digest: blockdigest.MakeTestBlockDigest(0xFedBeef)}
	c.Check(
		BuildPullLists(map[Locator]PullServers{
			locator1: PullServers{
				To:   []string{"t1"},
				From: []string{"f1", "f2"}},
			locator2: PullServers{
				To:   []string{"t2", "t1"},
				From: []string{"f3", "f4"}},
			locator3: PullServers{
				To:   []string{"t3", "t2", "t1"},
				From: []string{"f4", "f5"}},
			locator4: PullServers{
				To:   []string{"t4", "t3", "t2", "t1"},
				From: []string{"f1", "f5"}},
		}),
		PullListMapEquals,
		map[string]PullList{
			"t1": PullList{
				PullRequest{locator1, []string{"f1", "f2"}},
				PullRequest{locator2, []string{"f3", "f4"}},
				PullRequest{locator3, []string{"f4", "f5"}},
				PullRequest{locator4, []string{"f1", "f5"}},
			},
			"t2": PullList{
				PullRequest{locator2, []string{"f3", "f4"}},
				PullRequest{locator3, []string{"f4", "f5"}},
				PullRequest{locator4, []string{"f1", "f5"}},
			},
			"t3": PullList{
				PullRequest{locator3, []string{"f4", "f5"}},
				PullRequest{locator4, []string{"f1", "f5"}},
			},
			"t4": PullList{
				PullRequest{locator4, []string{"f1", "f5"}},
			},
		})
}
