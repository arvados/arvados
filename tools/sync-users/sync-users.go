// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"

	"git.arvados.org/arvados.git/lib/cmd"
	"git.arvados.org/arvados.git/sdk/go/arvados"
)

var version = "dev"

type resourceList interface {
	Len() int
	GetItems() []interface{}
}

// UserList implements resourceList interface
type UserList struct {
	arvados.UserList
}

// Len returns the amount of items this list holds
func (l UserList) Len() int {
	return len(l.Items)
}

// GetItems returns the list of items
func (l UserList) GetItems() (out []interface{}) {
	for _, item := range l.Items {
		out = append(out, item)
	}
	return
}

func main() {
	cfg, err := GetConfig()
	if err != nil {
		log.Fatalf("%v", err)
	}

	if err := doMain(&cfg); err != nil {
		log.Fatalf("%v", err)
	}
}

type ConfigParams struct {
	Client             *arvados.Client
	ClusterID          string
	CurrentUser        arvados.User
	DeactivateUnlisted bool
	Path               string
	SysUserUUID        string
	AnonUserUUID       string
	Verbose            bool
}

func ParseFlags(cfg *ConfigParams) error {
	flags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flags.Usage = func() {
		usageStr := `Synchronize remote users into Arvados from a CSV format file with 5 columns:
  * 1st: E-mail address
  * 2nd: First name
  * 3rd: Last name
  * 4th: Active status (0 or 1)
  * 5th: Admin status (0 or 1)`
		fmt.Fprintf(flags.Output(), "%s\n\n", usageStr)
		fmt.Fprintf(flags.Output(), "Usage:\n%s [OPTIONS] <input-file.csv>\n\n", os.Args[0])
		fmt.Fprintf(flags.Output(), "Options:\n")
		flags.PrintDefaults()
	}

	deactivateUnlisted := flags.Bool(
		"deactivate-unlisted",
		false,
		"Deactivate users that are not in the input file.")
	verbose := flags.Bool(
		"verbose",
		false,
		"Log informational messages. Off by default.")
	getVersion := flags.Bool(
		"version",
		false,
		"Print version information and exit.")

	if ok, code := cmd.ParseFlags(flags, os.Args[0], os.Args[1:], "input-file.csv", os.Stderr); !ok {
		os.Exit(code)
	} else if *getVersion {
		fmt.Printf("%s %s\n", os.Args[0], version)
		os.Exit(0)
	}

	// Input file as a required positional argument
	if flags.NArg() == 0 {
		return fmt.Errorf("please provide a path to an input file")
	}
	srcPath := &os.Args[flags.NFlag()+1]

	// Validations
	if *srcPath == "" {
		return fmt.Errorf("input file path invalid")
	}

	cfg.DeactivateUnlisted = *deactivateUnlisted
	cfg.Path = *srcPath
	cfg.Verbose = *verbose

	return nil
}

// GetConfig sets up a ConfigParams struct
func GetConfig() (cfg ConfigParams, err error) {
	err = ParseFlags(&cfg)
	if err != nil {
		return
	}

	cfg.Client = arvados.NewClientFromEnv()

	// Check current user permissions
	u, err := cfg.Client.CurrentUser()
	if err != nil {
		return cfg, fmt.Errorf("error getting the current user: %s", err)
	}
	if !u.IsAdmin {
		return cfg, fmt.Errorf("current user %q is not an admin user", u.UUID)
	}
	if cfg.Verbose {
		log.Printf("Running as admin user %q (%s)", u.Email, u.UUID)
	}
	cfg.CurrentUser = u

	var ac struct {
		ClusterID string
		Login     struct {
			LoginCluster string
		}
	}
	err = cfg.Client.RequestAndDecode(&ac, "GET", "arvados/v1/config", nil, nil)
	if err != nil {
		return cfg, fmt.Errorf("error getting the exported config: %s", err)
	}
	if ac.Login.LoginCluster != "" && ac.Login.LoginCluster != ac.ClusterID {
		return cfg, fmt.Errorf("cannot run on a cluster other than the login cluster")
	}
	cfg.SysUserUUID = ac.ClusterID + "-tpzed-000000000000000"
	cfg.AnonUserUUID = ac.ClusterID + "-tpzed-anonymouspublic"
	cfg.ClusterID = ac.ClusterID

	return cfg, nil
}

func doMain(cfg *ConfigParams) error {
	// Try opening the input file early, just in case there's a problem.
	f, err := os.Open(cfg.Path)
	if err != nil {
		return fmt.Errorf("error opening input file: %s", err)
	}
	defer f.Close()

	allUsers := make(map[string]arvados.User)
	dupedEmails := make(map[string][]arvados.User)
	processedUsers := make(map[string]bool)
	results, err := GetAll(cfg.Client, "users", arvados.ResourceListParams{}, &UserList{})
	if err != nil {
		return fmt.Errorf("error getting all users: %s", err)
	}
	log.Printf("Found %d users in cluster %q", len(results), cfg.ClusterID)
	localUserUuidRegex := regexp.MustCompile(fmt.Sprintf("^%s-tpzed-[0-9a-z]{15}$", cfg.ClusterID))
	for _, item := range results {
		u := item.(arvados.User)
		// Remote user check
		if !localUserUuidRegex.MatchString(u.UUID) {
			if cfg.Verbose {
				log.Printf("Remote user %q (%s) won't be considered for processing", u.Email, u.UUID)
			}
			continue
		}
		// Duplicated user's email check
		email := strings.ToLower(u.Email)
		if ul, ok := dupedEmails[email]; ok {
			log.Printf("Duplicated email %q found in user %s", email, u.UUID)
			dupedEmails[email] = append(ul, u)
			continue
		}
		if eu, ok := allUsers[email]; ok {
			log.Printf("Duplicated email %q found in users %s and %s", email, eu.UUID, u.UUID)
			dupedEmails[email] = []arvados.User{eu, u}
			delete(allUsers, email)
			continue
		}
		allUsers[email] = u
		processedUsers[email] = false
	}

	loadedRecords, err := LoadInputFile(f)
	if err != nil {
		return fmt.Errorf("reading input file %q: %s", cfg.Path, err)
	}
	log.Printf("Loaded %d records from input file", len(loadedRecords))

	updatesSucceeded := map[string]bool{}
	updatesFailed := map[string]bool{}
	updatesSkipped := map[string]bool{}

	for _, record := range loadedRecords {
		processedUsers[record.Email] = true
		if record.Email == cfg.CurrentUser.Email {
			updatesSkipped[record.Email] = true
			log.Printf("Skipping current user %q (%s) from processing", record.Email, cfg.CurrentUser.UUID)
			continue
		}
		if updated, err := ProcessRecord(cfg, record, allUsers); err != nil {
			log.Printf("error processing record %q: %s", record.Email, err)
			updatesFailed[record.Email] = true
		} else if updated {
			updatesSucceeded[record.Email] = true
		}
	}

	if cfg.DeactivateUnlisted {
		for email, user := range allUsers {
			if shouldSkip(cfg, user) {
				updatesSkipped[email] = true
				log.Printf("Skipping unlisted user %q (%s) from deactivating", user.Email, user.UUID)
				continue
			}
			if !processedUsers[email] && allUsers[email].IsActive {
				if cfg.Verbose {
					log.Printf("Deactivating unlisted user %q (%s)", user.Email, user.UUID)
				}
				var updatedUser arvados.User
				if err := UnsetupUser(cfg.Client, user.UUID, &updatedUser); err != nil {
					log.Printf("error deactivating unlisted user %q: %s", user.UUID, err)
					updatesFailed[email] = true
				} else {
					allUsers[email] = updatedUser
					updatesSucceeded[email] = true
				}
			}
		}
	}

	log.Printf("User update successes: %d, skips: %d, failures: %d", len(updatesSucceeded), len(updatesSkipped), len(updatesFailed))

	// Report duplicated emails detection
	if len(dupedEmails) > 0 {
		emails := make([]string, len(dupedEmails))
		i := 0
		for e := range dupedEmails {
			emails[i] = e
			i++
		}
		return fmt.Errorf("skipped %d duplicated email address(es) in the cluster's local user list: %v", len(dupedEmails), emails)
	}

	return nil
}

func shouldSkip(cfg *ConfigParams, user arvados.User) bool {
	switch user.UUID {
	case cfg.SysUserUUID, cfg.AnonUserUUID:
		return true
	case cfg.CurrentUser.UUID:
		return true
	}
	return false
}

type userRecord struct {
	Email     string
	FirstName string
	LastName  string
	Active    bool
	Admin     bool
}

// ProcessRecord creates or updates a user based on the given record
func ProcessRecord(cfg *ConfigParams, record userRecord, allUsers map[string]arvados.User) (bool, error) {
	if cfg.Verbose {
		log.Printf("Processing record for user %q", record.Email)
	}

	wantedActiveStatus := strconv.FormatBool(record.Active)
	wantedAdminStatus := strconv.FormatBool(record.Admin)
	createRequired := false
	updateRequired := false
	// Check if user exists, set its active & admin status.
	var user arvados.User
	user, ok := allUsers[record.Email]
	if !ok {
		if cfg.Verbose {
			log.Printf("User %q does not exist, creating", record.Email)
		}
		createRequired = true
		err := CreateUser(cfg.Client, &user, map[string]string{
			"email":      record.Email,
			"first_name": record.FirstName,
			"last_name":  record.LastName,
			"is_active":  wantedActiveStatus,
			"is_admin":   wantedAdminStatus,
		})
		if err != nil {
			return false, fmt.Errorf("error creating user %q: %s", record.Email, err)
		}
	}
	if record.Active != user.IsActive {
		updateRequired = true
		if record.Active {
			if cfg.Verbose {
				log.Printf("User %q is inactive, activating", record.Email)
			}
			// Here we assume the 'setup' is done elsewhere if needed.
			err := UpdateUser(cfg.Client, user.UUID, &user, map[string]string{
				"is_active": wantedActiveStatus,
				"is_admin":  wantedAdminStatus, // Just in case it needs to be changed.
			})
			if err != nil {
				return false, fmt.Errorf("error updating user %q: %s", record.Email, err)
			}
		} else {
			if cfg.Verbose {
				log.Printf("User %q is active, deactivating", record.Email)
			}
			err := UnsetupUser(cfg.Client, user.UUID, &user)
			if err != nil {
				return false, fmt.Errorf("error deactivating user %q: %s", record.Email, err)
			}
		}
	}
	// Inactive users cannot be admins.
	if user.IsActive && record.Admin != user.IsAdmin {
		if cfg.Verbose {
			log.Printf("User %q is active, changing admin status to %v", record.Email, record.Admin)
		}
		updateRequired = true
		err := UpdateUser(cfg.Client, user.UUID, &user, map[string]string{
			"is_admin": wantedAdminStatus,
		})
		if err != nil {
			return false, fmt.Errorf("error updating user %q: %s", record.Email, err)
		}
	}
	allUsers[record.Email] = user
	if createRequired {
		log.Printf("Created user %q", record.Email)
	}
	if updateRequired {
		log.Printf("Updated user %q", record.Email)
	}

	return createRequired || updateRequired, nil
}

// LoadInputFile reads the input file and returns a list of user records
func LoadInputFile(f *os.File) (loadedRecords []userRecord, err error) {
	lineNo := 0
	csvReader := csv.NewReader(f)
	loadedRecords = make([]userRecord, 0)

	for {
		record, e := csvReader.Read()
		if e == io.EOF {
			break
		}
		lineNo++
		if e != nil {
			err = fmt.Errorf("parsing error at line %d: %s", lineNo, e)
			return
		}
		if len(record) != 5 {
			err = fmt.Errorf("parsing error at line %d: expected 5 fields, found %d", lineNo, len(record))
			return
		}
		email := strings.ToLower(strings.TrimSpace(record[0]))
		firstName := strings.TrimSpace(record[1])
		lastName := strings.TrimSpace(record[2])
		active := strings.TrimSpace(record[3])
		admin := strings.TrimSpace(record[4])
		if email == "" || firstName == "" || lastName == "" || active == "" || admin == "" {
			err = fmt.Errorf("parsing error at line %d: fields cannot be empty", lineNo)
			return
		}
		activeBool, err := strconv.ParseBool(active)
		if err != nil {
			return nil, fmt.Errorf("parsing error at line %d: active status not recognized", lineNo)
		}
		adminBool, err := strconv.ParseBool(admin)
		if err != nil {
			return nil, fmt.Errorf("parsing error at line %d: admin status not recognized", lineNo)
		}
		loadedRecords = append(loadedRecords, userRecord{
			Email:     email,
			FirstName: firstName,
			LastName:  lastName,
			Active:    activeBool,
			Admin:     adminBool,
		})
	}
	return loadedRecords, nil
}

// GetAll adds all objects of type 'resource' to the 'allItems' list
func GetAll(c *arvados.Client, res string, params arvados.ResourceListParams, page resourceList) (allItems []interface{}, err error) {
	// Use the maximum page size the server allows
	limit := 1<<31 - 1
	params.Limit = &limit
	params.Offset = 0
	params.Order = "uuid"
	for {
		if err = GetResourceList(c, &page, res, params); err != nil {
			return allItems, err
		}
		// Have we finished paging?
		if page.Len() == 0 {
			break
		}
		allItems = append(allItems, page.GetItems()...)
		params.Offset += page.Len()
	}
	return allItems, nil
}

func jsonReader(rscName string, ob interface{}) io.Reader {
	j, err := json.Marshal(ob)
	if err != nil {
		panic(err)
	}
	v := url.Values{}
	v[rscName] = []string{string(j)}
	return bytes.NewBufferString(v.Encode())
}

// GetResourceList fetches res list using params
func GetResourceList(c *arvados.Client, dst *resourceList, res string, params interface{}) error {
	return c.RequestAndDecode(dst, "GET", "/arvados/v1/"+res, nil, params)
}

// CreateUser creates a user with userData parameters, assigns it to dst
func CreateUser(c *arvados.Client, dst *arvados.User, userData map[string]string) error {
	return c.RequestAndDecode(dst, "POST", "/arvados/v1/users", jsonReader("user", userData), nil)
}

// UpdateUser updates a user with userData parameters
func UpdateUser(c *arvados.Client, userUUID string, dst *arvados.User, userData map[string]string) error {
	return c.RequestAndDecode(&dst, "PUT", "/arvados/v1/users/"+userUUID, jsonReader("user", userData), nil)
}

// UnsetupUser deactivates a user
func UnsetupUser(c *arvados.Client, userUUID string, dst *arvados.User) error {
	return c.RequestAndDecode(&dst, "POST", "/arvados/v1/users/"+userUUID+"/unsetup", nil, nil)
}
