package main

import (
	"crypto/md5"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
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

	// reset all variables between tests
	srcConfig = arvadosclient.APIConfig{}
	dstConfig = arvadosclient.APIConfig{}
	blobSigningKey = ""
	srcKeepServicesJSON = ""
	dstKeepServicesJSON = ""
	replications = 0
	prefix = ""
	srcConfigFile = ""
	dstConfigFile = ""
	arvSrc = arvadosclient.ArvadosClient{}
	arvDst = arvadosclient.ArvadosClient{}
	kcSrc = &keepclient.KeepClient{}
	kcDst = &keepclient.KeepClient{}
}

func (s *ServerRequiredSuite) TearDownSuite(c *C) {
	arvadostest.StopKeep()
	arvadostest.StopAPI()
}

var testKeepServicesJSON = "{ \"kind\":\"arvados#keepServiceList\", \"etag\":\"\", \"self_link\":\"\", \"offset\":null, \"limit\":null, \"items\":[ { \"href\":\"/keep_services/zzzzz-bi6l4-123456789012340\", \"kind\":\"arvados#keepService\", \"etag\":\"641234567890enhj7hzx432e5\", \"uuid\":\"zzzzz-bi6l4-123456789012340\", \"owner_uuid\":\"zzzzz-tpzed-123456789012345\", \"service_host\":\"keep0.zzzzz.arvadosapi.com\", \"service_port\":25107, \"service_ssl_flag\":false, \"service_type\":\"disk\", \"read_only\":false }, { \"href\":\"/keep_services/zzzzz-bi6l4-123456789012341\", \"kind\":\"arvados#keepService\", \"etag\":\"641234567890enhj7hzx432e5\", \"uuid\":\"zzzzz-bi6l4-123456789012341\", \"owner_uuid\":\"zzzzz-tpzed-123456789012345\", \"service_host\":\"keep0.zzzzz.arvadosapi.com\", \"service_port\":25108, \"service_ssl_flag\":false, \"service_type\":\"disk\", \"read_only\":false } ], \"items_available\":2 }"

// Testing keep-rsync needs two sets of keep services: src and dst.
// The test setup hence tweaks keep-rsync initialization to achieve this.
// First invoke initializeKeepRsync and then invoke StartKeepWithParams
// to create the keep servers to be used as destination.
func setupRsync(c *C, enforcePermissions bool, setupDstServers bool) {
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
	c.Check(err, IsNil)

	// Create an additional keep server to be used as destination and reload kcDst
	// Set replications to 1 since those many keep servers were created for dst.
	if setupDstServers {
		arvadostest.StartKeepWithParams(true, enforcePermissions)
		replications = 1

		kcDst, err = keepclient.MakeKeepClient(&arvDst)
		c.Check(err, IsNil)
		kcDst.Want_replicas = 1
	}
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
	c.Check(err, IsNil)
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
	c.Check(err, IsNil)
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
	srcKeepServicesJSON = testKeepServicesJSON

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
	c.Check(err, IsNil)
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
	c.Check(err, IsNil)
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

// Put some blocks in Src and some more in Dst
// And copy missing blocks from Src to Dst
func (s *ServerRequiredSuite) TestKeepRsync(c *C) {
	testKeepRsync(c, false, "")
}

// Put some blocks in Src and some more in Dst with blob signing enabled.
// And copy missing blocks from Src to Dst
func (s *ServerRequiredSuite) TestKeepRsync_WithBlobSigning(c *C) {
	testKeepRsync(c, true, "")
}

// Put some blocks in Src and some more in Dst
// Use prefix while doing rsync
// And copy missing blocks from Src to Dst
func (s *ServerRequiredSuite) TestKeepRsync_WithPrefix(c *C) {
	data := []byte("test-data-4")
	hash := fmt.Sprintf("%x", md5.Sum(data))

	testKeepRsync(c, false, hash[0:3])
}

// Put some blocks in Src and some more in Dst
// Use prefix not in src while doing rsync
// And copy missing blocks from Src to Dst
func (s *ServerRequiredSuite) TestKeepRsync_WithNoSuchPrefixInSrc(c *C) {
	testKeepRsync(c, false, "999")
}

// Put 5 blocks in src. Put 2 of those blocks in dst
// Hence there are 3 additional blocks in src
// Also, put 2 extra blocks in dst; they are hence only in dst
// Run rsync and verify that those 7 blocks are now available in dst
func testKeepRsync(c *C, enforcePermissions bool, indexPrefix string) {
	setupRsync(c, enforcePermissions, true)

	prefix = indexPrefix

	// setupTestData
	setupTestData(c, enforcePermissions, prefix)

	err := performKeepRsync()
	c.Check(err, IsNil)

	// Now GetIndex from dst and verify that all 5 from src and the 2 extra blocks are found
	dstIndex, err := getUniqueLocators(kcDst, "")
	c.Check(err, IsNil)

	if prefix == "" {
		for _, locator := range srcLocators {
			_, ok := dstIndex[locator]
			c.Assert(ok, Equals, true)
		}
	} else {
		for _, locator := range srcLocatorsMatchingPrefix {
			_, ok := dstIndex[locator]
			c.Assert(ok, Equals, true)
		}
	}

	for _, locator := range extraDstLocators {
		_, ok := dstIndex[locator]
		c.Assert(ok, Equals, true)
	}

	if prefix == "" {
		// all blocks from src and the two extra blocks
		c.Assert(len(dstIndex), Equals, len(srcLocators)+len(extraDstLocators))
	} else {
		// one matching prefix, 2 that were initially copied into dst along with src, and the extra blocks
		c.Assert(len(dstIndex), Equals, len(srcLocatorsMatchingPrefix)+len(extraDstLocators)+2)
	}
}

// Setup test data in src and dst.
var srcLocators []string
var srcLocatorsMatchingPrefix []string
var dstLocators []string
var extraDstLocators []string

func setupTestData(c *C, enforcePermissions bool, indexPrefix string) {
	srcLocators = []string{}
	srcLocatorsMatchingPrefix = []string{}
	dstLocators = []string{}
	extraDstLocators = []string{}

	tomorrow := time.Now().AddDate(0, 0, 1)

	// Put a few blocks in src using kcSrc
	for i := 0; i < 5; i++ {
		data := []byte(fmt.Sprintf("test-data-%d", i))
		hash := fmt.Sprintf("%x", md5.Sum(data))

		hash2, rep, err := kcSrc.PutB(data)
		c.Check(hash2, Matches, fmt.Sprintf(`^%s\+11(\+.+)?$`, hash))
		c.Check(rep, Equals, 2)
		c.Check(err, IsNil)

		getLocator := hash
		if enforcePermissions {
			getLocator = keepclient.SignLocator(getLocator, arvSrc.ApiToken, tomorrow, []byte(blobSigningKey))
		}

		reader, blocklen, _, err := kcSrc.Get(getLocator)
		c.Check(err, IsNil)
		c.Check(blocklen, Equals, int64(11))
		all, err := ioutil.ReadAll(reader)
		c.Check(all, DeepEquals, data)

		srcLocators = append(srcLocators, fmt.Sprintf("%s+%d", hash, blocklen))
		if strings.HasPrefix(hash, indexPrefix) {
			srcLocatorsMatchingPrefix = append(srcLocatorsMatchingPrefix, fmt.Sprintf("%s+%d", hash, blocklen))
		}
	}

	// Put first two of those src blocks in dst using kcDst
	for i := 0; i < 2; i++ {
		data := []byte(fmt.Sprintf("test-data-%d", i))
		hash := fmt.Sprintf("%x", md5.Sum(data))

		hash2, rep, err := kcDst.PutB(data)
		c.Check(hash2, Matches, fmt.Sprintf(`^%s\+11(\+.+)?$`, hash))
		c.Check(rep, Equals, 1)
		c.Check(err, IsNil)

		getLocator := hash
		if enforcePermissions {
			getLocator = keepclient.SignLocator(getLocator, arvDst.ApiToken, tomorrow, []byte(blobSigningKey))
		}

		reader, blocklen, _, err := kcDst.Get(getLocator)
		c.Check(err, IsNil)
		c.Check(blocklen, Equals, int64(11))
		all, err := ioutil.ReadAll(reader)
		c.Check(all, DeepEquals, data)

		dstLocators = append(dstLocators, fmt.Sprintf("%s+%d", hash, blocklen))
	}

	// Put two more blocks in dst; they are not in src at all
	for i := 0; i < 2; i++ {
		data := []byte(fmt.Sprintf("other-data-%d", i))
		hash := fmt.Sprintf("%x", md5.Sum(data))

		hash2, rep, err := kcDst.PutB(data)
		c.Check(hash2, Matches, fmt.Sprintf(`^%s\+12(\+.+)?$`, hash))
		c.Check(rep, Equals, 1)
		c.Check(err, IsNil)

		getLocator := hash
		if enforcePermissions {
			getLocator = keepclient.SignLocator(getLocator, arvDst.ApiToken, tomorrow, []byte(blobSigningKey))
		}

		reader, blocklen, _, err := kcDst.Get(getLocator)
		c.Check(err, IsNil)
		c.Check(blocklen, Equals, int64(12))
		all, err := ioutil.ReadAll(reader)
		c.Check(all, DeepEquals, data)

		extraDstLocators = append(extraDstLocators, fmt.Sprintf("%s+%d", hash, blocklen))
	}
}

// Setup rsync using srcKeepServicesJSON with fake keepservers.
// Expect error during performKeepRsync due to unreachable src keepservers.
func (s *ServerRequiredSuite) TestErrorDuringRsync_FakeSrcKeepservers(c *C) {
	srcKeepServicesJSON = testKeepServicesJSON

	setupRsync(c, false, false)

	err := performKeepRsync()
	c.Check(strings.HasSuffix(err.Error(), "no such host"), Equals, true)
}

// Setup rsync using dstKeepServicesJSON with fake keepservers.
// Expect error during performKeepRsync due to unreachable dst keepservers.
func (s *ServerRequiredSuite) TestErrorDuringRsync_FakeDstKeepservers(c *C) {
	dstKeepServicesJSON = testKeepServicesJSON

	setupRsync(c, false, false)

	err := performKeepRsync()
	c.Check(strings.HasSuffix(err.Error(), "no such host"), Equals, true)
}

// Test rsync with signature error during Get from src.
func (s *ServerRequiredSuite) TestErrorDuringRsync_ErrorGettingBlockFromSrc(c *C) {
	setupRsync(c, true, true)

	// put some blocks in src and dst
	setupTestData(c, true, "")

	// Change blob signing key to a fake key, so that Get from src fails
	blobSigningKey = "123456789012345678901234yhksjoll2grmku38mi7yxd66h5j4q9w4jzanezacp8s6q0ro3hxakfye02152hncy6zml2ed0uc"

	err := performKeepRsync()
	c.Check(err.Error(), Equals, "Block not found")
}

// Test rsync with error during Put to src.
func (s *ServerRequiredSuite) TestErrorDuringRsync_ErrorPuttingBlockInDst(c *C) {
	setupRsync(c, false, true)

	// put some blocks in src and dst
	setupTestData(c, true, "")

	// Increase Want_replicas on dst to result in insufficient replicas error during Put
	kcDst.Want_replicas = 2

	err := performKeepRsync()
	c.Check(err.Error(), Equals, "Could not write sufficient replicas")
}

// Test loadConfig func
func (s *ServerRequiredSuite) TestLoadConfig(c *C) {
	// Setup a src config file
	srcFile := setupConfigFile(c, "src-config")
	defer os.Remove(srcFile.Name())
	srcConfigFile = srcFile.Name()

	// Setup a dst config file
	dstFile := setupConfigFile(c, "dst-config")
	defer os.Remove(dstFile.Name())
	dstConfigFile = dstFile.Name()

	// load configuration from those files
	err := loadConfig()
	c.Check(err, IsNil)

	c.Assert(srcConfig.APIHost, Equals, "testhost")
	c.Assert(srcConfig.APIToken, Equals, "testtoken")
	c.Assert(srcConfig.APIHostInsecure, Equals, true)
	c.Assert(srcConfig.ExternalClient, Equals, false)

	c.Assert(dstConfig.APIHost, Equals, "testhost")
	c.Assert(dstConfig.APIToken, Equals, "testtoken")
	c.Assert(dstConfig.APIHostInsecure, Equals, true)
	c.Assert(dstConfig.ExternalClient, Equals, false)

	c.Assert(blobSigningKey, Equals, "abcdefg")
}

// Test loadConfig func without setting up the config files
func (s *ServerRequiredSuite) TestLoadConfig_MissingSrcConfig(c *C) {
	srcConfigFile = ""
	err := loadConfig()
	c.Assert(err.Error(), Equals, "-src-config-file must be specified")
}

// Test loadConfig func - error reading src config
func (s *ServerRequiredSuite) TestLoadConfig_ErrorLoadingSrcConfig(c *C) {
	srcConfigFile = "no-such-config-file"

	// load configuration
	err := loadConfig()
	c.Assert(strings.HasSuffix(err.Error(), "no such file or directory"), Equals, true)
}

// Test loadConfig func with no dst config file specified
func (s *ServerRequiredSuite) TestLoadConfig_MissingDstConfig(c *C) {
	// Setup a src config file
	srcFile := setupConfigFile(c, "src-config")
	defer os.Remove(srcFile.Name())
	srcConfigFile = srcFile.Name()

	// But do not setup the dst config file
	dstConfigFile = ""

	// load configuration
	err := loadConfig()
	c.Assert(err.Error(), Equals, "-dst-config-file must be specified")
}

// Test loadConfig func
func (s *ServerRequiredSuite) TestLoadConfig_ErrorLoadingDstConfig(c *C) {
	// Setup a src config file
	srcFile := setupConfigFile(c, "src-config")
	defer os.Remove(srcFile.Name())
	srcConfigFile = srcFile.Name()

	// But do not setup the dst config file
	dstConfigFile = "no-such-config-file"

	// load configuration
	err := loadConfig()
	c.Assert(strings.HasSuffix(err.Error(), "no such file or directory"), Equals, true)
}

func setupConfigFile(c *C, name string) *os.File {
	// Setup a config file
	file, err := ioutil.TempFile(os.TempDir(), name)
	c.Check(err, IsNil)

	fileContent := "ARVADOS_API_HOST=testhost\n"
	fileContent += "ARVADOS_API_TOKEN=testtoken\n"
	fileContent += "ARVADOS_API_HOST_INSECURE=true\n"
	fileContent += "ARVADOS_BLOB_SIGNING_KEY=abcdefg"

	_, err = file.Write([]byte(fileContent))
	c.Check(err, IsNil)

	return file
}
