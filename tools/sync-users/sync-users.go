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
	Path    string
	Verbose bool
	Client  *arvados.Client
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
		return cfg, fmt.Errorf("current user (%s) is not an admin user", u.UUID)
	}

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
	results, err := GetAll(cfg.Client, "users", arvados.ResourceListParams{}, &UserList{})
	if err != nil {
		return fmt.Errorf("error getting all users: %s", err)
	}
	log.Printf("Found %d users", len(results))
	for _, item := range results {
		u := item.(arvados.User)
		allUsers[strings.ToLower(u.Email)] = u
	}

	loadedRecords, err := LoadInputFile(f)
	if err != nil {
		return fmt.Errorf("reading input file %q: %s", cfg.Path, err)
	}
	log.Printf("Loaded %d records from input file", len(loadedRecords))

	updatesSucceeded, updatesFailed := 0, 0
	for _, record := range loadedRecords {
		if updated, err := ProcessRecord(cfg, record, allUsers); err != nil {
			log.Printf("error processing record %q: %s", record.Email, err)
			updatesFailed++
		} else if updated {
			updatesSucceeded++
		}
	}
	log.Printf("Updated %d account(s), failed to update %d account(s)", updatesSucceeded, updatesFailed)

	return nil
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
	wantedActiveStatus := strconv.FormatBool(record.Active)
	wantedAdminStatus := strconv.FormatBool(record.Admin)
	updateRequired := false
	// Check if user exists, set its active & admin status.
	var user arvados.User
	user, ok := allUsers[record.Email]
	if !ok {
		err := CreateUser(cfg.Client, &user, map[string]string{
			"email":      record.Email,
			"first_name": record.FirstName,
			"last_name":  record.LastName,
			"is_active":  strconv.FormatBool(record.Active),
			"is_admin":   strconv.FormatBool(record.Admin),
		})
		if err != nil {
			return false, fmt.Errorf("error creating user %q: %s", record.Email, err)
		}
		updateRequired = true
		log.Printf("Created user %q", record.Email)
	}
	if record.Active != user.IsActive {
		updateRequired = true
		if record.Active {
			// Here we assume the 'setup' is done elsewhere if needed.
			err := UpdateUser(cfg.Client, user.UUID, &user, map[string]string{
				"is_active": wantedActiveStatus,
				"is_admin":  wantedAdminStatus, // Just in case it needs to be changed.
			})
			if err != nil {
				return false, fmt.Errorf("error updating user %q: %s", record.Email, err)
			}
		} else {
			err := UnsetupUser(cfg.Client, user.UUID, &user)
			if err != nil {
				return false, fmt.Errorf("error deactivating user %q: %s", record.Email, err)
			}
		}
	}
	// Inactive users cannot be admins.
	if user.IsActive && record.Admin != user.IsAdmin {
		updateRequired = true
		err := UpdateUser(cfg.Client, user.UUID, &user, map[string]string{
			"is_admin": wantedAdminStatus,
		})
		if err != nil {
			return false, fmt.Errorf("error updating user %q: %s", record.Email, err)
		}
	}
	allUsers[record.Email] = user
	if updateRequired {
		log.Printf("Updated user %q", record.Email)
	}

	return updateRequired, nil
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
