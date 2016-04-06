package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strings"
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
var _ = Suite(&DoMainTestSuite{})

type ServerRequiredSuite struct{}
type DoMainTestSuite struct{}

func (s *ServerRequiredSuite) SetUpSuite(c *C) {
	arvadostest.StartAPI()
}

func (s *ServerRequiredSuite) TearDownSuite(c *C) {
	arvadostest.StopAPI()
	arvadostest.ResetEnv()
}

func (s *ServerRequiredSuite) SetUpTest(c *C) {
	blobSigningKey = ""
	keepServicesJSON = ""

	tempfile, err := ioutil.TempFile(os.TempDir(), "temp-log-file")
	c.Check(err, IsNil)
	log.SetOutput(tempfile)
	tempLogFileName = tempfile.Name()
}

func (s *ServerRequiredSuite) TearDownTest(c *C) {
	arvadostest.StopKeep(2)
	os.Remove(tempLogFileName)
}

var tempLogFileName = ""
var initialArgs []string
var kc *keepclient.KeepClient
var keepServicesJSON, blobSigningKey string

func (s *DoMainTestSuite) SetUpSuite(c *C) {
	initialArgs = os.Args
}

func (s *DoMainTestSuite) SetUpTest(c *C) {
	blobSigningKey = ""
	keepServicesJSON = ""

	args := []string{"keep-block-check"}
	os.Args = args

	tempfile, err := ioutil.TempFile(os.TempDir(), "temp-log-file")
	c.Check(err, IsNil)
	log.SetOutput(tempfile)
	tempLogFileName = tempfile.Name()
}

func (s *DoMainTestSuite) TearDownTest(c *C) {
	os.Remove(tempLogFileName)
	os.Args = initialArgs
}

var testKeepServicesJSON = "{ \"kind\":\"arvados#keepServiceList\", \"etag\":\"\", \"self_link\":\"\", \"offset\":null, \"limit\":null, \"items\":[ { \"href\":\"/keep_services/zzzzz-bi6l4-123456789012340\", \"kind\":\"arvados#keepService\", \"etag\":\"641234567890enhj7hzx432e5\", \"uuid\":\"zzzzz-bi6l4-123456789012340\", \"owner_uuid\":\"zzzzz-tpzed-123456789012345\", \"service_host\":\"keep0.zzzzz.arvadosapi.com\", \"service_port\":25107, \"service_ssl_flag\":false, \"service_type\":\"disk\", \"read_only\":false }, { \"href\":\"/keep_services/zzzzz-bi6l4-123456789012341\", \"kind\":\"arvados#keepService\", \"etag\":\"641234567890enhj7hzx432e5\", \"uuid\":\"zzzzz-bi6l4-123456789012341\", \"owner_uuid\":\"zzzzz-tpzed-123456789012345\", \"service_host\":\"keep0.zzzzz.arvadosapi.com\", \"service_port\":25108, \"service_ssl_flag\":false, \"service_type\":\"disk\", \"read_only\":false } ], \"items_available\":2 }"

var TestHash = "aaaa09c290d0fb1ca068ffaddf22cbd0"
var TestHash2 = "aaaac516f788aec4f30932ffb6395c39"

func setupKeepBlockCheck(c *C, enforcePermissions bool) {
	var config apiConfig
	config.APIHost = os.Getenv("ARVADOS_API_HOST")
	config.APIToken = arvadostest.DataManagerToken
	config.APIHostInsecure = matchTrue.MatchString(os.Getenv("ARVADOS_API_HOST_INSECURE"))
	if enforcePermissions {
		blobSigningKey = "zfhgfenhffzltr9dixws36j1yhksjoll2grmku38mi7yxd66h5j4q9w4jzanezacp8s6q0ro3hxakfye02152hncy6zml2ed0uc"
	}

	// Start Keep servers
	arvadostest.StartKeep(2, enforcePermissions)

	// setup keepclients
  var err error
	kc, err = setupKeepClient(config, keepServicesJSON)
	c.Check(err, IsNil)
}

// Setup test data
var allLocators []string

func setupTestData(c *C) {
	allLocators = []string{}

	// Put a few blocks
	for i := 0; i < 5; i++ {
		hash, _, err := kc.PutB([]byte(fmt.Sprintf("keep-block-check-test-data-%d", i)))
		c.Check(err, IsNil)
		allLocators = append(allLocators, strings.Split(hash, "+A")[0])
	}
}

func setupConfigFile(c *C, fileName string) string {
	// Setup a config file
	file, err := ioutil.TempFile(os.TempDir(), fileName)
	c.Check(err, IsNil)

	fileContent := "ARVADOS_API_HOST=" + os.Getenv("ARVADOS_API_HOST") + "\n"
	fileContent += "ARVADOS_API_TOKEN=" + arvadostest.DataManagerToken + "\n"
	fileContent += "ARVADOS_API_HOST_INSECURE=" + os.Getenv("ARVADOS_API_HOST_INSECURE") + "\n"
	fileContent += "ARVADOS_EXTERNAL_CLIENT=false\n"
	fileContent += "ARVADOS_BLOB_SIGNING_KEY=abcdefg"

	_, err = file.Write([]byte(fileContent))
	c.Check(err, IsNil)

	return file.Name()
}

func setupBlockHashFile(c *C, name string, blocks []string) string {
	// Setup a block hash file
	file, err := ioutil.TempFile(os.TempDir(), name)
	c.Check(err, IsNil)

	fileContent := ""
	for _, hash := range blocks {
		fileContent += fmt.Sprintf("%s\n", hash)
	}
	_, err = file.Write([]byte(fileContent))
	c.Check(err, IsNil)

	return file.Name()
}

func checkErrorLog(c *C, blocks []string, msg string) {
	buf, _ := ioutil.ReadFile(tempLogFileName)
	if len(blocks) == 0 {
		expected := `HEAD .*` + msg
		match, _ := regexp.MatchString(expected, string(buf))
		c.Assert(match, Equals, false)
		return
	}
	for _, hash := range blocks {
		expected := hash + `.*` + msg
		match, _ := regexp.MatchString(expected, string(buf))
		c.Assert(match, Equals, true)
	}
}

func (s *ServerRequiredSuite) TestBlockCheck(c *C) {
	setupKeepBlockCheck(c, false)
	setupTestData(c)
	performKeepBlockCheck(kc, blobSigningKey, "", allLocators)
	checkErrorLog(c, []string{}, "Block not found") // no errors
}

func (s *ServerRequiredSuite) TestBlockCheckWithBlobSigning(c *C) {
	setupKeepBlockCheck(c, true)
	setupTestData(c)
	performKeepBlockCheck(kc, blobSigningKey, "", allLocators)
	checkErrorLog(c, []string{}, "Block not found") // no errors
}

func (s *ServerRequiredSuite) TestBlockCheck_NoSuchBlock(c *C) {
	setupKeepBlockCheck(c, false)
	setupTestData(c)
	performKeepBlockCheck(kc, blobSigningKey, "", []string{TestHash, TestHash2})
	checkErrorLog(c, []string{TestHash, TestHash2}, "Block not found")
}

func (s *ServerRequiredSuite) TestBlockCheck_NoSuchBlock_WithMatchingPrefix(c *C) {
	setupKeepBlockCheck(c, false)
	setupTestData(c)
	performKeepBlockCheck(kc, blobSigningKey, "aaa", []string{TestHash, TestHash2})
	checkErrorLog(c, []string{TestHash, TestHash2}, "Block not found")
}

func (s *ServerRequiredSuite) TestBlockCheck_NoSuchBlock_WithPrefixMismatch(c *C) {
	setupKeepBlockCheck(c, false)
	setupTestData(c)
	performKeepBlockCheck(kc, blobSigningKey, "999", []string{TestHash, TestHash2})
	checkErrorLog(c, []string{}, "Block not found") // no errors
}

// Setup block-check using keepServicesJSON with fake keepservers.
// Expect error during performKeepBlockCheck due to unreachable keepservers.
func (s *ServerRequiredSuite) TestErrorDuringKeepBlockCheck_FakeKeepservers(c *C) {
	keepServicesJSON = testKeepServicesJSON
	setupKeepBlockCheck(c, false)
	performKeepBlockCheck(kc, blobSigningKey, "", []string{TestHash, TestHash2})
	checkErrorLog(c, []string{TestHash, TestHash2}, "timeout")
}

func (s *ServerRequiredSuite) TestBlockCheck_BadSignature(c *C) {
	setupKeepBlockCheck(c, true)
	setupTestData(c)
	performKeepBlockCheck(kc, "badblobsigningkey", "", []string{TestHash, TestHash2})
	checkErrorLog(c, []string{TestHash, TestHash2}, "HTTP 403")
}

// Test keep-block-check initialization with keepServicesJSON
func (s *ServerRequiredSuite) TestKeepBlockCheck_InitializeWithKeepServicesJSON(c *C) {
	keepServicesJSON = testKeepServicesJSON
	setupKeepBlockCheck(c, false)
	found := 0
	for k := range kc.LocalRoots() {
		if k == "zzzzz-bi6l4-123456789012340" || k == "zzzzz-bi6l4-123456789012341" {
			found++
		}
	}
	c.Check(found, Equals, 2)
}

// Test loadConfig func
func (s *ServerRequiredSuite) TestLoadConfig(c *C) {
	// Setup config file
	configFile := setupConfigFile(c, "config")
	defer os.Remove(configFile)

	// load configuration from the file
	config, blobSigningKey, err := loadConfig(configFile)
	c.Check(err, IsNil)

	c.Assert(config.APIHost, Equals, os.Getenv("ARVADOS_API_HOST"))
	c.Assert(config.APIToken, Equals, arvadostest.DataManagerToken)
	c.Assert(config.APIHostInsecure, Equals, matchTrue.MatchString(os.Getenv("ARVADOS_API_HOST_INSECURE")))
	c.Assert(config.ExternalClient, Equals, false)
	c.Assert(blobSigningKey, Equals, "abcdefg")
}

func (s *DoMainTestSuite) Test_doMain_WithNoConfig(c *C) {
	args := []string{"-prefix", "a"}
	os.Args = append(os.Args, args...)
	err := doMain()
	c.Check(err, NotNil)
	c.Assert(strings.Contains(err.Error(), "config file not specified"), Equals, true)
}

func (s *DoMainTestSuite) Test_doMain_WithNoSuchConfigFile(c *C) {
	args := []string{"-config", "no-such-file"}
	os.Args = append(os.Args, args...)
	err := doMain()
	c.Check(err, NotNil)
	c.Assert(strings.Contains(err.Error(), "no such file or directory"), Equals, true)
}

func (s *DoMainTestSuite) Test_doMain_WithNoBlockHashFile(c *C) {
	config := setupConfigFile(c, "config")
	defer os.Remove(config)

	args := []string{"-config", config}
	os.Args = append(os.Args, args...)

	// Start keepservers.
	arvadostest.StartKeep(2, false)
	defer arvadostest.StopKeep(2)

	err := doMain()
	c.Assert(strings.Contains(err.Error(), "block-hash-file not specified"), Equals, true)
}

func (s *DoMainTestSuite) Test_doMain_WithNoSuchBlockHashFile(c *C) {
	config := setupConfigFile(c, "config")
	defer os.Remove(config)

	args := []string{"-config", config, "-block-hash-file", "no-such-file"}
	os.Args = append(os.Args, args...)

	// Start keepservers.
	arvadostest.StartKeep(2, false)
	defer arvadostest.StopKeep(2)

	err := doMain()
	c.Assert(strings.Contains(err.Error(), "no such file or directory"), Equals, true)
}

func (s *DoMainTestSuite) Test_doMain(c *C) {
	config := setupConfigFile(c, "config")
	defer os.Remove(config)

	locatorFile := setupBlockHashFile(c, "block-hash", []string{TestHash, TestHash2})
	defer os.Remove(locatorFile)

	// Start keepservers.
	arvadostest.StartKeep(2, false)
	defer arvadostest.StopKeep(2)

	args := []string{"-config", config, "-block-hash-file", locatorFile}
	os.Args = append(os.Args, args...)

	err := doMain()
	c.Check(err, IsNil)
	checkErrorLog(c, []string{TestHash, TestHash2}, "Block not found")
}
