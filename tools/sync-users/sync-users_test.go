// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"testing"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	. "gopkg.in/check.v1"
)

// Gocheck boilerplate
func Test(t *testing.T) {
	TestingT(t)
}

type TestSuite struct {
	cfg   *ConfigParams
	ac    *arvados.Client
	users map[string]arvados.User
}

func (s *TestSuite) SetUpTest(c *C) {
	s.ac = arvados.NewClientFromEnv()
	u, err := s.ac.CurrentUser()
	c.Assert(err, IsNil)
	c.Assert(u.IsAdmin, Equals, true)

	s.users = make(map[string]arvados.User)
	ul := arvados.UserList{}
	s.ac.RequestAndDecode(&ul, "GET", "/arvados/v1/users", nil, arvados.ResourceListParams{})
	c.Assert(ul.ItemsAvailable, Not(Equals), 0)
	s.users = make(map[string]arvados.User)
	for _, u := range ul.Items {
		s.users[u.UUID] = u
	}

	// Set up command config
	os.Args = []string{"cmd", "somefile.csv"}
	config, err := GetConfig()
	c.Assert(err, IsNil)
	s.cfg = &config
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
	f, err = ioutil.TempFile("", "test_sync_users")
	if err != nil {
		return
	}
	for _, line := range data {
		fmt.Fprintf(f, "%s\n", strings.Join(line, ","))
	}
	err = f.Close()
	return
}

// RecordsToStrings formats the input data suitable for MakeTempCSVFile
func RecordsToStrings(records []userRecord) [][]string {
	data := [][]string{}
	for _, u := range records {
		data = append(data, []string{
			u.UserID,
			u.FirstName,
			u.LastName,
			fmt.Sprintf("%t", u.Active),
			fmt.Sprintf("%t", u.Admin)})
	}
	return data
}

func ListUsers(ac *arvados.Client) ([]arvados.User, error) {
	var ul arvados.UserList
	err := ac.RequestAndDecode(&ul, "GET", "/arvados/v1/users", nil, arvados.ResourceListParams{})
	if err != nil {
		return nil, err
	}
	return ul.Items, nil
}

func (s *TestSuite) TestParseFlagsWithoutPositionalArgument(c *C) {
	os.Args = []string{"cmd", "-verbose"}
	err := ParseFlags(&ConfigParams{})
	c.Assert(err, NotNil)
	c.Assert(err, ErrorMatches, ".*please provide a path to an input file.*")
}

func (s *TestSuite) TestParseFlagsWrongUserID(c *C) {
	os.Args = []string{"cmd", "-user-id", "nickname", "/tmp/somefile.csv"}
	err := ParseFlags(&ConfigParams{})
	c.Assert(err, NotNil)
	c.Assert(err, ErrorMatches, ".*user ID must be one of:.*")
}

func (s *TestSuite) TestParseFlagsWithPositionalArgument(c *C) {
	cfg := ConfigParams{}
	os.Args = []string{"cmd", "/tmp/somefile.csv"}
	err := ParseFlags(&cfg)
	c.Assert(err, IsNil)
	c.Assert(cfg.Path, Equals, "/tmp/somefile.csv")
	c.Assert(cfg.Verbose, Equals, false)
	c.Assert(cfg.DeactivateUnlisted, Equals, false)
	c.Assert(cfg.UserID, Equals, "email")
	c.Assert(cfg.CaseInsensitive, Equals, true)
}

func (s *TestSuite) TestParseFlagsWithOptionalFlags(c *C) {
	cfg := ConfigParams{}
	os.Args = []string{"cmd", "-verbose", "-deactivate-unlisted", "-user-id", "username", "/tmp/somefile.csv"}
	err := ParseFlags(&cfg)
	c.Assert(err, IsNil)
	c.Assert(cfg.Path, Equals, "/tmp/somefile.csv")
	c.Assert(cfg.Verbose, Equals, true)
	c.Assert(cfg.DeactivateUnlisted, Equals, true)
	c.Assert(cfg.UserID, Equals, "username")
	c.Assert(cfg.CaseInsensitive, Equals, false)
}

func (s *TestSuite) TestGetConfig(c *C) {
	os.Args = []string{"cmd", "/tmp/somefile.csv"}
	cfg, err := GetConfig()
	c.Assert(err, IsNil)
	c.Assert(cfg.AnonUserUUID, Not(Equals), "")
	c.Assert(cfg.SysUserUUID, Not(Equals), "")
	c.Assert(cfg.CurrentUser, Not(Equals), "")
	c.Assert(cfg.ClusterID, Not(Equals), "")
	c.Assert(cfg.Client, NotNil)
}

func (s *TestSuite) TestFailOnEmptyFields(c *C) {
	records := [][]string{
		{"", "first-name", "last-name", "1", "0"},
		{"user@example", "", "last-name", "1", "0"},
		{"user@example", "first-name", "", "1", "0"},
		{"user@example", "first-name", "last-name", "", "0"},
		{"user@example", "first-name", "last-name", "1", ""},
	}
	for _, record := range records {
		data := [][]string{record}
		tmpfile, err := MakeTempCSVFile(data)
		c.Assert(err, IsNil)
		defer os.Remove(tmpfile.Name())
		s.cfg.Path = tmpfile.Name()
		err = doMain(s.cfg)
		c.Assert(err, NotNil)
		c.Assert(err, ErrorMatches, ".*fields cannot be empty.*")
	}
}

func (s *TestSuite) TestIgnoreSpaces(c *C) {
	// Make sure users aren't already there from fixtures
	for _, user := range s.users {
		e := user.Email
		found := e == "user1@example.com" || e == "user2@example.com" || e == "user3@example.com"
		c.Assert(found, Equals, false)
	}
	// Use CSV data with leading/trailing whitespaces, confirm that they get ignored
	data := [][]string{
		{" user1@example.com", "  Example", "   User1", "1", "0"},
		{"user2@example.com ", "Example  ", "User2   ", "1", "0"},
		{" user3@example.com ", "  Example  ", "   User3   ", "1", "0"},
	}
	tmpfile, err := MakeTempCSVFile(data)
	c.Assert(err, IsNil)
	defer os.Remove(tmpfile.Name())
	s.cfg.Path = tmpfile.Name()
	err = doMain(s.cfg)
	c.Assert(err, IsNil)
	users, err := ListUsers(s.cfg.Client)
	c.Assert(err, IsNil)
	for _, userNr := range []int{1, 2, 3} {
		found := false
		for _, user := range users {
			if user.Email == fmt.Sprintf("user%d@example.com", userNr) &&
				user.LastName == fmt.Sprintf("User%d", userNr) &&
				user.FirstName == "Example" && user.IsActive == true {
				found = true
				break
			}
		}
		c.Assert(found, Equals, true)
	}
}

// Error out when records have != 5 records
func (s *TestSuite) TestWrongNumberOfFields(c *C) {
	for _, testCase := range [][][]string{
		{{"user1@example.com", "Example", "User1", "1"}},
		{{"user1@example.com", "Example", "User1", "1", "0", "extra data"}},
	} {
		tmpfile, err := MakeTempCSVFile(testCase)
		c.Assert(err, IsNil)
		defer os.Remove(tmpfile.Name())
		s.cfg.Path = tmpfile.Name()
		err = doMain(s.cfg)
		c.Assert(err, NotNil)
		c.Assert(err, ErrorMatches, ".*expected 5 fields, found.*")
	}
}

// Error out when records have incorrect data types
func (s *TestSuite) TestWrongDataFields(c *C) {
	for _, testCase := range [][][]string{
		{{"user1@example.com", "Example", "User1", "yep", "0"}},
		{{"user1@example.com", "Example", "User1", "1", "nope"}},
	} {
		tmpfile, err := MakeTempCSVFile(testCase)
		c.Assert(err, IsNil)
		defer os.Remove(tmpfile.Name())
		s.cfg.Path = tmpfile.Name()
		err = doMain(s.cfg)
		c.Assert(err, NotNil)
		c.Assert(err, ErrorMatches, ".*parsing error at line.*[active|admin] status not recognized.*")
	}
}

// Create, activate and deactivate users
func (s *TestSuite) TestUserCreationAndUpdate(c *C) {
	for _, tc := range []string{"email", "username"} {
		uIDPrefix := tc
		uIDSuffix := ""
		if tc == "email" {
			uIDSuffix = "@example.com"
		}
		s.cfg.UserID = tc
		records := []userRecord{{
			UserID:    fmt.Sprintf("%suser1%s", uIDPrefix, uIDSuffix),
			FirstName: "Example",
			LastName:  "User1",
			Active:    true,
			Admin:     false,
		}, {
			UserID:    fmt.Sprintf("%suser2%s", uIDPrefix, uIDSuffix),
			FirstName: "Example",
			LastName:  "User2",
			Active:    false, // initially inactive
			Admin:     false,
		}, {
			UserID:    fmt.Sprintf("%sadmin1%s", uIDPrefix, uIDSuffix),
			FirstName: "Example",
			LastName:  "Admin1",
			Active:    true,
			Admin:     true,
		}, {
			UserID:    fmt.Sprintf("%sadmin2%s", uIDPrefix, uIDSuffix),
			FirstName: "Example",
			LastName:  "Admin2",
			Active:    false, // initially inactive
			Admin:     true,
		}}
		// Make sure users aren't already there from fixtures
		for _, user := range s.users {
			uID, err := GetUserID(user, s.cfg.UserID)
			c.Assert(err, IsNil)
			found := false
			for _, r := range records {
				if uID == r.UserID {
					found = true
					break
				}
			}
			c.Assert(found, Equals, false)
		}
		// User creation
		tmpfile, err := MakeTempCSVFile(RecordsToStrings(records))
		c.Assert(err, IsNil)
		defer os.Remove(tmpfile.Name())
		s.cfg.Path = tmpfile.Name()
		err = doMain(s.cfg)
		c.Assert(err, IsNil)

		users, err := ListUsers(s.cfg.Client)
		c.Assert(err, IsNil)
		for _, r := range records {
			var foundUser arvados.User
			for _, user := range users {
				uID, err := GetUserID(user, s.cfg.UserID)
				c.Assert(err, IsNil)
				if uID == r.UserID {
					// Add an @example.com email if missing
					// (to avoid database reset errors)
					if tc == "username" && user.Email == "" {
						err := UpdateUser(s.cfg.Client, user.UUID, &user, map[string]string{
							"email": fmt.Sprintf("%s@example.com", user.Username),
						})
						c.Assert(err, IsNil)
					}
					foundUser = user
					break
				}
			}
			c.Assert(foundUser, NotNil)
			c.Logf("Checking creation for user %q", r.UserID)
			c.Assert(foundUser.FirstName, Equals, r.FirstName)
			c.Assert(foundUser.LastName, Equals, r.LastName)
			c.Assert(foundUser.IsActive, Equals, r.Active)
			c.Assert(foundUser.IsAdmin, Equals, (r.Active && r.Admin))
		}
		// User update
		for idx := range records {
			records[idx].Active = !records[idx].Active
			records[idx].FirstName = records[idx].FirstName + "Updated"
			records[idx].LastName = records[idx].LastName + "Updated"
		}
		tmpfile, err = MakeTempCSVFile(RecordsToStrings(records))
		c.Assert(err, IsNil)
		defer os.Remove(tmpfile.Name())
		s.cfg.Path = tmpfile.Name()
		err = doMain(s.cfg)
		c.Assert(err, IsNil)

		users, err = ListUsers(s.cfg.Client)
		c.Assert(err, IsNil)
		for _, r := range records {
			var foundUser arvados.User
			for _, user := range users {
				uID, err := GetUserID(user, s.cfg.UserID)
				c.Assert(err, IsNil)
				if uID == r.UserID {
					foundUser = user
					break
				}
			}
			c.Assert(foundUser, NotNil)
			c.Logf("Checking update for user %q", r.UserID)
			c.Assert(foundUser.FirstName, Equals, r.FirstName)
			c.Assert(foundUser.LastName, Equals, r.LastName)
			c.Assert(foundUser.IsActive, Equals, r.Active)
			c.Assert(foundUser.IsAdmin, Equals, (r.Active && r.Admin))
		}
	}
}

func (s *TestSuite) TestDeactivateUnlisted(c *C) {
	localUserUuidRegex := regexp.MustCompile(fmt.Sprintf("^%s-tpzed-[0-9a-z]{15}$", s.cfg.ClusterID))

	var user1 arvados.User
	for _, nr := range []int{1, 2} {
		var newUser arvados.User
		err := CreateUser(s.cfg.Client, &newUser, map[string]string{
			"email":      fmt.Sprintf("user%d@example.com", nr),
			"first_name": "Example",
			"last_name":  fmt.Sprintf("User%d", nr),
			"is_active":  "true",
			"is_admin":   "false",
		})
		c.Assert(err, IsNil)
		c.Assert(newUser.IsActive, Equals, true)
		if nr == 1 {
			user1 = newUser // for later confirmation
		}
	}

	users, err := ListUsers(s.cfg.Client)
	c.Assert(err, IsNil)
	previouslyActiveUsers := 0
	for _, u := range users {
		if u.UUID == fmt.Sprintf("%s-tpzed-anonymouspublic", s.cfg.ClusterID) && !u.IsActive {
			// Make sure the anonymous user is active for this test
			var au arvados.User
			err := UpdateUser(s.cfg.Client, u.UUID, &au, map[string]string{"is_active": "true"})
			c.Assert(err, IsNil)
			c.Assert(au.IsActive, Equals, true)
		}
		if localUserUuidRegex.MatchString(u.UUID) && u.IsActive {
			previouslyActiveUsers++
		}
	}
	// Active users: System root, Anonymous, current user and the
	// ones just created (other active users may exist from fixture).
	c.Logf("Initial active users count: %d", previouslyActiveUsers)
	c.Assert(previouslyActiveUsers > 5, Equals, true)

	// Here we omit user2@example.com from the CSV file.
	data := [][]string{
		{"user1@example.com", "Example", "User1", "1", "0"},
	}
	tmpfile, err := MakeTempCSVFile(data)
	c.Assert(err, IsNil)
	defer os.Remove(tmpfile.Name())

	s.cfg.DeactivateUnlisted = true
	s.cfg.Verbose = true
	s.cfg.Path = tmpfile.Name()
	err = doMain(s.cfg)
	c.Assert(err, IsNil)

	users, err = ListUsers(s.cfg.Client)
	c.Assert(err, IsNil)
	currentlyActiveUsers := 0
	acceptableActiveUUIDs := map[string]bool{
		fmt.Sprintf("%s-tpzed-000000000000000", s.cfg.ClusterID): true,
		fmt.Sprintf("%s-tpzed-anonymouspublic", s.cfg.ClusterID): true,
		s.cfg.CurrentUser.UUID: true,
		user1.UUID:             true,
	}
	remainingActiveUUIDs := map[string]bool{}
	seenUserEmails := map[string]bool{}
	for _, u := range users {
		if _, ok := seenUserEmails[u.Email]; ok {
			c.Errorf("Duplicated email address %q in user list (probably from fixtures). This test requires unique email addresses.", u.Email)
		}
		seenUserEmails[u.Email] = true
		if localUserUuidRegex.MatchString(u.UUID) && u.IsActive {
			c.Logf("Found remaining active user %q (%s)", u.Email, u.UUID)
			_, ok := acceptableActiveUUIDs[u.UUID]
			c.Assert(ok, Equals, true)
			remainingActiveUUIDs[u.UUID] = true
			currentlyActiveUsers++
		}
	}
	// 4 active users remaining: System root, Anonymous, the current user
	// and user1@example.com
	c.Logf("Active local users remaining: %v", remainingActiveUUIDs)
	c.Assert(currentlyActiveUsers, Equals, 4)
}

func (s *TestSuite) TestFailOnDuplicatedEmails(c *C) {
	for i := range []int{1, 2} {
		isAdmin := i == 2
		err := CreateUser(s.cfg.Client, &arvados.User{}, map[string]string{
			"email":      "somedupedemail@example.com",
			"first_name": fmt.Sprintf("Duped %d", i),
			"username":   fmt.Sprintf("dupedemail%d", i),
			"last_name":  "User",
			"is_active":  "true",
			"is_admin":   fmt.Sprintf("%t", isAdmin),
		})
		c.Assert(err, IsNil)
	}
	s.cfg.Verbose = true
	data := [][]string{
		{"user1@example.com", "Example", "User1", "0", "0"},
	}
	tmpfile, err := MakeTempCSVFile(data)
	c.Assert(err, IsNil)
	defer os.Remove(tmpfile.Name())
	s.cfg.Path = tmpfile.Name()
	err = doMain(s.cfg)
	c.Assert(err, NotNil)
	c.Assert(err, ErrorMatches, "skipped.*duplicated email address.*")
}

func (s *TestSuite) TestFailOnEmptyUsernames(c *C) {
	for i := range []int{1, 2} {
		var user arvados.User
		err := CreateUser(s.cfg.Client, &user, map[string]string{
			"email":      fmt.Sprintf("johndoe%d@example.com", i),
			"username":   "",
			"first_name": "John",
			"last_name":  "Doe",
			"is_active":  "true",
			"is_admin":   "false",
		})
		c.Assert(err, IsNil)
		c.Assert(user.Username, Equals, fmt.Sprintf("johndoe%d", i))
		if i == 1 {
			err = UpdateUser(s.cfg.Client, user.UUID, &user, map[string]string{
				"username": "",
			})
			c.Assert(err, IsNil)
			c.Assert(user.Username, Equals, "")
		}
	}

	s.cfg.Verbose = true
	data := [][]string{
		{"johndoe0", "John", "Doe", "0", "0"},
	}
	tmpfile, err := MakeTempCSVFile(data)
	c.Assert(err, IsNil)
	defer os.Remove(tmpfile.Name())
	s.cfg.Path = tmpfile.Name()
	s.cfg.UserID = "username"
	err = doMain(s.cfg)
	c.Assert(err, NotNil)
	c.Assert(err, ErrorMatches, "skipped 1 user account.*with empty username.*")
}

func (s *TestSuite) TestFailOnDupedUsernameAndCaseInsensitiveMatching(c *C) {
	for _, i := range []int{1, 2} {
		var user arvados.User
		emailPrefix := "john"
		if i == 1 {
			emailPrefix = "JOHN"
		}
		err := CreateUser(s.cfg.Client, &user, map[string]string{
			"email":      fmt.Sprintf("%sdoe@example.com", emailPrefix),
			"username":   "",
			"first_name": "John",
			"last_name":  "Doe",
			"is_active":  "true",
			"is_admin":   "false",
		})
		c.Assert(err, IsNil)
		c.Assert(user.Username, Equals, fmt.Sprintf("%sdoe", emailPrefix))
	}

	s.cfg.Verbose = true
	data := [][]string{
		{"johndoe", "John", "Doe", "0", "0"},
	}
	tmpfile, err := MakeTempCSVFile(data)
	c.Assert(err, IsNil)
	defer os.Remove(tmpfile.Name())
	s.cfg.Path = tmpfile.Name()
	s.cfg.UserID = "username"
	s.cfg.CaseInsensitive = true
	err = doMain(s.cfg)
	c.Assert(err, NotNil)
	c.Assert(err, ErrorMatches, "case insensitive collision for username.*between.*and.*")
}

func (s *TestSuite) TestSuccessOnUsernameAndCaseSensitiveMatching(c *C) {
	for _, i := range []int{1, 2} {
		var user arvados.User
		emailPrefix := "john"
		if i == 1 {
			emailPrefix = "JOHN"
		}
		err := CreateUser(s.cfg.Client, &user, map[string]string{
			"email":      fmt.Sprintf("%sdoe@example.com", emailPrefix),
			"username":   "",
			"first_name": "John",
			"last_name":  "Doe",
			"is_active":  "true",
			"is_admin":   "false",
		})
		c.Assert(err, IsNil)
		c.Assert(user.Username, Equals, fmt.Sprintf("%sdoe", emailPrefix))
	}

	s.cfg.Verbose = true
	data := [][]string{
		{"johndoe", "John", "Doe", "0", "0"},
	}
	tmpfile, err := MakeTempCSVFile(data)
	c.Assert(err, IsNil)
	defer os.Remove(tmpfile.Name())
	s.cfg.Path = tmpfile.Name()
	s.cfg.UserID = "username"
	s.cfg.CaseInsensitive = false
	err = doMain(s.cfg)
	c.Assert(err, IsNil)
}
