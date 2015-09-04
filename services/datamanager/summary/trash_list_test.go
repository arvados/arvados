package summary

import (
	"git.curoverse.com/arvados.git/sdk/go/blockdigest"
	"git.curoverse.com/arvados.git/services/datamanager/keep"
	. "gopkg.in/check.v1"
	"testing"
)

// Gocheck boilerplate
func TestTrash(t *testing.T) {
	TestingT(t)
}

type TrashSuite struct{}

var _ = Suite(&TrashSuite{})

func (s *TrashSuite) TestBuildTrashLists(c *C) {
	var sv0 = keep.ServerAddress{Host: "keep0.example.com", Port: 80}
	var sv1 = keep.ServerAddress{Host: "keep1.example.com", Port: 80}

	var block0 = blockdigest.MakeTestDigestWithSize(0xdeadbeef)
	var block1 = blockdigest.MakeTestDigestWithSize(0xfedbeef)

	var keepServerInfo = keep.ReadServers{
		KeepServerIndexToAddress: []keep.ServerAddress{sv0, sv1},
		BlockToServers: map[blockdigest.DigestWithSize][]keep.BlockServerInfo{
			block0: []keep.BlockServerInfo{
				keep.BlockServerInfo{0, 99},
				keep.BlockServerInfo{1, 101}},
			block1: []keep.BlockServerInfo{
				keep.BlockServerInfo{0, 99},
				keep.BlockServerInfo{1, 101}}}}

	// only block0 is in delete set
	var bs BlockSet = make(BlockSet)
	bs[block0] = struct{}{}

	// Test trash list where only sv0 is on writable list.
	c.Check(buildTrashListsInternal(
		map[string]struct{}{
			sv0.URL(): struct{}{}},
		&keepServerInfo,
		110,
		bs),
		DeepEquals,
		map[string]keep.TrashList{
			"http://keep0.example.com:80": keep.TrashList{keep.TrashRequest{"000000000000000000000000deadbeef", 99}}})

	// Test trash list where both sv0 and sv1 are on writable list.
	c.Check(buildTrashListsInternal(
		map[string]struct{}{
			sv0.URL(): struct{}{},
			sv1.URL(): struct{}{}},
		&keepServerInfo,
		110,
		bs),
		DeepEquals,
		map[string]keep.TrashList{
			"http://keep0.example.com:80": keep.TrashList{keep.TrashRequest{"000000000000000000000000deadbeef", 99}},
			"http://keep1.example.com:80": keep.TrashList{keep.TrashRequest{"000000000000000000000000deadbeef", 101}}})

	// Test trash list where only block on sv0 is expired
	c.Check(buildTrashListsInternal(
		map[string]struct{}{
			sv0.URL(): struct{}{},
			sv1.URL(): struct{}{}},
		&keepServerInfo,
		100,
		bs),
		DeepEquals,
		map[string]keep.TrashList{
			"http://keep0.example.com:80": keep.TrashList{keep.TrashRequest{"000000000000000000000000deadbeef", 99}}})

}
