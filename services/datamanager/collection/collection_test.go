package collection

import (
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/arvadostest"
	"git.curoverse.com/arvados.git/sdk/go/blockdigest"
	. "gopkg.in/check.v1"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Gocheck boilerplate
func Test(t *testing.T) {
	TestingT(t)
}

type MySuite struct{}

var _ = Suite(&MySuite{})

// This captures the result we expect from
// ReadCollections.Summarize().  Because CollectionUUIDToIndex is
// indeterminate, we replace BlockToCollectionIndices with
// BlockToCollectionUuids.
type ExpectedSummary struct {
	OwnerToCollectionSize     map[string]int
	BlockToDesiredReplication map[blockdigest.DigestWithSize]int
	BlockToCollectionUuids    map[blockdigest.DigestWithSize][]string
}

func CompareSummarizedReadCollections(c *C,
	summarized ReadCollections,
	expected ExpectedSummary) {

	c.Assert(summarized.OwnerToCollectionSize, DeepEquals,
		expected.OwnerToCollectionSize)

	c.Assert(summarized.BlockToDesiredReplication, DeepEquals,
		expected.BlockToDesiredReplication)

	summarizedBlockToCollectionUuids :=
		make(map[blockdigest.DigestWithSize]map[string]struct{})
	for digest, indices := range summarized.BlockToCollectionIndices {
		uuidSet := make(map[string]struct{})
		summarizedBlockToCollectionUuids[digest] = uuidSet
		for _, index := range indices {
			uuidSet[summarized.CollectionIndexToUUID[index]] = struct{}{}
		}
	}

	expectedBlockToCollectionUuids :=
		make(map[blockdigest.DigestWithSize]map[string]struct{})
	for digest, uuidSlice := range expected.BlockToCollectionUuids {
		uuidSet := make(map[string]struct{})
		expectedBlockToCollectionUuids[digest] = uuidSet
		for _, uuid := range uuidSlice {
			uuidSet[uuid] = struct{}{}
		}
	}

	c.Assert(summarizedBlockToCollectionUuids, DeepEquals,
		expectedBlockToCollectionUuids)
}

func (s *MySuite) TestSummarizeSimple(checker *C) {
	rc := MakeTestReadCollections([]TestCollectionSpec{TestCollectionSpec{
		ReplicationLevel: 5,
		Blocks:           []int{1, 2},
	}})

	rc.Summarize(nil)

	c := rc.UUIDToCollection["col0"]

	blockDigest1 := blockdigest.MakeTestDigestWithSize(1)
	blockDigest2 := blockdigest.MakeTestDigestWithSize(2)

	expected := ExpectedSummary{
		OwnerToCollectionSize:     map[string]int{c.OwnerUUID: c.TotalSize},
		BlockToDesiredReplication: map[blockdigest.DigestWithSize]int{blockDigest1: 5, blockDigest2: 5},
		BlockToCollectionUuids:    map[blockdigest.DigestWithSize][]string{blockDigest1: []string{c.UUID}, blockDigest2: []string{c.UUID}},
	}

	CompareSummarizedReadCollections(checker, rc, expected)
}

func (s *MySuite) TestSummarizeOverlapping(checker *C) {
	rc := MakeTestReadCollections([]TestCollectionSpec{
		TestCollectionSpec{
			ReplicationLevel: 5,
			Blocks:           []int{1, 2},
		},
		TestCollectionSpec{
			ReplicationLevel: 8,
			Blocks:           []int{2, 3},
		},
	})

	rc.Summarize(nil)

	c0 := rc.UUIDToCollection["col0"]
	c1 := rc.UUIDToCollection["col1"]

	blockDigest1 := blockdigest.MakeTestDigestWithSize(1)
	blockDigest2 := blockdigest.MakeTestDigestWithSize(2)
	blockDigest3 := blockdigest.MakeTestDigestWithSize(3)

	expected := ExpectedSummary{
		OwnerToCollectionSize: map[string]int{
			c0.OwnerUUID: c0.TotalSize,
			c1.OwnerUUID: c1.TotalSize,
		},
		BlockToDesiredReplication: map[blockdigest.DigestWithSize]int{
			blockDigest1: 5,
			blockDigest2: 8,
			blockDigest3: 8,
		},
		BlockToCollectionUuids: map[blockdigest.DigestWithSize][]string{
			blockDigest1: []string{c0.UUID},
			blockDigest2: []string{c0.UUID, c1.UUID},
			blockDigest3: []string{c1.UUID},
		},
	}

	CompareSummarizedReadCollections(checker, rc, expected)
}

type APITestData struct {
	// path and response map
	responses map[string]arvadostest.StubResponse

	// expected error, if any
	expectedError string
}

func (s *MySuite) TestGetCollectionsAndSummarize_DiscoveryError(c *C) {
	testGetCollectionsAndSummarize(c,
		APITestData{
			responses:     make(map[string]arvadostest.StubResponse),
			expectedError: "arvados API server error: 500.*",
		})
}

func (s *MySuite) TestGetCollectionsAndSummarize_ApiErrorGetCollections(c *C) {
	respMap := make(map[string]arvadostest.StubResponse)
	respMap["/discovery/v1/apis/arvados/v1/rest"] = arvadostest.StubResponse{200, `{"defaultCollectionReplication":2}`}
	respMap["/arvados/v1/collections"] = arvadostest.StubResponse{-1, ``}

	testGetCollectionsAndSummarize(c,
		APITestData{
			responses:     respMap,
			expectedError: "arvados API server error: 302.*",
		})
}

func (s *MySuite) TestGetCollectionsAndSummarize_GetCollectionsBadStreamName(c *C) {
	respMap := make(map[string]arvadostest.StubResponse)
	respMap["/discovery/v1/apis/arvados/v1/rest"] = arvadostest.StubResponse{200, `{"defaultCollectionReplication":2}`}
	respMap["/arvados/v1/collections"] = arvadostest.StubResponse{200, `{"items_available":1,"items":[{"modified_at":"2015-11-24T15:04:05Z","manifest_text":"badstreamname"}]}`}

	testGetCollectionsAndSummarize(c,
		APITestData{
			responses:     respMap,
			expectedError: "Invalid stream name: badstreamname",
		})
}

func (s *MySuite) TestGetCollectionsAndSummarize_GetCollectionsBadFileToken(c *C) {
	respMap := make(map[string]arvadostest.StubResponse)
	respMap["/discovery/v1/apis/arvados/v1/rest"] = arvadostest.StubResponse{200, `{"defaultCollectionReplication":2}`}
	respMap["/arvados/v1/collections"] = arvadostest.StubResponse{200, `{"items_available":1,"items":[{"modified_at":"2015-11-24T15:04:05Z","manifest_text":"./goodstream acbd18db4cc2f85cedef654fccc4a4d8+3 0:1:file1.txt file2.txt"}]}`}

	testGetCollectionsAndSummarize(c,
		APITestData{
			responses:     respMap,
			expectedError: "Invalid file token: file2.txt",
		})
}

func testGetCollectionsAndSummarize(c *C, testData APITestData) {
	apiStub := arvadostest.ServerStub{testData.responses}

	api := httptest.NewServer(&apiStub)
	defer api.Close()

	arv := arvadosclient.ArvadosClient{
		Scheme:    "http",
		ApiServer: api.URL[7:],
		ApiToken:  "abc123",
		Client:    &http.Client{Transport: &http.Transport{}},
	}

	// GetCollectionsAndSummarize
	_, err := GetCollectionsAndSummarize(GetCollectionsParams{arv, nil, 10})

	if testData.expectedError == "" {
		c.Assert(err, IsNil)
	} else {
		c.Assert(err, ErrorMatches, testData.expectedError)
	}
}
