// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	. "gopkg.in/check.v1"
)

// Gocheck boilerplate
func Test(t *testing.T) {
	TestingT(t)
}

type TestSuite struct {
	cfg   *ConfigParams
	users map[string]arvados.User
}

func (s *TestSuite) SetUpTest(c *C) {
	ac := arvados.NewClientFromEnv()
	u, err := ac.CurrentUser()
	c.Assert(err, IsNil)
	// Check that the parent group doesn't exist
	sysUserUUID := u.UUID[:12] + "000000000000000"
	gl := arvados.GroupList{}
	params := arvados.ResourceListParams{
		Filters: []arvados.Filter{{
			Attr:     "owner_uuid",
			Operator: "=",
			Operand:  sysUserUUID,
		}, {
			Attr:     "name",
			Operator: "=",
			Operand:  "Externally synchronized groups",
		}},
	}
	ac.RequestAndDecode(&gl, "GET", "/arvados/v1/groups", nil, params)
	c.Assert(gl.ItemsAvailable, Equals, 0)
	// Set up config
	os.Args = []string{"cmd", "somefile.csv"}
	config, err := GetConfig()
	c.Assert(err, IsNil)
	config.UserID = "email"
	// Confirm that the parent group was created
	gl = arvados.GroupList{}
	ac.RequestAndDecode(&gl, "GET", "/arvados/v1/groups", nil, params)
	c.Assert(gl.ItemsAvailable, Equals, 1)
	// Config set up complete, save config for further testing
	s.cfg = &config

	// Fetch current user list
	ul := arvados.UserList{}
	params = arvados.ResourceListParams{
		Filters: []arvados.Filter{{
			Attr:     "uuid",
			Operator: "!=",
			Operand:  s.cfg.SysUserUUID,
		}},
	}
	ac.RequestAndDecode(&ul, "GET", "/arvados/v1/users", nil, params)
	c.Assert(ul.ItemsAvailable, Not(Equals), 0)
	s.users = make(map[string]arvados.User)
	for _, u := range ul.Items {
		s.users[u.UUID] = u
	}
	c.Assert(len(s.users), Not(Equals), 0)
}

func (s *TestSuite) TearDownTest(c *C) {
	var dst interface{}
	// Reset database to fixture state after every test run.
	err := s.cfg.Client.RequestAndDecode(&dst, "POST", "/database/reset", nil, nil)
	c.Assert(err, IsNil)
}

var _ = Suite(&TestSuite{})

// MakeTempCSVFile creates a temp file with data as comma separated values
func MakeTempCSVFile(data [][]string) (f *os.File, err error) {
	f, err = ioutil.TempFile("", "test_sync_remote_groups")
	if err != nil {
		return
	}
	for _, line := range data {
		fmt.Fprintf(f, "%s\n", strings.Join(line, ","))
	}
	err = f.Close()
	return
}

// GroupMembershipExists checks that both needed links exist between user and group
func GroupMembershipExists(ac *arvados.Client, userUUID string, groupUUID string, perm string) bool {
	ll := LinkList{}
	// Check Group -> User can_read permission
	params := arvados.ResourceListParams{
		Filters: []arvados.Filter{{
			Attr:     "link_class",
			Operator: "=",
			Operand:  "permission",
		}, {
			Attr:     "tail_uuid",
			Operator: "=",
			Operand:  groupUUID,
		}, {
			Attr:     "name",
			Operator: "=",
			Operand:  "can_read",
		}, {
			Attr:     "head_uuid",
			Operator: "=",
			Operand:  userUUID,
		}},
	}
	ac.RequestAndDecode(&ll, "GET", "/arvados/v1/links", nil, params)
	if ll.Len() != 1 {
		return false
	}
	// Check User -> Group can_write permission
	params = arvados.ResourceListParams{
		Filters: []arvados.Filter{{
			Attr:     "link_class",
			Operator: "=",
			Operand:  "permission",
		}, {
			Attr:     "head_uuid",
			Operator: "=",
			Operand:  groupUUID,
		}, {
			Attr:     "name",
			Operator: "=",
			Operand:  perm,
		}, {
			Attr:     "tail_uuid",
			Operator: "=",
			Operand:  userUUID,
		}},
	}
	ac.RequestAndDecode(&ll, "GET", "/arvados/v1/links", nil, params)
	return ll.Len() == 1
}

// If named group exists, return its UUID
func RemoteGroupExists(cfg *ConfigParams, groupName string) (uuid string, err error) {
	gl := arvados.GroupList{}
	params := arvados.ResourceListParams{
		Filters: []arvados.Filter{{
			Attr:     "name",
			Operator: "=",
			Operand:  groupName,
		}, {
			Attr:     "owner_uuid",
			Operator: "=",
			Operand:  cfg.SysUserUUID,
		}, {
			Attr:     "group_class",
			Operator: "=",
			Operand:  "role",
		}},
	}
	err = cfg.Client.RequestAndDecode(&gl, "GET", "/arvados/v1/groups", nil, params)
	if err != nil {
		return "", err
	}
	if gl.ItemsAvailable == 0 {
		// No group with this name
		uuid = ""
	} else if gl.ItemsAvailable == 1 {
		// Group found
		uuid = gl.Items[0].UUID
	} else {
		// This should never happen
		uuid = ""
		err = fmt.Errorf("more than 1 group found with the same name and parent")
	}
	return
}

func (s *TestSuite) TestParseFlagsWithPositionalArgument(c *C) {
	cfg := ConfigParams{}
	os.Args = []string{"cmd", "-verbose", "-case-insensitive", "/tmp/somefile.csv"}
	err := ParseFlags(&cfg)
	c.Assert(err, IsNil)
	c.Check(cfg.Path, Equals, "/tmp/somefile.csv")
	c.Check(cfg.Verbose, Equals, true)
	c.Check(cfg.CaseInsensitive, Equals, true)
}

func (s *TestSuite) TestParseFlagsWithoutPositionalArgument(c *C) {
	os.Args = []string{"cmd", "-verbose"}
	err := ParseFlags(&ConfigParams{})
	c.Assert(err, NotNil)
}

func (s *TestSuite) TestGetUserID(c *C) {
	u := arvados.User{
		Email:    "testuser@example.com",
		Username: "Testuser",
	}
	email, err := GetUserID(u, "email")
	c.Assert(err, IsNil)
	c.Check(email, Equals, "testuser@example.com")
	_, err = GetUserID(u, "bogus")
	c.Assert(err, NotNil)
}

func (s *TestSuite) TestGetConfig(c *C) {
	os.Args = []string{"cmd", "/tmp/somefile.csv"}
	cfg, err := GetConfig()
	c.Assert(err, IsNil)
	c.Check(cfg.SysUserUUID, NotNil)
	c.Check(cfg.Client, NotNil)
	c.Check(cfg.ParentGroupUUID, NotNil)
	c.Check(cfg.ParentGroupName, Equals, "Externally synchronized groups")
}

// Ignore leading & trailing spaces on group & users names
func (s *TestSuite) TestIgnoreSpaces(c *C) {
	activeUserEmail := s.users[arvadostest.ActiveUserUUID].Email
	activeUserUUID := s.users[arvadostest.ActiveUserUUID].UUID
	// Confirm that the groups don't exist
	for _, groupName := range []string{"TestGroup1", "TestGroup2", "Test Group 3"} {
		groupUUID, err := RemoteGroupExists(s.cfg, groupName)
		c.Assert(err, IsNil)
		c.Assert(groupUUID, Equals, "")
	}
	data := [][]string{
		{" TestGroup1", activeUserEmail},
		{"TestGroup2 ", " " + activeUserEmail},
		{" Test Group 3 ", activeUserEmail + " "},
	}
	tmpfile, err := MakeTempCSVFile(data)
	c.Assert(err, IsNil)
	defer os.Remove(tmpfile.Name()) // clean up
	s.cfg.Path = tmpfile.Name()
	err = doMain(s.cfg)
	c.Assert(err, IsNil)
	// Check that 3 groups were created correctly, and have the active user as
	// a member.
	for _, groupName := range []string{"TestGroup1", "TestGroup2", "Test Group 3"} {
		groupUUID, err := RemoteGroupExists(s.cfg, groupName)
		c.Assert(err, IsNil)
		c.Assert(groupUUID, Not(Equals), "")
		c.Assert(GroupMembershipExists(s.cfg.Client, activeUserUUID, groupUUID, "can_write"), Equals, true)
	}
}

// Error out when records have <2 or >3 records
func (s *TestSuite) TestWrongNumberOfFields(c *C) {
	for _, testCase := range [][][]string{
		{{"field1"}},
		{{"field1", "field2", "field3", "field4"}},
		{{"field1", "field2", "field3", "field4", "field5"}},
	} {
		tmpfile, err := MakeTempCSVFile(testCase)
		c.Assert(err, IsNil)
		defer os.Remove(tmpfile.Name())
		s.cfg.Path = tmpfile.Name()
		err = doMain(s.cfg)
		c.Assert(err, Not(IsNil))
	}
}

// Check different membership permissions
func (s *TestSuite) TestMembershipLevels(c *C) {
	userEmail := s.users[arvadostest.ActiveUserUUID].Email
	userUUID := s.users[arvadostest.ActiveUserUUID].UUID
	data := [][]string{
		{"TestGroup1", userEmail, "can_read"},
		{"TestGroup2", userEmail, "can_write"},
		{"TestGroup3", userEmail, "can_manage"},
		{"TestGroup4", userEmail, "invalid_permission"},
	}
	tmpfile, err := MakeTempCSVFile(data)
	c.Assert(err, IsNil)
	defer os.Remove(tmpfile.Name()) // clean up
	s.cfg.Path = tmpfile.Name()
	err = doMain(s.cfg)
	c.Assert(err, IsNil)
	for _, record := range data {
		groupName := record[0]
		permLevel := record[2]
		if permLevel != "invalid_permission" {
			groupUUID, err := RemoteGroupExists(s.cfg, groupName)
			c.Assert(err, IsNil)
			c.Assert(groupUUID, Not(Equals), "")
			c.Assert(GroupMembershipExists(s.cfg.Client, userUUID, groupUUID, permLevel), Equals, true)
		} else {
			groupUUID, err := RemoteGroupExists(s.cfg, groupName)
			c.Assert(err, IsNil)
			c.Assert(groupUUID, Equals, "")
		}
	}
}

// Check membership level change
func (s *TestSuite) TestMembershipLevelUpdate(c *C) {
	userEmail := s.users[arvadostest.ActiveUserUUID].Email
	userUUID := s.users[arvadostest.ActiveUserUUID].UUID
	groupName := "TestGroup1"
	// Give read permissions
	tmpfile, err := MakeTempCSVFile([][]string{{groupName, userEmail, "can_read"}})
	c.Assert(err, IsNil)
	defer os.Remove(tmpfile.Name()) // clean up
	s.cfg.Path = tmpfile.Name()
	err = doMain(s.cfg)
	c.Assert(err, IsNil)
	// Check permissions
	groupUUID, err := RemoteGroupExists(s.cfg, groupName)
	c.Assert(err, IsNil)
	c.Assert(groupUUID, Not(Equals), "")
	c.Assert(GroupMembershipExists(s.cfg.Client, userUUID, groupUUID, "can_read"), Equals, true)
	c.Assert(GroupMembershipExists(s.cfg.Client, userUUID, groupUUID, "can_write"), Equals, false)
	c.Assert(GroupMembershipExists(s.cfg.Client, userUUID, groupUUID, "can_manage"), Equals, false)

	// Give write permissions
	tmpfile, err = MakeTempCSVFile([][]string{{groupName, userEmail, "can_write"}})
	c.Assert(err, IsNil)
	defer os.Remove(tmpfile.Name()) // clean up
	s.cfg.Path = tmpfile.Name()
	err = doMain(s.cfg)
	c.Assert(err, IsNil)
	// Check permissions
	c.Assert(GroupMembershipExists(s.cfg.Client, userUUID, groupUUID, "can_read"), Equals, false)
	c.Assert(GroupMembershipExists(s.cfg.Client, userUUID, groupUUID, "can_write"), Equals, true)
	c.Assert(GroupMembershipExists(s.cfg.Client, userUUID, groupUUID, "can_manage"), Equals, false)

	// Give manage permissions
	tmpfile, err = MakeTempCSVFile([][]string{{groupName, userEmail, "can_manage"}})
	c.Assert(err, IsNil)
	defer os.Remove(tmpfile.Name()) // clean up
	s.cfg.Path = tmpfile.Name()
	err = doMain(s.cfg)
	c.Assert(err, IsNil)
	// Check permissions
	c.Assert(GroupMembershipExists(s.cfg.Client, userUUID, groupUUID, "can_read"), Equals, false)
	c.Assert(GroupMembershipExists(s.cfg.Client, userUUID, groupUUID, "can_write"), Equals, false)
	c.Assert(GroupMembershipExists(s.cfg.Client, userUUID, groupUUID, "can_manage"), Equals, true)
}

// The absence of a user membership on the CSV file implies its removal
func (s *TestSuite) TestMembershipRemoval(c *C) {
	localUserEmail := s.users[arvadostest.ActiveUserUUID].Email
	localUserUUID := s.users[arvadostest.ActiveUserUUID].UUID
	remoteUserEmail := s.users[arvadostest.FederatedActiveUserUUID].Email
	remoteUserUUID := s.users[arvadostest.FederatedActiveUserUUID].UUID
	data := [][]string{
		{"TestGroup1", localUserEmail},
		{"TestGroup1", remoteUserEmail},
		{"TestGroup2", localUserEmail},
		{"TestGroup2", remoteUserEmail},
	}
	tmpfile, err := MakeTempCSVFile(data)
	c.Assert(err, IsNil)
	defer os.Remove(tmpfile.Name()) // clean up
	s.cfg.Path = tmpfile.Name()
	err = doMain(s.cfg)
	c.Assert(err, IsNil)
	// Confirm that memberships exist
	for _, groupName := range []string{"TestGroup1", "TestGroup2"} {
		groupUUID, err := RemoteGroupExists(s.cfg, groupName)
		c.Assert(err, IsNil)
		c.Assert(groupUUID, Not(Equals), "")
		c.Assert(GroupMembershipExists(s.cfg.Client, localUserUUID, groupUUID, "can_write"), Equals, true)
		c.Assert(GroupMembershipExists(s.cfg.Client, remoteUserUUID, groupUUID, "can_write"), Equals, true)
	}
	// New CSV with some previous membership missing
	data = [][]string{
		{"TestGroup1", localUserEmail},
		{"TestGroup2", remoteUserEmail},
	}
	tmpfile2, err := MakeTempCSVFile(data)
	c.Assert(err, IsNil)
	defer os.Remove(tmpfile2.Name()) // clean up
	s.cfg.Path = tmpfile2.Name()
	err = doMain(s.cfg)
	c.Assert(err, IsNil)
	// Confirm TestGroup1 memberships
	groupUUID, err := RemoteGroupExists(s.cfg, "TestGroup1")
	c.Assert(err, IsNil)
	c.Assert(groupUUID, Not(Equals), "")
	c.Assert(GroupMembershipExists(s.cfg.Client, localUserUUID, groupUUID, "can_write"), Equals, true)
	c.Assert(GroupMembershipExists(s.cfg.Client, remoteUserUUID, groupUUID, "can_write"), Equals, false)
	// Confirm TestGroup1 memberships
	groupUUID, err = RemoteGroupExists(s.cfg, "TestGroup2")
	c.Assert(err, IsNil)
	c.Assert(groupUUID, Not(Equals), "")
	c.Assert(GroupMembershipExists(s.cfg.Client, localUserUUID, groupUUID, "can_write"), Equals, false)
	c.Assert(GroupMembershipExists(s.cfg.Client, remoteUserUUID, groupUUID, "can_write"), Equals, true)
}

// If a group doesn't exist on the system, create it before adding users
func (s *TestSuite) TestAutoCreateGroupWhenNotExisting(c *C) {
	groupName := "Testers"
	// Confirm that group doesn't exist
	groupUUID, err := RemoteGroupExists(s.cfg, groupName)
	c.Assert(err, IsNil)
	c.Assert(groupUUID, Equals, "")
	// Make a tmp CSV file
	data := [][]string{
		{groupName, s.users[arvadostest.ActiveUserUUID].Email},
	}
	tmpfile, err := MakeTempCSVFile(data)
	c.Assert(err, IsNil)
	defer os.Remove(tmpfile.Name()) // clean up
	s.cfg.Path = tmpfile.Name()
	err = doMain(s.cfg)
	c.Assert(err, IsNil)
	// "Testers" group should now exist
	groupUUID, err = RemoteGroupExists(s.cfg, groupName)
	c.Assert(err, IsNil)
	c.Assert(groupUUID, Not(Equals), "")
	// active user should be a member
	c.Assert(GroupMembershipExists(s.cfg.Client, arvadostest.ActiveUserUUID, groupUUID, "can_write"), Equals, true)
}

// Users listed on the file that don't exist on the system are ignored
func (s *TestSuite) TestIgnoreNonexistantUsers(c *C) {
	activeUserEmail := s.users[arvadostest.ActiveUserUUID].Email
	activeUserUUID := s.users[arvadostest.ActiveUserUUID].UUID
	// Confirm that group doesn't exist
	groupUUID, err := RemoteGroupExists(s.cfg, "TestGroup4")
	c.Assert(err, IsNil)
	c.Assert(groupUUID, Equals, "")
	// Create file & run command
	data := [][]string{
		{"TestGroup4", "nonexistantuser@unknowndomain.com"}, // Processed first
		{"TestGroup4", activeUserEmail},
	}
	tmpfile, err := MakeTempCSVFile(data)
	c.Assert(err, IsNil)
	defer os.Remove(tmpfile.Name()) // clean up
	s.cfg.Path = tmpfile.Name()
	err = doMain(s.cfg)
	c.Assert(err, IsNil)
	// Confirm that memberships exist
	groupUUID, err = RemoteGroupExists(s.cfg, "TestGroup4")
	c.Assert(err, IsNil)
	c.Assert(groupUUID, Not(Equals), "")
	c.Assert(GroupMembershipExists(s.cfg.Client, activeUserUUID, groupUUID, "can_write"), Equals, true)
}

// Entries with missing data are ignored.
func (s *TestSuite) TestIgnoreEmptyFields(c *C) {
	activeUserEmail := s.users[arvadostest.ActiveUserUUID].Email
	activeUserUUID := s.users[arvadostest.ActiveUserUUID].UUID
	// Confirm that group doesn't exist
	for _, groupName := range []string{"TestGroup4", "TestGroup5"} {
		groupUUID, err := RemoteGroupExists(s.cfg, groupName)
		c.Assert(err, IsNil)
		c.Assert(groupUUID, Equals, "")
	}
	// Create file & run command
	data := [][]string{
		{"", activeUserEmail},               // Empty field
		{"TestGroup5", ""},                  // Empty field
		{"TestGroup5", activeUserEmail, ""}, // Empty 3rd field: is optional but cannot be empty
		{"TestGroup4", activeUserEmail},
	}
	tmpfile, err := MakeTempCSVFile(data)
	c.Assert(err, IsNil)
	defer os.Remove(tmpfile.Name()) // clean up
	s.cfg.Path = tmpfile.Name()
	err = doMain(s.cfg)
	c.Assert(err, IsNil)
	// Confirm that records about TestGroup5 were skipped
	groupUUID, err := RemoteGroupExists(s.cfg, "TestGroup5")
	c.Assert(err, IsNil)
	c.Assert(groupUUID, Equals, "")
	// Confirm that membership exists
	groupUUID, err = RemoteGroupExists(s.cfg, "TestGroup4")
	c.Assert(err, IsNil)
	c.Assert(groupUUID, Not(Equals), "")
	c.Assert(GroupMembershipExists(s.cfg.Client, activeUserUUID, groupUUID, "can_write"), Equals, true)
}

// Instead of emails, use username as identifier
func (s *TestSuite) TestUseUsernames(c *C) {
	activeUserName := s.users[arvadostest.ActiveUserUUID].Username
	activeUserUUID := s.users[arvadostest.ActiveUserUUID].UUID
	// Confirm that group doesn't exist
	groupUUID, err := RemoteGroupExists(s.cfg, "TestGroup1")
	c.Assert(err, IsNil)
	c.Assert(groupUUID, Equals, "")
	// Create file & run command
	data := [][]string{
		{"TestGroup1", activeUserName},
	}
	tmpfile, err := MakeTempCSVFile(data)
	c.Assert(err, IsNil)
	defer os.Remove(tmpfile.Name()) // clean up
	s.cfg.Path = tmpfile.Name()
	s.cfg.UserID = "username"
	err = doMain(s.cfg)
	c.Assert(err, IsNil)
	// Confirm that memberships exist
	groupUUID, err = RemoteGroupExists(s.cfg, "TestGroup1")
	c.Assert(err, IsNil)
	c.Assert(groupUUID, Not(Equals), "")
	c.Assert(GroupMembershipExists(s.cfg.Client, activeUserUUID, groupUUID, "can_write"), Equals, true)
}

func (s *TestSuite) TestUseUsernamesWithCaseInsensitiveMatching(c *C) {
	activeUserName := strings.ToUpper(s.users[arvadostest.ActiveUserUUID].Username)
	activeUserUUID := s.users[arvadostest.ActiveUserUUID].UUID
	// Confirm that group doesn't exist
	groupUUID, err := RemoteGroupExists(s.cfg, "TestGroup1")
	c.Assert(err, IsNil)
	c.Assert(groupUUID, Equals, "")
	// Create file & run command
	data := [][]string{
		{"TestGroup1", activeUserName},
	}
	tmpfile, err := MakeTempCSVFile(data)
	c.Assert(err, IsNil)
	defer os.Remove(tmpfile.Name()) // clean up
	s.cfg.Path = tmpfile.Name()
	s.cfg.UserID = "username"
	s.cfg.CaseInsensitive = true
	err = doMain(s.cfg)
	c.Assert(err, IsNil)
	// Confirm that memberships exist
	groupUUID, err = RemoteGroupExists(s.cfg, "TestGroup1")
	c.Assert(err, IsNil)
	c.Assert(groupUUID, Not(Equals), "")
	c.Assert(GroupMembershipExists(s.cfg.Client, activeUserUUID, groupUUID, "can_write"), Equals, true)
}

func (s *TestSuite) TestUsernamesCaseInsensitiveCollision(c *C) {
	activeUserName := s.users[arvadostest.ActiveUserUUID].Username
	activeUserUUID := s.users[arvadostest.ActiveUserUUID].UUID

	nu := arvados.User{}
	nuUsername := strings.ToUpper(activeUserName)
	err := s.cfg.Client.RequestAndDecode(&nu, "POST", "/arvados/v1/users", nil, map[string]interface{}{
		"user": map[string]string{
			"username": nuUsername,
		},
	})
	c.Assert(err, IsNil)

	// Manually remove non-fixture user because /database/reset fails otherwise
	defer s.cfg.Client.RequestAndDecode(nil, "DELETE", "/arvados/v1/users/"+nu.UUID, nil, nil)

	c.Assert(nu.Username, Equals, nuUsername)
	c.Assert(nu.UUID, Not(Equals), activeUserUUID)
	c.Assert(nu.Username, Not(Equals), activeUserName)

	data := [][]string{
		{"SomeGroup", activeUserName},
	}
	tmpfile, err := MakeTempCSVFile(data)
	c.Assert(err, IsNil)
	defer os.Remove(tmpfile.Name()) // clean up

	s.cfg.Path = tmpfile.Name()
	s.cfg.UserID = "username"
	s.cfg.CaseInsensitive = true
	err = doMain(s.cfg)
	// Should get an error because of "ACTIVE" and "Active" usernames
	c.Assert(err, NotNil)
	c.Assert(err, ErrorMatches, ".*case insensitive collision.*")
}
