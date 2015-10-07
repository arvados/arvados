package main

import (
	"crypto/md5"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"git.curoverse.com/arvados.git/sdk/go/arvadostest"
	"git.curoverse.com/arvados.git/sdk/go/keepclient"

	. "gopkg.in/check.v1"
)

// Gocheck boilerplate
func Test(t *testing.T) {
	TestingT(t)
}

// Gocheck boilerplate
var _ = Suite(&ServerRequiredSuite{})

// Tests that require the Keep server running
type ServerRequiredSuite struct{}

func (s *ServerRequiredSuite) SetUpSuite(c *C) {
}

func (s *ServerRequiredSuite) SetUpTest(c *C) {
	arvadostest.ResetEnv()
	srcKeepServicesJSON = ""
	dstKeepServicesJSON = ""
}

func (s *ServerRequiredSuite) TearDownSuite(c *C) {
	arvadostest.StopKeep()
	arvadostest.StopAPI()
}

// Testing keep-rsync needs two sets of keep services: src and dst.
// The test setup hence tweaks keep-rsync initialzation to achieve this.
// First invoke initializeKeepRsync and then invoke StartKeepAdditional
// to create the keep servers to be used as destination.
func setupRsync(c *C) {
	// srcConfig
	srcConfig.APIHost = os.Getenv("ARVADOS_API_HOST")
	srcConfig.APIToken = os.Getenv("ARVADOS_API_TOKEN")
	srcConfig.APIHostInsecure = matchTrue.MatchString(os.Getenv("ARVADOS_API_HOST_INSECURE"))

	// dstConfig
	dstConfig.APIHost = os.Getenv("ARVADOS_API_HOST")
	dstConfig.APIToken = os.Getenv("ARVADOS_API_TOKEN")
	dstConfig.APIHostInsecure = matchTrue.MatchString(os.Getenv("ARVADOS_API_HOST_INSECURE"))

	replications = 1

	// Start API and Keep servers
	arvadostest.StartAPI()
	arvadostest.StartKeep()

	// initialize keep-rsync
	err := initializeKeepRsync()
	c.Assert(err, Equals, nil)

	// Create two more keep servers to be used as destination
	arvadostest.StartKeepAdditional(true)

	// load kcDst
	kcDst, err = keepclient.MakeKeepClient(&arvDst)
	c.Assert(err, Equals, nil)
	kcDst.Want_replicas = 1
}

// Test readConfigFromFile method
func (s *ServerRequiredSuite) TestReadConfigFromFile(c *C) {
	// Setup a test config file
	file, err := ioutil.TempFile(os.TempDir(), "config")
	c.Assert(err, Equals, nil)
	defer os.Remove(file.Name())

	fileContent := "ARVADOS_API_HOST=testhost\n"
	fileContent += "ARVADOS_API_TOKEN=testtoken\n"
	fileContent += "ARVADOS_API_HOST_INSECURE=true"

	_, err = file.Write([]byte(fileContent))

	// Invoke readConfigFromFile method with this test filename
	config, err := readConfigFromFile(file.Name())
	c.Assert(err, Equals, nil)
	c.Assert(config.APIHost, Equals, "testhost")
	c.Assert(config.APIToken, Equals, "testtoken")
	c.Assert(config.APIHostInsecure, Equals, true)
	c.Assert(config.ExternalClient, Equals, false)
}

// Test keep-rsync initialization, with src and dst keep servers.
// Do a Put and Get in src, both of which should succeed.
// Do a Put and Get in dst, both of which should succeed.
// Do a Get in dst for the src hash, which should raise block not found error.
// Do a Get in src for the dst hash, which should raise block not found error.
func (s *ServerRequiredSuite) TestRsyncPutInOne_GetFromOtherShouldFail(c *C) {
	setupRsync(c)

	// Put a block in src using kcSrc and Get it
	srcData := []byte("test-data1")
	locatorInSrc := fmt.Sprintf("%x", md5.Sum(srcData))

	hash, rep, err := kcSrc.PutB(srcData)
	c.Check(hash, Matches, fmt.Sprintf(`^%s\+10(\+.+)?$`, locatorInSrc))
	c.Check(rep, Equals, 2)
	c.Check(err, Equals, nil)

	reader, blocklen, _, err := kcSrc.Get(locatorInSrc)
	c.Assert(err, Equals, nil)
	c.Check(blocklen, Equals, int64(10))
	all, err := ioutil.ReadAll(reader)
	c.Check(all, DeepEquals, srcData)

	// Put a different block in src using kcSrc and Get it
	dstData := []byte("test-data2")
	locatorInDst := fmt.Sprintf("%x", md5.Sum(dstData))

	hash, rep, err = kcDst.PutB(dstData)
	c.Check(hash, Matches, fmt.Sprintf(`^%s\+10(\+.+)?$`, locatorInDst))
	c.Check(rep, Equals, 1)
	c.Check(err, Equals, nil)

	reader, blocklen, _, err = kcDst.Get(locatorInDst)
	c.Assert(err, Equals, nil)
	c.Check(blocklen, Equals, int64(10))
	all, err = ioutil.ReadAll(reader)
	c.Check(all, DeepEquals, dstData)

	// Get srcLocator using kcDst should fail with NotFound error
	_, _, _, err = kcDst.Get(locatorInSrc)
	c.Assert(err.Error(), Equals, "Block not found")

	// Get dstLocator using kcSrc should fail with NotFound error
	_, _, _, err = kcSrc.Get(locatorInDst)
	c.Assert(err.Error(), Equals, "Block not found")
}

// Test keep-rsync initialization, with srcKeepServicesJSON
func (s *ServerRequiredSuite) TestRsyncInitializeWithKeepServicesJSON(c *C) {
	srcKeepServicesJSON = "{ \"kind\":\"arvados#keepServiceList\", \"etag\":\"\", \"self_link\":\"\", \"offset\":null, \"limit\":null, \"items\":[ { \"href\":\"/keep_services/zzzzz-bi6l4-123456789012340\", \"kind\":\"arvados#keepService\", \"etag\":\"641234567890enhj7hzx432e5\", \"uuid\":\"zzzzz-bi6l4-123456789012340\", \"owner_uuid\":\"zzzzz-tpzed-123456789012345\", \"service_host\":\"keep0.zzzzz.arvadosapi.com\", \"service_port\":25107, \"service_ssl_flag\":false, \"service_type\":\"disk\", \"read_only\":false }, { \"href\":\"/keep_services/zzzzz-bi6l4-123456789012341\", \"kind\":\"arvados#keepService\", \"etag\":\"641234567890enhj7hzx432e5\", \"uuid\":\"zzzzz-bi6l4-123456789012341\", \"owner_uuid\":\"zzzzz-tpzed-123456789012345\", \"service_host\":\"keep0.zzzzz.arvadosapi.com\", \"service_port\":25108, \"service_ssl_flag\":false, \"service_type\":\"disk\", \"read_only\":false } ], \"items_available\":2 }"

	setupRsync(c)

	localRoots := kcSrc.LocalRoots()
	c.Check(localRoots != nil, Equals, true)

	foundIt := false
	for k := range localRoots {
		if k == "zzzzz-bi6l4-123456789012340" {
			foundIt = true
		}
	}
	c.Check(foundIt, Equals, true)

	foundIt = false
	for k := range localRoots {
		if k == "zzzzz-bi6l4-123456789012341" {
			foundIt = true
		}
	}
	c.Check(foundIt, Equals, true)
}

// Put 5 blocks in src. Put 2 of those blocks in dst
// Hence there are 3 additional blocks in src
// Also, put 2 extra blocks in dts; they are hence only in dst
// Run rsync and verify that those 7 blocks are now available in dst
func (s *ServerRequiredSuite) TestKeepRsync(c *C) {
	setupRsync(c)

	// Put a few blocks in src using kcSrc
	var srcLocators []string
	for i := 0; i < 5; i++ {
		data := []byte(fmt.Sprintf("test-data-%d", i))
		hash := fmt.Sprintf("%x", md5.Sum(data))

		hash2, rep, err := kcSrc.PutB(data)
		c.Check(hash2, Matches, fmt.Sprintf(`^%s\+11(\+.+)?$`, hash))
		c.Check(rep, Equals, 2)
		c.Check(err, Equals, nil)

		reader, blocklen, _, err := kcSrc.Get(hash)
		c.Assert(err, Equals, nil)
		c.Check(blocklen, Equals, int64(11))
		all, err := ioutil.ReadAll(reader)
		c.Check(all, DeepEquals, data)

		srcLocators = append(srcLocators, fmt.Sprintf("%s+%d", hash, blocklen))
	}

	// Put just two of those blocks in dst using kcDst
	var dstLocators []string
	for i := 0; i < 2; i++ {
		data := []byte(fmt.Sprintf("test-data-%d", i))
		hash := fmt.Sprintf("%x", md5.Sum(data))

		hash2, rep, err := kcDst.PutB(data)
		c.Check(hash2, Matches, fmt.Sprintf(`^%s\+11(\+.+)?$`, hash))
		c.Check(rep, Equals, 1)
		c.Check(err, Equals, nil)

		reader, blocklen, _, err := kcDst.Get(hash)
		c.Assert(err, Equals, nil)
		c.Check(blocklen, Equals, int64(11))
		all, err := ioutil.ReadAll(reader)
		c.Check(all, DeepEquals, data)

		dstLocators = append(dstLocators, fmt.Sprintf("%s+%d", hash, blocklen))
	}

	// Put two more blocks in dst; they are not in src at all
	var extraDstLocators []string
	for i := 0; i < 2; i++ {
		data := []byte(fmt.Sprintf("other-data-%d", i))
		hash := fmt.Sprintf("%x", md5.Sum(data))

		hash2, rep, err := kcDst.PutB(data)
		c.Check(hash2, Matches, fmt.Sprintf(`^%s\+12(\+.+)?$`, hash))
		c.Check(rep, Equals, 1)
		c.Check(err, Equals, nil)

		reader, blocklen, _, err := kcDst.Get(hash)
		c.Assert(err, Equals, nil)
		c.Check(blocklen, Equals, int64(12))
		all, err := ioutil.ReadAll(reader)
		c.Check(all, DeepEquals, data)

		extraDstLocators = append(extraDstLocators, fmt.Sprintf("%s+%d", hash, blocklen))
	}

	err := performKeepRsync()
	c.Check(err, Equals, nil)

	// Now GetIndex from dst and verify that all 5 from src and the 2 extra blocks are found
	dstIndex, err := getUniqueLocators(kcDst, "")
	c.Check(err, Equals, nil)
	for _, locator := range srcLocators {
		_, ok := dstIndex[locator]
		c.Assert(ok, Equals, true)
	}
	for _, locator := range extraDstLocators {
		_, ok := dstIndex[locator]
		c.Assert(ok, Equals, true)
	}
}
