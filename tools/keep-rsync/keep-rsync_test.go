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

var kcSrc, kcDst *keepclient.KeepClient
var srcKeepServicesJSON, dstKeepServicesJSON, blobSigningKey string

func (s *ServerRequiredSuite) SetUpTest(c *C) {
	arvadostest.ResetEnv()

	// reset all variables between tests
	blobSigningKey = ""
	srcKeepServicesJSON = ""
	dstKeepServicesJSON = ""
	kcSrc = &keepclient.KeepClient{}
	kcDst = &keepclient.KeepClient{}
}

func (s *ServerRequiredSuite) TearDownSuite(c *C) {
	arvadostest.StopKeep()
	arvadostest.StopAPI()
}

var testKeepServicesJSON = "{ \"kind\":\"arvados#keepServiceList\", \"etag\":\"\", \"self_link\":\"\", \"offset\":null, \"limit\":null, \"items\":[ { \"href\":\"/keep_services/zzzzz-bi6l4-123456789012340\", \"kind\":\"arvados#keepService\", \"etag\":\"641234567890enhj7hzx432e5\", \"uuid\":\"zzzzz-bi6l4-123456789012340\", \"owner_uuid\":\"zzzzz-tpzed-123456789012345\", \"service_host\":\"keep0.zzzzz.arvadosapi.com\", \"service_port\":25107, \"service_ssl_flag\":false, \"service_type\":\"disk\", \"read_only\":false }, { \"href\":\"/keep_services/zzzzz-bi6l4-123456789012341\", \"kind\":\"arvados#keepService\", \"etag\":\"641234567890enhj7hzx432e5\", \"uuid\":\"zzzzz-bi6l4-123456789012341\", \"owner_uuid\":\"zzzzz-tpzed-123456789012345\", \"service_host\":\"keep0.zzzzz.arvadosapi.com\", \"service_port\":25108, \"service_ssl_flag\":false, \"service_type\":\"disk\", \"read_only\":false } ], \"items_available\":2 }"

// Testing keep-rsync needs two sets of keep services: src and dst.
// The test setup hence creates 3 servers instead of the default 2,
// and uses the first 2 as src and the 3rd as dst keep servers.
func setupRsync(c *C, enforcePermissions, updateDstReplications bool, replications int) {
	// srcConfig
	var srcConfig arvadosclient.APIConfig
	srcConfig.APIHost = os.Getenv("ARVADOS_API_HOST")
	srcConfig.APIToken = os.Getenv("ARVADOS_API_TOKEN")
	srcConfig.APIHostInsecure = matchTrue.MatchString(os.Getenv("ARVADOS_API_HOST_INSECURE"))

	// dstConfig
	var dstConfig arvadosclient.APIConfig
	dstConfig.APIHost = os.Getenv("ARVADOS_API_HOST")
	dstConfig.APIToken = os.Getenv("ARVADOS_API_TOKEN")
	dstConfig.APIHostInsecure = matchTrue.MatchString(os.Getenv("ARVADOS_API_HOST_INSECURE"))

	if enforcePermissions {
		blobSigningKey = "zfhgfenhffzltr9dixws36j1yhksjoll2grmku38mi7yxd66h5j4q9w4jzanezacp8s6q0ro3hxakfye02152hncy6zml2ed0uc"
	}

	// Start API and Keep servers
	arvadostest.StartAPI()
	arvadostest.StartKeepWithParams(true, enforcePermissions)

	// setup keepclients
	var err error
	kcSrc, kcDst, err = setupKeepClients(srcConfig, dstConfig, srcKeepServicesJSON, dstKeepServicesJSON, replications)
	c.Check(err, IsNil)

	for uuid := range kcSrc.LocalRoots() {
		if strings.HasSuffix(uuid, "02") {
			delete(kcSrc.LocalRoots(), uuid)
		}
	}
	for uuid := range kcSrc.GatewayRoots() {
		if strings.HasSuffix(uuid, "02") {
			delete(kcSrc.GatewayRoots(), uuid)
		}
	}
	for uuid := range kcSrc.WritableLocalRoots() {
		if strings.HasSuffix(uuid, "02") {
			delete(kcSrc.WritableLocalRoots(), uuid)
		}
	}

	for uuid := range kcDst.LocalRoots() {
		if strings.HasSuffix(uuid, "00") || strings.HasSuffix(uuid, "01") {
			delete(kcDst.LocalRoots(), uuid)
		}
	}
	for uuid := range kcDst.GatewayRoots() {
		if strings.HasSuffix(uuid, "00") || strings.HasSuffix(uuid, "01") {
			delete(kcDst.GatewayRoots(), uuid)
		}
	}
	for uuid := range kcDst.WritableLocalRoots() {
		if strings.HasSuffix(uuid, "00") || strings.HasSuffix(uuid, "01") {
			delete(kcDst.WritableLocalRoots(), uuid)
		}
	}

	if updateDstReplications {
		kcDst.Want_replicas = replications
	}
}

// Test keep-rsync initialization, with src and dst keep servers.
// Do a Put and Get in src, both of which should succeed.
// Do a Put and Get in dst, both of which should succeed.
// Do a Get in dst for the src hash, which should raise block not found error.
// Do a Get in src for the dst hash, which should raise block not found error.
func (s *ServerRequiredSuite) TestRsyncPutInOne_GetFromOtherShouldFail(c *C) {
	setupRsync(c, false, true, 1)

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

	setupRsync(c, false, true, 1)

	localRoots := kcSrc.LocalRoots()
	c.Check(localRoots, NotNil)

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
	setupRsync(c, true, true, 1)

	// Put a block in src using kcSrc and Get it
	srcData := []byte("test-data1")
	locatorInSrc := fmt.Sprintf("%x", md5.Sum(srcData))

	hash, rep, err := kcSrc.PutB(srcData)
	c.Check(hash, Matches, fmt.Sprintf(`^%s\+10(\+.+)?$`, locatorInSrc))
	c.Check(rep, Equals, 2)
	c.Check(err, Equals, nil)

	tomorrow := time.Now().AddDate(0, 0, 1)
	signedLocator := keepclient.SignLocator(locatorInSrc, kcSrc.Arvados.ApiToken, tomorrow, []byte(blobSigningKey))

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

	signedLocator = keepclient.SignLocator(locatorInDst, kcDst.Arvados.ApiToken, tomorrow, []byte(blobSigningKey))

	reader, blocklen, _, err = kcDst.Get(signedLocator)
	c.Check(err, IsNil)
	c.Check(blocklen, Equals, int64(10))
	all, err = ioutil.ReadAll(reader)
	c.Check(all, DeepEquals, dstData)

	// Get srcLocator using kcDst should fail with Not Found error
	signedLocator = keepclient.SignLocator(locatorInSrc, kcDst.Arvados.ApiToken, tomorrow, []byte(blobSigningKey))
	_, _, _, err = kcDst.Get(locatorInSrc)
	c.Assert(err.Error(), Equals, "Block not found")

	// Get dstLocator using kcSrc should fail with Not Found error
	signedLocator = keepclient.SignLocator(locatorInDst, kcSrc.Arvados.ApiToken, tomorrow, []byte(blobSigningKey))
	_, _, _, err = kcSrc.Get(locatorInDst)
	c.Assert(err.Error(), Equals, "Block not found")
}

// Test keep-rsync initialization with default replications count
func (s *ServerRequiredSuite) TestInitializeRsyncDefaultReplicationsCount(c *C) {
	setupRsync(c, false, false, 0)

	// Must have got default replications value as 2 from dst discovery document
	c.Assert(kcDst.Want_replicas, Equals, 2)
}

// Test keep-rsync initialization with replications count argument
func (s *ServerRequiredSuite) TestInitializeRsyncReplicationsCount(c *C) {
	setupRsync(c, false, false, 3)

	// Since replications value is provided, default is not used
	c.Assert(kcDst.Want_replicas, Equals, 3)
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
func testKeepRsync(c *C, enforcePermissions bool, prefix string) {
	setupRsync(c, enforcePermissions, true, 1)

	// setupTestData
	setupTestData(c, prefix)

	err := performKeepRsync(kcSrc, kcDst, blobSigningKey, prefix)
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
var srcLocators, srcLocatorsMatchingPrefix, dstLocators, extraDstLocators []string

func setupTestData(c *C, indexPrefix string) {
	srcLocators = []string{}
	srcLocatorsMatchingPrefix = []string{}
	dstLocators = []string{}
	extraDstLocators = []string{}

	// Put a few blocks in src using kcSrc
	for i := 0; i < 5; i++ {
		hash, _, err := kcSrc.PutB([]byte(fmt.Sprintf("test-data-%d", i)))
		c.Check(err, IsNil)

		srcLocators = append(srcLocators, strings.Split(hash, "+A")[0])
		if strings.HasPrefix(hash, indexPrefix) {
			srcLocatorsMatchingPrefix = append(srcLocatorsMatchingPrefix, strings.Split(hash, "+A")[0])
		}
	}

	// Put first two of those src blocks in dst using kcDst
	for i := 0; i < 2; i++ {
		hash, _, err := kcDst.PutB([]byte(fmt.Sprintf("test-data-%d", i)))
		c.Check(err, IsNil)
		dstLocators = append(dstLocators, strings.Split(hash, "+A")[0])
	}

	// Put two more blocks in dst; they are not in src at all
	for i := 0; i < 2; i++ {
		hash, _, err := kcDst.PutB([]byte(fmt.Sprintf("other-data-%d", i)))
		c.Check(err, IsNil)
		dstLocators = append(dstLocators, strings.Split(hash, "+A")[0])
		extraDstLocators = append(extraDstLocators, strings.Split(hash, "+A")[0])
	}
}

// Setup rsync using srcKeepServicesJSON with fake keepservers.
// Expect error during performKeepRsync due to unreachable src keepservers.
func (s *ServerRequiredSuite) TestErrorDuringRsync_FakeSrcKeepservers(c *C) {
	srcKeepServicesJSON = testKeepServicesJSON

	setupRsync(c, false, false, 1)

	err := performKeepRsync(kcSrc, kcDst, "", "")
	c.Check(strings.HasSuffix(err.Error(), "no such host"), Equals, true)
}

// Setup rsync using dstKeepServicesJSON with fake keepservers.
// Expect error during performKeepRsync due to unreachable dst keepservers.
func (s *ServerRequiredSuite) TestErrorDuringRsync_FakeDstKeepservers(c *C) {
	dstKeepServicesJSON = testKeepServicesJSON

	setupRsync(c, false, false, 1)

	err := performKeepRsync(kcSrc, kcDst, "", "")
	c.Check(strings.HasSuffix(err.Error(), "no such host"), Equals, true)
}

// Test rsync with signature error during Get from src.
func (s *ServerRequiredSuite) TestErrorDuringRsync_ErrorGettingBlockFromSrc(c *C) {
	setupRsync(c, true, true, 1)

	// put some blocks in src and dst
	setupTestData(c, "")

	// Change blob signing key to a fake key, so that Get from src fails
	blobSigningKey = "thisisfakeblobsigningkey"

	err := performKeepRsync(kcSrc, kcDst, blobSigningKey, "")
	c.Check(strings.HasSuffix(err.Error(), "Block not found"), Equals, true)
}

// Test rsync with error during Put to src.
func (s *ServerRequiredSuite) TestErrorDuringRsync_ErrorPuttingBlockInDst(c *C) {
	setupRsync(c, false, true, 1)

	// put some blocks in src and dst
	setupTestData(c, "")

	// Increase Want_replicas on dst to result in insufficient replicas error during Put
	kcDst.Want_replicas = 2

	err := performKeepRsync(kcSrc, kcDst, blobSigningKey, "")
	c.Check(strings.HasSuffix(err.Error(), "Could not write sufficient replicas"), Equals, true)
}

// Test loadConfig func
func (s *ServerRequiredSuite) TestLoadConfig(c *C) {
	// Setup a src config file
	srcFile := setupConfigFile(c, "src-config")
	defer os.Remove(srcFile.Name())
	srcConfigFile := srcFile.Name()

	// Setup a dst config file
	dstFile := setupConfigFile(c, "dst-config")
	defer os.Remove(dstFile.Name())
	dstConfigFile := dstFile.Name()

	// load configuration from those files
	srcConfig, dstConfig, srcBlobSigningKey, _, err := loadConfig(srcConfigFile, dstConfigFile)
	c.Check(err, IsNil)

	c.Assert(srcConfig.APIHost, Equals, "testhost")
	c.Assert(srcConfig.APIToken, Equals, "testtoken")
	c.Assert(srcConfig.APIHostInsecure, Equals, true)
	c.Assert(srcConfig.ExternalClient, Equals, false)

	c.Assert(dstConfig.APIHost, Equals, "testhost")
	c.Assert(dstConfig.APIToken, Equals, "testtoken")
	c.Assert(dstConfig.APIHostInsecure, Equals, true)
	c.Assert(dstConfig.ExternalClient, Equals, false)

	c.Assert(srcBlobSigningKey, Equals, "abcdefg")
}

// Test loadConfig func without setting up the config files
func (s *ServerRequiredSuite) TestLoadConfig_MissingSrcConfig(c *C) {
	_, _, _, _, err := loadConfig("", "")
	c.Assert(err.Error(), Equals, "-src-config-file must be specified")
}

// Test loadConfig func - error reading src config
func (s *ServerRequiredSuite) TestLoadConfig_ErrorLoadingSrcConfig(c *C) {
	_, _, _, _, err := loadConfig("no-such-config-file", "")
	c.Assert(strings.HasSuffix(err.Error(), "no such file or directory"), Equals, true)
}

// Test loadConfig func with no dst config file specified
func (s *ServerRequiredSuite) TestLoadConfig_MissingDstConfig(c *C) {
	// Setup a src config file
	srcFile := setupConfigFile(c, "src-config")
	defer os.Remove(srcFile.Name())
	srcConfigFile := srcFile.Name()

	// load configuration
	_, _, _, _, err := loadConfig(srcConfigFile, "")
	c.Assert(err.Error(), Equals, "-dst-config-file must be specified")
}

// Test loadConfig func
func (s *ServerRequiredSuite) TestLoadConfig_ErrorLoadingDstConfig(c *C) {
	// Setup a src config file
	srcFile := setupConfigFile(c, "src-config")
	defer os.Remove(srcFile.Name())
	srcConfigFile := srcFile.Name()

	// load configuration
	_, _, _, _, err := loadConfig(srcConfigFile, "no-such-config-file")
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
