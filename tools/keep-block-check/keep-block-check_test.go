package main

import (
	"bytes"
	"fmt"
	"io"
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

var kc *keepclient.KeepClient
var keepServicesJSON, blobSigningKey string
var TestHash = "aaaa09c290d0fb1ca068ffaddf22cbd0"
var TestHash2 = "aaaac516f788aec4f30932ffb6395c39"
var allLocators []string

func (s *ServerRequiredSuite) SetUpSuite(c *C) {
	arvadostest.StartAPI()
}

func (s *ServerRequiredSuite) TearDownSuite(c *C) {
	arvadostest.StopAPI()
	arvadostest.ResetEnv()
}

var logBuffer bytes.Buffer

func (s *ServerRequiredSuite) SetUpTest(c *C) {
	blobSigningKey = ""
	keepServicesJSON = ""

	logOutput := io.MultiWriter(&logBuffer)
	log.SetOutput(logOutput)
}

func (s *ServerRequiredSuite) TearDownTest(c *C) {
	arvadostest.StopKeep(2)
	log.SetOutput(os.Stdout)
	log.Printf("%v", logBuffer.String())
}

func (s *DoMainTestSuite) SetUpSuite(c *C) {
}

var testArgs = []string{}

func (s *DoMainTestSuite) SetUpTest(c *C) {
	blobSigningKey = ""
	keepServicesJSON = ""

	logOutput := io.MultiWriter(&logBuffer)
	log.SetOutput(logOutput)
}

func (s *DoMainTestSuite) TearDownTest(c *C) {
	log.SetOutput(os.Stdout)
	log.Printf("%v", logBuffer.String())
	testArgs = []string{}
}

func setupKeepBlockCheck(c *C, enforcePermissions bool) {
	var config apiConfig
	config.APIHost = os.Getenv("ARVADOS_API_HOST")
	config.APIToken = arvadostest.DataManagerToken
	config.APIHostInsecure = matchTrue.MatchString(os.Getenv("ARVADOS_API_HOST_INSECURE"))
	if enforcePermissions {
		blobSigningKey = arvadostest.BlobSigningKey
	}

	// Start Keep servers
	arvadostest.StartKeep(2, enforcePermissions)

	// setup keepclients
	var err error
	kc, err = setupKeepClient(config, keepServicesJSON)
	c.Check(err, IsNil)
}

// Setup test data
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

	// Add config to file. While at it, throw some extra white space
	fileContent := "ARVADOS_API_HOST=" + os.Getenv("ARVADOS_API_HOST") + "\n"
	fileContent += "ARVADOS_API_TOKEN=" + arvadostest.DataManagerToken + "\n"
	fileContent += "\n"
	fileContent += "ARVADOS_API_HOST_INSECURE=" + os.Getenv("ARVADOS_API_HOST_INSECURE") + "\n"
	fileContent += " ARVADOS_EXTERNAL_CLIENT = false \n"
	fileContent += " NotANameValuePairAndShouldGetIgnored \n"
	fileContent += "ARVADOS_BLOB_SIGNING_KEY=abcdefg\n"

	_, err = file.Write([]byte(fileContent))
	c.Check(err, IsNil)

	return file.Name()
}

func setupBlockHashFile(c *C, name string, blocks []string) string {
	// Setup a block hash file
	file, err := ioutil.TempFile(os.TempDir(), name)
	c.Check(err, IsNil)

	// Add the hashes to the file. While at it, throw some extra white space
	fileContent := ""
	for _, hash := range blocks {
		fileContent += fmt.Sprintf(" %s \n", hash)
	}
	fileContent += "\n"
	_, err = file.Write([]byte(fileContent))
	c.Check(err, IsNil)

	return file.Name()
}

func checkErrorLog(c *C, blocks []string, prefix, suffix string) {
	for _, hash := range blocks {
		expected := prefix + `.*` + hash + `.*` + suffix
		match, _ := regexp.MatchString(expected, logBuffer.String())
		c.Assert(match, Equals, true)
	}
}

func checkNoErrorsLogged(c *C, prefix, suffix string) {
	expected := prefix + `.*` + suffix
	match, _ := regexp.MatchString(expected, logBuffer.String())
	c.Assert(match, Equals, false)
}

func (s *ServerRequiredSuite) TestBlockCheck(c *C) {
	setupKeepBlockCheck(c, false)
	setupTestData(c)
	err := performKeepBlockCheck(kc, blobSigningKey, allLocators, true)
	c.Check(err, IsNil)
	checkNoErrorsLogged(c, "Error verifying block", "Block not found")
}

func (s *ServerRequiredSuite) TestBlockCheckWithBlobSigning(c *C) {
	setupKeepBlockCheck(c, true)
	setupTestData(c)
	err := performKeepBlockCheck(kc, blobSigningKey, allLocators, true)
	c.Check(err, IsNil)
	checkNoErrorsLogged(c, "Error verifying block", "Block not found")
}

func (s *ServerRequiredSuite) TestBlockCheck_NoSuchBlock(c *C) {
	setupKeepBlockCheck(c, false)
	setupTestData(c)
	allLocators = append(allLocators, TestHash)
	allLocators = append(allLocators, TestHash2)
	err := performKeepBlockCheck(kc, blobSigningKey, allLocators, true)
	c.Check(err, NotNil)
	c.Assert(err.Error(), Equals, "Block verification failed for 2 out of 7 blocks with matching prefix.")
	checkErrorLog(c, []string{TestHash, TestHash2}, "Error verifying block", "Block not found")
}

func (s *ServerRequiredSuite) TestBlockCheck_NoSuchBlock_WithMatchingPrefix(c *C) {
	setupKeepBlockCheck(c, false)
	setupTestData(c)
	allLocators = append(allLocators, TestHash)
	allLocators = append(allLocators, TestHash2)
	locatorFile := setupBlockHashFile(c, "block-hash", allLocators)
	defer os.Remove(locatorFile)
	locators, err := getBlockLocators(locatorFile, "aaa")
	c.Check(err, IsNil)
	err = performKeepBlockCheck(kc, blobSigningKey, locators, true)
	c.Check(err, NotNil)
	// Of the 7 blocks in allLocators, only two match the prefix and hence only those are checked
	c.Assert(err.Error(), Equals, "Block verification failed for 2 out of 2 blocks with matching prefix.")
	checkErrorLog(c, []string{TestHash, TestHash2}, "Error verifying block", "Block not found")
}

func (s *ServerRequiredSuite) TestBlockCheck_NoSuchBlock_WithPrefixMismatch(c *C) {
	setupKeepBlockCheck(c, false)
	setupTestData(c)
	allLocators = append(allLocators, TestHash)
	allLocators = append(allLocators, TestHash2)
	locatorFile := setupBlockHashFile(c, "block-hash", allLocators)
	defer os.Remove(locatorFile)
	locators, err := getBlockLocators(locatorFile, "999")
	c.Check(err, IsNil)
	err = performKeepBlockCheck(kc, blobSigningKey, locators, true)
	c.Check(err, IsNil) // there were no matching locators and hence no errors
}

func (s *ServerRequiredSuite) TestBlockCheck_BadSignature(c *C) {
	setupKeepBlockCheck(c, true)
	setupTestData(c)
	err := performKeepBlockCheck(kc, "badblobsigningkey", []string{TestHash, TestHash2}, false)
	c.Assert(err.Error(), Equals, "Block verification failed for 2 out of 2 blocks with matching prefix.")
	checkErrorLog(c, []string{TestHash, TestHash2}, "Error verifying block", "HTTP 403")
	// verbose logging not requested
	c.Assert(strings.Contains(logBuffer.String(), "Checking block 1 of 2"), Equals, false)
}

var testKeepServicesJSON = `{
  "kind":"arvados#keepServiceList",
  "etag":"",
  "self_link":"",
  "offset":null, "limit":null,
  "items":[
    {"href":"/keep_services/zzzzz-bi6l4-123456789012340",
     "kind":"arvados#keepService",
     "uuid":"zzzzz-bi6l4-123456789012340",
     "service_host":"keep0.zzzzz.arvadosapi.com",
     "service_port":25107,
     "service_ssl_flag":false,
     "service_type":"disk",
     "read_only":false },
    {"href":"/keep_services/zzzzz-bi6l4-123456789012341",
     "kind":"arvados#keepService",
     "uuid":"zzzzz-bi6l4-123456789012341",
     "service_host":"keep0.zzzzz.arvadosapi.com",
     "service_port":25108,
     "service_ssl_flag":false,
     "service_type":"disk",
     "read_only":false }
    ],
  "items_available":2 }`

// Setup block-check using keepServicesJSON with fake keepservers.
// Expect error during performKeepBlockCheck due to unreachable keepservers.
func (s *ServerRequiredSuite) TestErrorDuringKeepBlockCheck_FakeKeepservers(c *C) {
	keepServicesJSON = testKeepServicesJSON
	setupKeepBlockCheck(c, false)
	err := performKeepBlockCheck(kc, blobSigningKey, []string{TestHash, TestHash2}, true)
	c.Assert(err.Error(), Equals, "Block verification failed for 2 out of 2 blocks with matching prefix.")
	checkErrorLog(c, []string{TestHash, TestHash2}, "Error verifying block", "")
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
	testArgs = append(testArgs, args...)
	err := doMain(testArgs)
	c.Check(err, NotNil)
	c.Assert(strings.Contains(err.Error(), "config file not specified"), Equals, true)
}

func (s *DoMainTestSuite) Test_doMain_WithNoSuchConfigFile(c *C) {
	args := []string{"-config", "no-such-file"}
	testArgs = append(testArgs, args...)
	err := doMain(testArgs)
	c.Check(err, NotNil)
	c.Assert(strings.Contains(err.Error(), "no such file or directory"), Equals, true)
}

func (s *DoMainTestSuite) Test_doMain_WithNoBlockHashFile(c *C) {
	config := setupConfigFile(c, "config")
	defer os.Remove(config)

	args := []string{"-config", config}
	testArgs = append(testArgs, args...)

	// Start keepservers.
	arvadostest.StartKeep(2, false)
	defer arvadostest.StopKeep(2)

	err := doMain(testArgs)
	c.Assert(strings.Contains(err.Error(), "block-hash-file not specified"), Equals, true)
}

func (s *DoMainTestSuite) Test_doMain_WithNoSuchBlockHashFile(c *C) {
	config := setupConfigFile(c, "config")
	defer os.Remove(config)

	args := []string{"-config", config, "-block-hash-file", "no-such-file"}
	testArgs = append(testArgs, args...)

	// Start keepservers.
	arvadostest.StartKeep(2, false)
	defer arvadostest.StopKeep(2)

	err := doMain(testArgs)
	c.Assert(strings.Contains(err.Error(), "no such file or directory"), Equals, true)
}

func (s *DoMainTestSuite) Test_doMain(c *C) {
	// Start keepservers.
	arvadostest.StartKeep(2, false)
	defer arvadostest.StopKeep(2)

	config := setupConfigFile(c, "config")
	defer os.Remove(config)

	locatorFile := setupBlockHashFile(c, "block-hash", []string{TestHash, TestHash2})
	defer os.Remove(locatorFile)

	args := []string{"-config", config, "-block-hash-file", locatorFile, "-v"}
	testArgs = append(testArgs, args...)

	err := doMain(testArgs)
	c.Check(err, NotNil)
	c.Assert(err.Error(), Equals, "Block verification failed for 2 out of 2 blocks with matching prefix.")
	checkErrorLog(c, []string{TestHash, TestHash2}, "Error verifying block", "Block not found")
	c.Assert(strings.Contains(logBuffer.String(), "Checking block 1 of 2"), Equals, true)
}
