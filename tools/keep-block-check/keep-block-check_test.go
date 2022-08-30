// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

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
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvadosclient"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"git.arvados.org/arvados.git/sdk/go/keepclient"

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
var logBuffer bytes.Buffer

var TestHash = "aaaa09c290d0fb1ca068ffaddf22cbd0"
var TestHash2 = "aaaac516f788aec4f30932ffb6395c39"

var blobSignatureTTL = time.Duration(2*7*24) * time.Hour

func (s *ServerRequiredSuite) TearDownSuite(c *C) {
	arvadostest.ResetEnv()
}

func (s *ServerRequiredSuite) SetUpTest(c *C) {
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

func (s *DoMainTestSuite) SetUpTest(c *C) {
	logOutput := io.MultiWriter(&logBuffer)
	log.SetOutput(logOutput)
	keepclient.RefreshServiceDiscovery()
}

func (s *DoMainTestSuite) TearDownTest(c *C) {
	log.SetOutput(os.Stdout)
	log.Printf("%v", logBuffer.String())
}

func setupKeepBlockCheck(c *C, enforcePermissions bool, keepServicesJSON string) {
	setupKeepBlockCheckWithTTL(c, enforcePermissions, keepServicesJSON, blobSignatureTTL)
}

func setupKeepBlockCheckWithTTL(c *C, enforcePermissions bool, keepServicesJSON string, ttl time.Duration) {
	var config apiConfig
	config.APIHost = os.Getenv("ARVADOS_API_HOST")
	config.APIToken = arvadostest.DataManagerToken
	config.APIHostInsecure = arvadosclient.StringBool(os.Getenv("ARVADOS_API_HOST_INSECURE"))

	// Start Keep servers
	arvadostest.StartKeep(2, enforcePermissions)

	// setup keepclients
	var err error
	kc, ttl, err = setupKeepClient(config, keepServicesJSON, ttl)
	c.Assert(ttl, Equals, blobSignatureTTL)
	c.Check(err, IsNil)

	keepclient.RefreshServiceDiscovery()
}

// Setup test data
func setupTestData(c *C) []string {
	allLocators := []string{}

	// Put a few blocks
	for i := 0; i < 5; i++ {
		hash, _, err := kc.PutB([]byte(fmt.Sprintf("keep-block-check-test-data-%d", i)))
		c.Check(err, IsNil)
		allLocators = append(allLocators, strings.Split(hash, "+A")[0])
	}

	return allLocators
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
		expected := `(?ms).*` + prefix + `.*` + hash + `.*` + suffix + `.*`
		c.Check(logBuffer.String(), Matches, expected)
	}
}

func checkNoErrorsLogged(c *C, prefix, suffix string) {
	expected := prefix + `.*` + suffix
	match, _ := regexp.MatchString(expected, logBuffer.String())
	c.Assert(match, Equals, false)
}

func (s *ServerRequiredSuite) TestBlockCheck(c *C) {
	setupKeepBlockCheck(c, false, "")
	allLocators := setupTestData(c)
	err := performKeepBlockCheck(kc, blobSignatureTTL, "", allLocators, true)
	c.Check(err, IsNil)
	checkNoErrorsLogged(c, "Error verifying block", "Block not found")
}

func (s *ServerRequiredSuite) TestBlockCheckWithBlobSigning(c *C) {
	setupKeepBlockCheck(c, true, "")
	allLocators := setupTestData(c)
	err := performKeepBlockCheck(kc, blobSignatureTTL, arvadostest.BlobSigningKey, allLocators, true)
	c.Check(err, IsNil)
	checkNoErrorsLogged(c, "Error verifying block", "Block not found")
}

func (s *ServerRequiredSuite) TestBlockCheckWithBlobSigningAndTTLFromDiscovery(c *C) {
	setupKeepBlockCheckWithTTL(c, true, "", 0)
	allLocators := setupTestData(c)
	err := performKeepBlockCheck(kc, blobSignatureTTL, arvadostest.BlobSigningKey, allLocators, true)
	c.Check(err, IsNil)
	checkNoErrorsLogged(c, "Error verifying block", "Block not found")
}

func (s *ServerRequiredSuite) TestBlockCheck_NoSuchBlock(c *C) {
	setupKeepBlockCheck(c, false, "")
	allLocators := setupTestData(c)
	allLocators = append(allLocators, TestHash)
	allLocators = append(allLocators, TestHash2)
	err := performKeepBlockCheck(kc, blobSignatureTTL, "", allLocators, true)
	c.Check(err, NotNil)
	c.Assert(err.Error(), Equals, "Block verification failed for 2 out of 7 blocks with matching prefix")
	checkErrorLog(c, []string{TestHash, TestHash2}, "Error verifying block", "Block not found")
}

func (s *ServerRequiredSuite) TestBlockCheck_NoSuchBlock_WithMatchingPrefix(c *C) {
	setupKeepBlockCheck(c, false, "")
	allLocators := setupTestData(c)
	allLocators = append(allLocators, TestHash)
	allLocators = append(allLocators, TestHash2)
	locatorFile := setupBlockHashFile(c, "block-hash", allLocators)
	defer os.Remove(locatorFile)
	locators, err := getBlockLocators(locatorFile, "aaa")
	c.Check(err, IsNil)
	err = performKeepBlockCheck(kc, blobSignatureTTL, "", locators, true)
	c.Check(err, NotNil)
	// Of the 7 blocks in allLocators, only two match the prefix and hence only those are checked
	c.Assert(err.Error(), Equals, "Block verification failed for 2 out of 2 blocks with matching prefix")
	checkErrorLog(c, []string{TestHash, TestHash2}, "Error verifying block", "Block not found")
}

func (s *ServerRequiredSuite) TestBlockCheck_NoSuchBlock_WithPrefixMismatch(c *C) {
	setupKeepBlockCheck(c, false, "")
	allLocators := setupTestData(c)
	allLocators = append(allLocators, TestHash)
	allLocators = append(allLocators, TestHash2)
	locatorFile := setupBlockHashFile(c, "block-hash", allLocators)
	defer os.Remove(locatorFile)
	locators, err := getBlockLocators(locatorFile, "999")
	c.Check(err, IsNil)
	err = performKeepBlockCheck(kc, blobSignatureTTL, "", locators, true)
	c.Check(err, IsNil) // there were no matching locators in file and hence nothing was checked
}

func (s *ServerRequiredSuite) TestBlockCheck_BadSignature(c *C) {
	setupKeepBlockCheck(c, true, "")
	setupTestData(c)
	err := performKeepBlockCheck(kc, blobSignatureTTL, "badblobsigningkey", []string{TestHash, TestHash2}, false)
	c.Assert(err.Error(), Equals, "Block verification failed for 2 out of 2 blocks with matching prefix")
	checkErrorLog(c, []string{TestHash, TestHash2}, "Error verifying block", "HTTP 403")
	// verbose logging not requested
	c.Assert(strings.Contains(logBuffer.String(), "Verifying block 1 of 2"), Equals, false)
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
	setupKeepBlockCheck(c, false, testKeepServicesJSON)
	err := performKeepBlockCheck(kc, blobSignatureTTL, "", []string{TestHash, TestHash2}, true)
	c.Assert(err.Error(), Equals, "Block verification failed for 2 out of 2 blocks with matching prefix")
	checkErrorLog(c, []string{TestHash, TestHash2}, "Error verifying block", "")
}

// Test keep-block-check initialization with keepServicesJSON
func (s *ServerRequiredSuite) TestKeepBlockCheck_InitializeWithKeepServicesJSON(c *C) {
	setupKeepBlockCheck(c, false, testKeepServicesJSON)
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
	c.Assert(config.APIHostInsecure, Equals, arvadosclient.StringBool(os.Getenv("ARVADOS_API_HOST_INSECURE")))
	c.Assert(blobSigningKey, Equals, "abcdefg")
}

func (s *DoMainTestSuite) Test_doMain_WithNoConfig(c *C) {
	args := []string{"-prefix", "a"}
	var stderr bytes.Buffer
	code := doMain(args, &stderr)
	c.Check(code, Equals, 1)
	c.Check(stderr.String(), Matches, ".*config file not specified\n")
}

func (s *DoMainTestSuite) Test_doMain_WithNoSuchConfigFile(c *C) {
	args := []string{"-config", "no-such-file"}
	var stderr bytes.Buffer
	code := doMain(args, &stderr)
	c.Check(code, Equals, 1)
	c.Check(stderr.String(), Matches, ".*no such file or directory\n")
}

func (s *DoMainTestSuite) Test_doMain_WithNoBlockHashFile(c *C) {
	config := setupConfigFile(c, "config")
	defer os.Remove(config)

	// Start keepservers.
	arvadostest.StartKeep(2, false)
	defer arvadostest.StopKeep(2)

	args := []string{"-config", config}
	var stderr bytes.Buffer
	code := doMain(args, &stderr)
	c.Check(code, Equals, 1)
	c.Check(stderr.String(), Matches, ".*block-hash-file not specified\n")
}

func (s *DoMainTestSuite) Test_doMain_WithNoSuchBlockHashFile(c *C) {
	config := setupConfigFile(c, "config")
	defer os.Remove(config)

	arvadostest.StartKeep(2, false)
	defer arvadostest.StopKeep(2)

	args := []string{"-config", config, "-block-hash-file", "no-such-file"}
	var stderr bytes.Buffer
	code := doMain(args, &stderr)
	c.Check(code, Equals, 1)
	c.Check(stderr.String(), Matches, ".*no such file or directory\n")
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
	var stderr bytes.Buffer
	code := doMain(args, &stderr)
	c.Check(code, Equals, 1)
	c.Assert(stderr.String(), Matches, "Block verification failed for 2 out of 2 blocks with matching prefix\n")
	checkErrorLog(c, []string{TestHash, TestHash2}, "Error verifying block", "Block not found")
	c.Assert(strings.Contains(logBuffer.String(), "Verifying block 1 of 2"), Equals, true)
}
