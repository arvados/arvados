package main

import (
	"crypto/md5"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

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
// The test setup hence tweaks keep-rsync initialization to achieve this.
// First invoke initializeKeepRsync and then invoke StartKeepWithParams
// to create the keep servers to be used as destination.
func setupRsync(c *C, enforcePermissions bool, overwriteReplications bool) {
	// srcConfig
	srcConfig.APIHost = os.Getenv("ARVADOS_API_HOST")
	srcConfig.APIToken = os.Getenv("ARVADOS_API_TOKEN")
	srcConfig.APIHostInsecure = matchTrue.MatchString(os.Getenv("ARVADOS_API_HOST_INSECURE"))

	// dstConfig
	dstConfig.APIHost = os.Getenv("ARVADOS_API_HOST")
	dstConfig.APIToken = os.Getenv("ARVADOS_API_TOKEN")
	dstConfig.APIHostInsecure = matchTrue.MatchString(os.Getenv("ARVADOS_API_HOST_INSECURE"))

	if enforcePermissions {
		blobSigningKey = "zfhgfenhffzltr9dixws36j1yhksjoll2grmku38mi7yxd66h5j4q9w4jzanezacp8s6q0ro3hxakfye02152hncy6zml2ed0uc"
	}

	// Start API and Keep servers
	arvadostest.StartAPI()
	arvadostest.StartKeepWithParams(false, enforcePermissions)

	// initialize keep-rsync
	err := initializeKeepRsync()
	c.Assert(err, Equals, nil)

	// Create two more keep servers to be used as destination
	arvadostest.StartKeepWithParams(true, enforcePermissions)

	// set replications to 1 since those many keep servers were created for dst.
	if overwriteReplications {
		replications = 1
	}

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
	fileContent += "ARVADOS_API_HOST_INSECURE=true\n"
	fileContent += "ARVADOS_BLOB_SIGNING_KEY=abcdefg"

	_, err = file.Write([]byte(fileContent))

	// Invoke readConfigFromFile method with this test filename
	config, err := readConfigFromFile(file.Name())
	c.Assert(err, Equals, nil)
	c.Assert(config.APIHost, Equals, "testhost")
	c.Assert(config.APIToken, Equals, "testtoken")
	c.Assert(config.APIHostInsecure, Equals, true)
	c.Assert(config.ExternalClient, Equals, false)
	c.Assert(blobSigningKey, Equals, "abcdefg")
}

// Test keep-rsync initialization, with src and dst keep servers.
// Do a Put and Get in src, both of which should succeed.
// Do a Put and Get in dst, both of which should succeed.
// Do a Get in dst for the src hash, which should raise block not found error.
// Do a Get in src for the dst hash, which should raise block not found error.
func (s *ServerRequiredSuite) TestRsyncPutInOne_GetFromOtherShouldFail(c *C) {
	setupRsync(c, false, true)

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

	// Get srcLocator using kcDst should fail with Not Found error
	_, _, _, err = kcDst.Get(locatorInSrc)
	c.Assert(err.Error(), Equals, "Block not found")

	// Get dstLocator using kcSrc should fail with Not Found error
	_, _, _, err = kcSrc.Get(locatorInDst)
	c.Assert(err.Error(), Equals, "Block not found")
}

// Test keep-rsync initialization, with srcKeepServicesJSON
func (s *ServerRequiredSuite) TestRsyncInitializeWithKeepServicesJSON(c *C) {
	srcKeepServicesJSON = "{ \"kind\":\"arvados#keepServiceList\", \"etag\":\"\", \"self_link\":\"\", \"offset\":null, \"limit\":null, \"items\":[ { \"href\":\"/keep_services/zzzzz-bi6l4-123456789012340\", \"kind\":\"arvados#keepService\", \"etag\":\"641234567890enhj7hzx432e5\", \"uuid\":\"zzzzz-bi6l4-123456789012340\", \"owner_uuid\":\"zzzzz-tpzed-123456789012345\", \"service_host\":\"keep0.zzzzz.arvadosapi.com\", \"service_port\":25107, \"service_ssl_flag\":false, \"service_type\":\"disk\", \"read_only\":false }, { \"href\":\"/keep_services/zzzzz-bi6l4-123456789012341\", \"kind\":\"arvados#keepService\", \"etag\":\"641234567890enhj7hzx432e5\", \"uuid\":\"zzzzz-bi6l4-123456789012341\", \"owner_uuid\":\"zzzzz-tpzed-123456789012345\", \"service_host\":\"keep0.zzzzz.arvadosapi.com\", \"service_port\":25108, \"service_ssl_flag\":false, \"service_type\":\"disk\", \"read_only\":false } ], \"items_available\":2 }"

	setupRsync(c, false, true)

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

// Test keep-rsync initialization, with src and dst keep servers with blobSigningKey.
// Do a Put and Get in src, both of which should succeed.
// Do a Put and Get in dst, both of which should succeed.
// Do a Get in dst for the src hash, which should raise block not found error.
// Do a Get in src for the dst hash, which should raise block not found error.
func (s *ServerRequiredSuite) TestRsyncWithBlobSigning_PutInOne_GetFromOtherShouldFail(c *C) {
	setupRsync(c, true, true)

	// Put a block in src using kcSrc and Get it
	srcData := []byte("test-data1")
	locatorInSrc := fmt.Sprintf("%x", md5.Sum(srcData))

	hash, rep, err := kcSrc.PutB(srcData)
	c.Check(hash, Matches, fmt.Sprintf(`^%s\+10(\+.+)?$`, locatorInSrc))
	c.Check(rep, Equals, 2)
	c.Check(err, Equals, nil)

	tomorrow := time.Now().AddDate(0, 0, 1)
	signedLocator := keepclient.SignLocator(locatorInSrc, arvSrc.ApiToken, tomorrow, []byte(blobSigningKey))

	reader, blocklen, _, err := kcSrc.Get(signedLocator)
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

	signedLocator = keepclient.SignLocator(locatorInDst, arvDst.ApiToken, tomorrow, []byte(blobSigningKey))

	reader, blocklen, _, err = kcDst.Get(signedLocator)
	c.Assert(err, Equals, nil)
	c.Check(blocklen, Equals, int64(10))
	all, err = ioutil.ReadAll(reader)
	c.Check(all, DeepEquals, dstData)

	// Get srcLocator using kcDst should fail with Not Found error
	signedLocator = keepclient.SignLocator(locatorInSrc, arvDst.ApiToken, tomorrow, []byte(blobSigningKey))
	_, _, _, err = kcDst.Get(locatorInSrc)
	c.Assert(err.Error(), Equals, "Block not found")

	// Get dstLocator using kcSrc should fail with Not Found error
	signedLocator = keepclient.SignLocator(locatorInDst, arvSrc.ApiToken, tomorrow, []byte(blobSigningKey))
	_, _, _, err = kcSrc.Get(locatorInDst)
	c.Assert(err.Error(), Equals, "Block not found")
}

// Test keep-rsync initialization with default replications count
func (s *ServerRequiredSuite) TestInitializeRsyncDefaultReplicationsCount(c *C) {
	setupRsync(c, false, false)

	// Must have got default replications value as 2 from dst discovery document
	c.Assert(replications, Equals, 2)
}

// Test keep-rsync initialization with replications count argument
func (s *ServerRequiredSuite) TestInitializeRsyncReplicationsCount(c *C) {
	// First set replications to 3 to mimic passing input argument
	replications = 3

	setupRsync(c, false, false)

	// Since replications value is provided, default is not used
	c.Assert(replications, Equals, 3)
}
