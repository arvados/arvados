// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"crypto/md5"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvadosclient"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"git.arvados.org/arvados.git/sdk/go/keepclient"

	. "gopkg.in/check.v1"
)

var kcSrc, kcDst *keepclient.KeepClient
var srcKeepServicesJSON, dstKeepServicesJSON, blobSigningKey string
var blobSignatureTTL = time.Duration(2*7*24) * time.Hour

func resetGlobals() {
	blobSigningKey = ""
	srcKeepServicesJSON = ""
	dstKeepServicesJSON = ""
	kcSrc = nil
	kcDst = nil
}

// Gocheck boilerplate
func Test(t *testing.T) {
	TestingT(t)
}

var _ = Suite(&ServerRequiredSuite{})
var _ = Suite(&ServerNotRequiredSuite{})
var _ = Suite(&DoMainTestSuite{})

type ServerRequiredSuite struct{}

func (s *ServerRequiredSuite) TearDownSuite(c *C) {
	arvadostest.ResetEnv()
}

func (s *ServerRequiredSuite) SetUpTest(c *C) {
	resetGlobals()
}

func (s *ServerRequiredSuite) TearDownTest(c *C) {
	arvadostest.StopKeep(3)
}

func (s *ServerNotRequiredSuite) SetUpTest(c *C) {
	resetGlobals()
}

type ServerNotRequiredSuite struct{}

type DoMainTestSuite struct {
	initialArgs []string
}

func (s *DoMainTestSuite) SetUpTest(c *C) {
	s.initialArgs = os.Args
	os.Args = []string{"keep-rsync"}
	resetGlobals()
}

func (s *DoMainTestSuite) TearDownTest(c *C) {
	os.Args = s.initialArgs
}

var testKeepServicesJSON = "{ \"kind\":\"arvados#keepServiceList\", \"etag\":\"\", \"self_link\":\"\", \"offset\":null, \"limit\":null, \"items\":[ { \"href\":\"/keep_services/zzzzz-bi6l4-123456789012340\", \"kind\":\"arvados#keepService\", \"etag\":\"641234567890enhj7hzx432e5\", \"uuid\":\"zzzzz-bi6l4-123456789012340\", \"owner_uuid\":\"zzzzz-tpzed-123456789012345\", \"service_host\":\"keep0.zzzzz.arvadosapi.com\", \"service_port\":25107, \"service_ssl_flag\":false, \"service_type\":\"disk\", \"read_only\":false }, { \"href\":\"/keep_services/zzzzz-bi6l4-123456789012341\", \"kind\":\"arvados#keepService\", \"etag\":\"641234567890enhj7hzx432e5\", \"uuid\":\"zzzzz-bi6l4-123456789012341\", \"owner_uuid\":\"zzzzz-tpzed-123456789012345\", \"service_host\":\"keep0.zzzzz.arvadosapi.com\", \"service_port\":25108, \"service_ssl_flag\":false, \"service_type\":\"disk\", \"read_only\":false } ], \"items_available\":2 }"

// Testing keep-rsync needs two sets of keep services: src and dst.
// The test setup hence creates 3 servers instead of the default 2,
// and uses the first 2 as src and the 3rd as dst keep servers.
func setupRsync(c *C, enforcePermissions bool, replications int) {
	// srcConfig
	var srcConfig apiConfig
	srcConfig.APIHost = os.Getenv("ARVADOS_API_HOST")
	srcConfig.APIToken = arvadostest.SystemRootToken
	srcConfig.APIHostInsecure = arvadosclient.StringBool(os.Getenv("ARVADOS_API_HOST_INSECURE"))

	// dstConfig
	var dstConfig apiConfig
	dstConfig.APIHost = os.Getenv("ARVADOS_API_HOST")
	dstConfig.APIToken = arvadostest.SystemRootToken
	dstConfig.APIHostInsecure = arvadosclient.StringBool(os.Getenv("ARVADOS_API_HOST_INSECURE"))

	if enforcePermissions {
		blobSigningKey = arvadostest.BlobSigningKey
	}

	// Start Keep servers
	arvadostest.StartKeep(3, enforcePermissions)
	keepclient.RefreshServiceDiscovery()

	// setup keepclients
	var err error
	kcSrc, _, err = setupKeepClient(srcConfig, srcKeepServicesJSON, false, 0, blobSignatureTTL)
	c.Assert(err, IsNil)

	kcDst, _, err = setupKeepClient(dstConfig, dstKeepServicesJSON, true, replications, 0)
	c.Assert(err, IsNil)

	srcRoots := map[string]string{}
	dstRoots := map[string]string{}
	for uuid, root := range kcSrc.LocalRoots() {
		if strings.HasSuffix(uuid, "02") {
			dstRoots[uuid] = root
		} else {
			srcRoots[uuid] = root
		}
	}
	if srcKeepServicesJSON == "" {
		kcSrc.SetServiceRoots(srcRoots, srcRoots, srcRoots)
	}
	if dstKeepServicesJSON == "" {
		kcDst.SetServiceRoots(dstRoots, dstRoots, dstRoots)
	}

	if replications == 0 {
		// Must have got default replications value of 2 from dst discovery document
		c.Assert(kcDst.Want_replicas, Equals, 2)
	} else {
		// Since replications value is provided, it is used
		c.Assert(kcDst.Want_replicas, Equals, replications)
	}
}

func (s *ServerRequiredSuite) TestRsyncPutInOne_GetFromOtherShouldFail(c *C) {
	setupRsync(c, false, 1)

	// Put a block in src and verify that it is not found in dst
	testNoCrosstalk(c, "test-data-1", kcSrc, kcDst)

	// Put a block in dst and verify that it is not found in src
	testNoCrosstalk(c, "test-data-2", kcDst, kcSrc)
}

func (s *ServerRequiredSuite) TestRsyncWithBlobSigning_PutInOne_GetFromOtherShouldFail(c *C) {
	setupRsync(c, true, 1)

	// Put a block in src and verify that it is not found in dst
	testNoCrosstalk(c, "test-data-1", kcSrc, kcDst)

	// Put a block in dst and verify that it is not found in src
	testNoCrosstalk(c, "test-data-2", kcDst, kcSrc)
}

// Do a Put in the first and Get from the second,
// which should raise block not found error.
func testNoCrosstalk(c *C, testData string, kc1, kc2 *keepclient.KeepClient) {
	// Put a block using kc1
	locator, _, err := kc1.PutB([]byte(testData))
	c.Assert(err, Equals, nil)

	locator = strings.Join(strings.Split(locator, "+")[:2], "+")
	_, _, _, err = kc2.Get(keepclient.SignLocator(locator, kc2.Arvados.ApiToken, time.Now().AddDate(0, 0, 1), blobSignatureTTL, []byte(blobSigningKey)))
	c.Assert(err, NotNil)
	c.Check(err.Error(), Equals, "Block not found")
}

// Test keep-rsync initialization, with srcKeepServicesJSON
func (s *ServerRequiredSuite) TestRsyncInitializeWithKeepServicesJSON(c *C) {
	srcKeepServicesJSON = testKeepServicesJSON

	setupRsync(c, false, 1)

	localRoots := kcSrc.LocalRoots()
	c.Check(localRoots, NotNil)
	c.Check(localRoots["zzzzz-bi6l4-123456789012340"], Not(Equals), "")
	c.Check(localRoots["zzzzz-bi6l4-123456789012341"], Not(Equals), "")
}

// Test keep-rsync initialization with default replications count
func (s *ServerRequiredSuite) TestInitializeRsyncDefaultReplicationsCount(c *C) {
	setupRsync(c, false, 0)
}

// Test keep-rsync initialization with replications count argument
func (s *ServerRequiredSuite) TestInitializeRsyncReplicationsCount(c *C) {
	setupRsync(c, false, 3)
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
	c.Check(len(dstIndex) > len(dstLocators), Equals, true)
}

// Put some blocks in Src and some more in Dst
// Use prefix not in src while doing rsync
// And copy missing blocks from Src to Dst
func (s *ServerRequiredSuite) TestKeepRsync_WithNoSuchPrefixInSrc(c *C) {
	testKeepRsync(c, false, "999")
	c.Check(len(dstIndex), Equals, len(dstLocators))
}

// Put 5 blocks in src. Put 2 of those blocks in dst
// Hence there are 3 additional blocks in src
// Also, put 2 extra blocks in dst; they are hence only in dst
// Run rsync and verify that those 7 blocks are now available in dst
func testKeepRsync(c *C, enforcePermissions bool, prefix string) {
	setupRsync(c, enforcePermissions, 1)

	// setupTestData
	setupTestData(c, prefix)

	err := performKeepRsync(kcSrc, kcDst, blobSignatureTTL, blobSigningKey, prefix)
	c.Check(err, IsNil)

	// Now GetIndex from dst and verify that all 5 from src and the 2 extra blocks are found
	dstIndex, err = getUniqueLocators(kcDst, "")
	c.Check(err, IsNil)

	for _, locator := range srcLocatorsMatchingPrefix {
		_, ok := dstIndex[locator]
		c.Assert(ok, Equals, true)
	}

	for _, locator := range extraDstLocators {
		_, ok := dstIndex[locator]
		c.Assert(ok, Equals, true)
	}

	if prefix == "" {
		// all blocks from src and the two extra blocks
		c.Assert(len(dstIndex), Equals, len(srcLocators)+len(extraDstLocators))
	} else {
		// 1 matching prefix and copied over, 2 that were initially copied into dst along with src, and the 2 extra blocks
		c.Assert(len(dstIndex), Equals, len(srcLocatorsMatchingPrefix)+len(extraDstLocators)+2)
	}
}

// Setup test data in src and dst.
var srcLocators, srcLocatorsMatchingPrefix, dstLocators, extraDstLocators []string
var dstIndex map[string]bool

func setupTestData(c *C, indexPrefix string) {
	srcLocators = []string{}
	srcLocatorsMatchingPrefix = []string{}
	dstLocators = []string{}
	extraDstLocators = []string{}
	dstIndex = make(map[string]bool)

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

	setupRsync(c, false, 1)

	err := performKeepRsync(kcSrc, kcDst, blobSignatureTTL, "", "")
	c.Assert(err, NotNil)
	c.Check(err.Error(), Matches, ".*no such host.*")
}

// Setup rsync using dstKeepServicesJSON with fake keepservers.
// Expect error during performKeepRsync due to unreachable dst keepservers.
func (s *ServerRequiredSuite) TestErrorDuringRsync_FakeDstKeepservers(c *C) {
	dstKeepServicesJSON = testKeepServicesJSON

	setupRsync(c, false, 1)

	err := performKeepRsync(kcSrc, kcDst, blobSignatureTTL, "", "")
	c.Assert(err, NotNil)
	c.Check(err.Error(), Matches, ".*no such host.*")
}

// Test rsync with signature error during Get from src.
func (s *ServerRequiredSuite) TestErrorDuringRsync_ErrorGettingBlockFromSrc(c *C) {
	setupRsync(c, true, 1)

	// put some blocks in src and dst
	setupTestData(c, "")

	// Change blob signing key to a fake key, so that Get from src fails
	blobSigningKey = "thisisfakeblobsigningkey"

	err := performKeepRsync(kcSrc, kcDst, blobSignatureTTL, blobSigningKey, "")
	c.Assert(err, NotNil)
	c.Check(err.Error(), Matches, ".*HTTP 400 \"invalid signature\".*")
}

// Test rsync with error during Put to src.
func (s *ServerRequiredSuite) TestErrorDuringRsync_ErrorPuttingBlockInDst(c *C) {
	setupRsync(c, false, 1)

	// put some blocks in src and dst
	setupTestData(c, "")

	// Increase Want_replicas on dst to result in insufficient replicas error during Put
	kcDst.Want_replicas = 2

	err := performKeepRsync(kcSrc, kcDst, blobSignatureTTL, blobSigningKey, "")
	c.Assert(err, NotNil)
	c.Check(err.Error(), Matches, ".*Could not write sufficient replicas.*")
}

// Test loadConfig func
func (s *ServerNotRequiredSuite) TestLoadConfig(c *C) {
	// Setup a src config file
	srcFile := setupConfigFile(c, "src-config")
	defer os.Remove(srcFile.Name())
	srcConfigFile := srcFile.Name()

	// Setup a dst config file
	dstFile := setupConfigFile(c, "dst-config")
	defer os.Remove(dstFile.Name())
	dstConfigFile := dstFile.Name()

	// load configuration from those files
	srcConfig, srcBlobSigningKey, err := loadConfig(srcConfigFile)
	c.Check(err, IsNil)

	c.Assert(srcConfig.APIHost, Equals, os.Getenv("ARVADOS_API_HOST"))
	c.Assert(srcConfig.APIToken, Equals, arvadostest.SystemRootToken)
	c.Assert(srcConfig.APIHostInsecure, Equals, arvadosclient.StringBool(os.Getenv("ARVADOS_API_HOST_INSECURE")))

	dstConfig, _, err := loadConfig(dstConfigFile)
	c.Check(err, IsNil)

	c.Assert(dstConfig.APIHost, Equals, os.Getenv("ARVADOS_API_HOST"))
	c.Assert(dstConfig.APIToken, Equals, arvadostest.SystemRootToken)
	c.Assert(dstConfig.APIHostInsecure, Equals, arvadosclient.StringBool(os.Getenv("ARVADOS_API_HOST_INSECURE")))

	c.Assert(srcBlobSigningKey, Equals, "abcdefg")
}

// Test loadConfig func without setting up the config files
func (s *ServerNotRequiredSuite) TestLoadConfig_MissingSrcConfig(c *C) {
	_, _, err := loadConfig("")
	c.Assert(err.Error(), Equals, "config file not specified")
}

// Test loadConfig func - error reading config
func (s *ServerNotRequiredSuite) TestLoadConfig_ErrorLoadingSrcConfig(c *C) {
	_, _, err := loadConfig("no-such-config-file")
	c.Assert(err, NotNil)
	c.Check(err.Error(), Matches, ".*no such file or directory.*")
}

func (s *ServerNotRequiredSuite) TestSetupKeepClient_NoBlobSignatureTTL(c *C) {
	var srcConfig apiConfig
	srcConfig.APIHost = os.Getenv("ARVADOS_API_HOST")
	srcConfig.APIToken = arvadostest.SystemRootToken
	srcConfig.APIHostInsecure = arvadosclient.StringBool(os.Getenv("ARVADOS_API_HOST_INSECURE"))

	_, ttl, err := setupKeepClient(srcConfig, srcKeepServicesJSON, false, 0, 0)
	c.Check(err, IsNil)
	c.Assert(ttl, Equals, blobSignatureTTL)
}

func setupConfigFile(c *C, name string) *os.File {
	// Setup a config file
	file, err := ioutil.TempFile(os.TempDir(), name)
	c.Check(err, IsNil)

	fileContent := "ARVADOS_API_HOST=" + os.Getenv("ARVADOS_API_HOST") + "\n"
	fileContent += "ARVADOS_API_TOKEN=" + arvadostest.SystemRootToken + "\n"
	fileContent += "ARVADOS_API_HOST_INSECURE=" + os.Getenv("ARVADOS_API_HOST_INSECURE") + "\n"
	fileContent += "ARVADOS_BLOB_SIGNING_KEY=abcdefg"

	_, err = file.Write([]byte(fileContent))
	c.Check(err, IsNil)

	return file
}

func (s *DoMainTestSuite) Test_doMain_NoSrcConfig(c *C) {
	err := doMain()
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "Error loading src configuration from file: config file not specified")
}

func (s *DoMainTestSuite) Test_doMain_SrcButNoDstConfig(c *C) {
	srcConfig := setupConfigFile(c, "src")
	args := []string{"-replications", "3", "-src", srcConfig.Name()}
	os.Args = append(os.Args, args...)
	err := doMain()
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "Error loading dst configuration from file: config file not specified")
}

func (s *DoMainTestSuite) Test_doMain_BadSrcConfig(c *C) {
	args := []string{"-src", "abcd"}
	os.Args = append(os.Args, args...)
	err := doMain()
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Matches, "Error loading src configuration from file: Error reading config file.*")
}

func (s *DoMainTestSuite) Test_doMain_WithReplicationsButNoSrcConfig(c *C) {
	args := []string{"-replications", "3"}
	os.Args = append(os.Args, args...)
	err := doMain()
	c.Check(err, NotNil)
	c.Assert(err.Error(), Equals, "Error loading src configuration from file: config file not specified")
}

func (s *DoMainTestSuite) Test_doMainWithSrcAndDstConfig(c *C) {
	srcConfig := setupConfigFile(c, "src")
	dstConfig := setupConfigFile(c, "dst")
	args := []string{"-src", srcConfig.Name(), "-dst", dstConfig.Name()}
	os.Args = append(os.Args, args...)

	// Start keepservers. Since we are not doing any tweaking as
	// in setupRsync func, kcSrc and kcDst will be the same and no
	// actual copying to dst will happen, but that's ok.
	arvadostest.StartKeep(2, false)
	defer arvadostest.StopKeep(2)
	keepclient.RefreshServiceDiscovery()

	err := doMain()
	c.Check(err, IsNil)
}
