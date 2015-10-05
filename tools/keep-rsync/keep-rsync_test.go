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
	srcConfig = make(map[string]string)
	srcConfig["ARVADOS_API_HOST"] = os.Getenv("ARVADOS_API_HOST")
	srcConfig["ARVADOS_API_TOKEN"] = os.Getenv("ARVADOS_API_TOKEN")
	srcConfig["ARVADOS_API_HOST_INSECURE"] = os.Getenv("ARVADOS_API_HOST_INSECURE")

	// dstConfig
	dstConfig = make(map[string]string)
	dstConfig["ARVADOS_API_HOST"] = os.Getenv("ARVADOS_API_HOST")
	dstConfig["ARVADOS_API_TOKEN"] = os.Getenv("ARVADOS_API_TOKEN")
	dstConfig["ARVADOS_API_HOST_INSECURE"] = os.Getenv("ARVADOS_API_HOST_INSECURE")

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
	c.Assert(config["ARVADOS_API_HOST"], Equals, "testhost")
	c.Assert(config["ARVADOS_API_TOKEN"], Equals, "testtoken")
	c.Assert(config["ARVADOS_API_HOST_INSECURE"], Equals, "true")
	c.Assert(config["EXTERNAL_CLIENT"], Equals, "")
}

// Test keep-rsync initialization, with src and dst keep servers.
// Do a Put and Get in src, both of which should succeed.
// Do a Get in dst for the same hash, which should raise block not found error.
func (s *ServerRequiredSuite) TestRsyncPutInSrc_GetFromDstShouldFail(c *C) {
	setupRsync(c)

	// Put a block in src using kcSrc and Get it
	data := []byte("test-data")
	hash := fmt.Sprintf("%x", md5.Sum(data))

	hash2, rep, err := kcSrc.PutB(data)
	c.Check(hash2, Matches, fmt.Sprintf(`^%s\+9(\+.+)?$`, hash))
	c.Check(rep, Equals, 2)
	c.Check(err, Equals, nil)

	reader, blocklen, _, err := kcSrc.Get(hash)
	c.Assert(err, Equals, nil)
	c.Check(blocklen, Equals, int64(9))
	all, err := ioutil.ReadAll(reader)
	c.Check(all, DeepEquals, data)

	// Get using kcDst should fail with NotFound error
	_, _, _, err = kcDst.Get(hash)
	c.Assert(err.Error(), Equals, "Block not found")
}

// Test keep-rsync initialization, with srcKeepServicesJSON
func (s *ServerRequiredSuite) TestRsyncInitializeWithKeepServicesJSON(c *C) {
	srcKeepServicesJSON = "{ \"kind\":\"arvados#keepServiceList\", \"etag\":\"\", \"self_link\":\"\", \"offset\":null, \"limit\":null, \"items\":[ { \"href\":\"/keep_services/zzzzz-bi6l4-123456789012340\", \"kind\":\"arvados#keepService\", \"etag\":\"641234567890enhj7hzx432e5\", \"uuid\":\"zzzzz-bi6l4-123456789012340\", \"owner_uuid\":\"zzzzz-tpzed-123456789012345\", \"service_host\":\"keep0.zzzzz.arvadosapi.com\", \"service_port\":25107, \"service_ssl_flag\":false, \"service_type\":\"disk\", \"read_only\":false }, { \"href\":\"/keep_services/zzzzz-bi6l4-123456789012341\", \"kind\":\"arvados#keepService\", \"etag\":\"641234567890enhj7hzx432e5\", \"uuid\":\"zzzzz-bi6l4-123456789012341\", \"owner_uuid\":\"zzzzz-tpzed-123456789012345\", \"service_host\":\"keep0.zzzzz.arvadosapi.com\", \"service_port\":25108, \"service_ssl_flag\":false, \"service_type\":\"disk\", \"read_only\":false } ], \"items_available\":2 }"

	setupRsync(c)

	localRoots := kcSrc.LocalRoots()
	c.Check(localRoots != nil, Equals, true)

	foundIt := false
	for k, _ := range localRoots {
		if k == "zzzzz-bi6l4-123456789012340" {
			foundIt = true
		}
	}
	c.Check(foundIt, Equals, true)

	foundIt = false
	for k, _ := range localRoots {
		if k == "zzzzz-bi6l4-123456789012341" {
			foundIt = true
		}
	}
	c.Check(foundIt, Equals, true)
}
