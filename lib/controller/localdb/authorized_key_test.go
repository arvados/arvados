// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package localdb

import (
	_ "embed"
	"errors"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"git.arvados.org/arvados.git/sdk/go/httpserver"
	. "gopkg.in/check.v1"
)

var _ = Suite(&authorizedKeySuite{})

type authorizedKeySuite struct {
	localdbSuite
}

//go:embed testdata/rsa.pub
var testPubKey string

func (s *authorizedKeySuite) TestAuthorizedKeyCreate(c *C) {
	ak, err := s.localdb.AuthorizedKeyCreate(s.userctx, arvados.CreateOptions{
		Attrs: map[string]interface{}{
			"name":     "testkey",
			"key_type": "SSH",
		}})
	c.Assert(err, IsNil)
	c.Check(ak.KeyType, Equals, "SSH")
	defer s.localdb.AuthorizedKeyDelete(s.userctx, arvados.DeleteOptions{UUID: ak.UUID})
	updated, err := s.localdb.AuthorizedKeyUpdate(s.userctx, arvados.UpdateOptions{
		UUID:  ak.UUID,
		Attrs: map[string]interface{}{"name": "testkeyrenamed"}})
	c.Check(err, IsNil)
	c.Check(updated.UUID, Equals, ak.UUID)
	c.Check(updated.Name, Equals, "testkeyrenamed")
	c.Check(updated.ModifiedByUserUUID, Equals, arvadostest.ActiveUserUUID)

	_, err = s.localdb.AuthorizedKeyCreate(s.userctx, arvados.CreateOptions{
		Attrs: map[string]interface{}{
			"name":       "testkey",
			"public_key": "ssh-dsa boguskey\n",
		}})
	c.Check(err, ErrorMatches, `Public key does not appear to be valid: ssh: no key found`)
	_, err = s.localdb.AuthorizedKeyUpdate(s.userctx, arvados.UpdateOptions{
		UUID: ak.UUID,
		Attrs: map[string]interface{}{
			"public_key": strings.Replace(testPubKey, "A", "#", 1),
		}})
	c.Check(err, ErrorMatches, `Public key does not appear to be valid: ssh: no key found`)
	_, err = s.localdb.AuthorizedKeyUpdate(s.userctx, arvados.UpdateOptions{
		UUID: ak.UUID,
		Attrs: map[string]interface{}{
			"public_key": testPubKey + testPubKey,
		}})
	c.Check(err, ErrorMatches, `Public key does not appear to be valid: extra data after key`)
	_, err = s.localdb.AuthorizedKeyUpdate(s.userctx, arvados.UpdateOptions{
		UUID: ak.UUID,
		Attrs: map[string]interface{}{
			"public_key": testPubKey + "# extra data\n",
		}})
	c.Check(err, ErrorMatches, `Public key does not appear to be valid: extra data after key`)
	_, err = s.localdb.AuthorizedKeyUpdate(s.userctx, arvados.UpdateOptions{
		UUID: ak.UUID,
		Attrs: map[string]interface{}{
			"public_key": strings.Replace(testPubKey, "ssh-rsa", "ssh-dsa", 1),
		}})
	c.Check(err, ErrorMatches, `Public key does not appear to be valid: leading type field "ssh-dsa" does not match actual key type "ssh-rsa"`)
	var se httpserver.HTTPStatusError
	if c.Check(errors.As(err, &se), Equals, true) {
		c.Check(se.HTTPStatus(), Equals, http.StatusBadRequest)
	}

	dirents, err := os.ReadDir("./testdata")
	c.Assert(err, IsNil)
	c.Assert(dirents, Not(HasLen), 0)
	for _, dirent := range dirents {
		if !strings.HasSuffix(dirent.Name(), ".pub") {
			continue
		}
		pubkeyfile := "./testdata/" + dirent.Name()
		c.Logf("checking public key from %s", pubkeyfile)
		pubkey, err := ioutil.ReadFile(pubkeyfile)
		if !c.Check(err, IsNil) {
			continue
		}
		updated, err := s.localdb.AuthorizedKeyUpdate(s.userctx, arvados.UpdateOptions{
			UUID: ak.UUID,
			Attrs: map[string]interface{}{
				"public_key": string(pubkey),
			}})
		c.Check(err, IsNil)
		c.Check(updated.PublicKey, Equals, string(pubkey))

		_, err = s.localdb.AuthorizedKeyUpdate(s.userctx, arvados.UpdateOptions{
			UUID: ak.UUID,
			Attrs: map[string]interface{}{
				"public_key": strings.Replace(string(pubkey), " ", "-bogus ", 1),
			}})
		c.Check(err, ErrorMatches, `.*type field ".*" does not match actual key type ".*"`)
	}

	deleted, err := s.localdb.AuthorizedKeyDelete(s.userctx, arvados.DeleteOptions{UUID: ak.UUID})
	c.Check(err, IsNil)
	c.Check(deleted.UUID, Equals, ak.UUID)
}
