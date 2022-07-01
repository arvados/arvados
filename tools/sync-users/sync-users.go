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
	CaseInsensitive    bool
	Client             *arvados.Client
	ClusterID          string
	CurrentUser        arvados.User
	DeactivateUnlisted bool
	Path               string
	UserID             string
	SysUserUUID        string
	AnonUserUUID       string
	Verbose            bool
}

func ParseFlags(cfg *ConfigParams) error {
	// Acceptable attributes to identify a user on the CSV file
	userIDOpts := map[string]bool{
		"email":    true, // default
		"username": true,
	}

	flags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flags.Usage = func() {
		usageStr := `Synchronize remote users into Arvados from a CSV format file with 5 columns:
  * 1st: User Identifier (email or username)
  * 2nd: First name
  * 3rd: Last name
  * 4th: Active status (0 or 1)
  * 5th: Admin status (0 or 1)`
		fmt.Fprintf(flags.Output(), "%s\n\n", usageStr)
		fmt.Fprintf(flags.Output(), "Usage:\n%s [OPTIONS] <input-file.csv>\n\n", os.Args[0])
		fmt.Fprintf(flags.Output(), "Options:\n")
		flags.PrintDefaults()
	}

	caseInsensitive := flags.Bool(
		"case-insensitive",
		true,
		"Performs case insensitive matching on user IDs. Always ON whe using 'email' user IDs.")
	deactivateUnlisted := flags.Bool(
		"deactivate-unlisted",
		false,
		"Deactivate users that are not in the input file.")
	userID := flags.String(
		"user-id",
		"email",
		"Attribute by which every user is identified. Valid values are: email and username.")
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
	if !userIDOpts[*userID] {
		var options []string
		for opt := range userIDOpts {
			options = append(options, opt)
		}
		return fmt.Errorf("user ID must be one of: %s", strings.Join(options, ", "))
	}
	if *userID == "email" {
		// Always do case-insensitive email addresses matching
		*caseInsensitive = true
	}

	cfg.CaseInsensitive = *caseInsensitive
	cfg.DeactivateUnlisted = *deactivateUnlisted
	cfg.Path = *srcPath
	cfg.UserID = *userID
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

// GetUserID returns the correct user id value depending on the selector
func GetUserID(u arvados.User, idSelector string) (string, error) {
	switch idSelector {
	case "email":
		return u.Email, nil
	case "username":
		return u.Username, nil
	default:
		return "", fmt.Errorf("cannot identify user by %q selector", idSelector)
	}
}

func doMain(cfg *ConfigParams) error {
	// Try opening the input file early, just in case there's a problem.
	f, err := os.Open(cfg.Path)
	if err != nil {
		return fmt.Errorf("error opening input file: %s", err)
	}
	defer f.Close()

	iCaseLog := ""
	if cfg.UserID == "username" && cfg.CaseInsensitive {
		iCaseLog = " - username matching requested to be case-insensitive"
	}
	log.Printf("%s %s started. Using %q as users id%s", os.Args[0], version, cfg.UserID, iCaseLog)

	allUsers := make(map[string]arvados.User)
	userIDToUUID := make(map[string]string) // Index by email or username
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

		// Duplicated user id check
		uID, err := GetUserID(u, cfg.UserID)
		if err != nil {
			return err
		}
		if uID == "" {
			return fmt.Errorf("%s is empty for user with uuid %q", cfg.UserID, u.UUID)
		}
		if cfg.CaseInsensitive {
			uID = strings.ToLower(uID)
		}
		if alreadySeenUUID, found := userIDToUUID[uID]; found {
			if cfg.UserID == "username" && uID != "" {
				return fmt.Errorf("case insensitive collision for username %q between %q and %q", uID, u.UUID, alreadySeenUUID)
			} else if cfg.UserID == "email" && uID != "" {
				log.Printf("Duplicated email %q found in user %s - ignoring", uID, u.UUID)
				if len(dupedEmails[uID]) == 0 {
					dupedEmails[uID] = []arvados.User{allUsers[alreadySeenUUID]}
				}
				dupedEmails[uID] = append(dupedEmails[uID], u)
				delete(allUsers, alreadySeenUUID) // Skip even the first occurrence,
				// for security purposes.
				continue
			}
		}
		if cfg.Verbose {
			log.Printf("Seen user %q (%s)", uID, u.UUID)
		}
		userIDToUUID[uID] = u.UUID
		allUsers[u.UUID] = u
		processedUsers[u.UUID] = false
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
		if cfg.CaseInsensitive {
			record.UserID = strings.ToLower(record.UserID)
		}
		recordUUID := userIDToUUID[record.UserID]
		processedUsers[recordUUID] = true
		if cfg.UserID == "email" && record.UserID == cfg.CurrentUser.Email {
			updatesSkipped[recordUUID] = true
			log.Printf("Skipping current user %q (%s) from processing", record.UserID, cfg.CurrentUser.UUID)
			continue
		}
		if updated, err := ProcessRecord(cfg, record, userIDToUUID, allUsers); err != nil {
			log.Printf("error processing record %q: %s", record.UserID, err)
			updatesFailed[recordUUID] = true
		} else if updated {
			updatesSucceeded[recordUUID] = true
		}
	}

	if cfg.DeactivateUnlisted {
		for userUUID, user := range allUsers {
			if shouldSkip(cfg, user) {
				updatesSkipped[userUUID] = true
				log.Printf("Skipping unlisted user %q (%s) from deactivating", user.Email, user.UUID)
				continue
			}
			if !processedUsers[userUUID] && allUsers[userUUID].IsActive {
				if cfg.Verbose {
					log.Printf("Deactivating unlisted user %q (%s)", user.Username, user.UUID)
				}
				var updatedUser arvados.User
				if err := UnsetupUser(cfg.Client, user.UUID, &updatedUser); err != nil {
					log.Printf("error deactivating unlisted user %q: %s", user.UUID, err)
					updatesFailed[userUUID] = true
				} else {
					allUsers[userUUID] = updatedUser
					updatesSucceeded[userUUID] = true
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
	UserID    string
	FirstName string
	LastName  string
	Active    bool
	Admin     bool
}

// ProcessRecord creates or updates a user based on the given record
func ProcessRecord(cfg *ConfigParams, record userRecord, userIDToUUID map[string]string, allUsers map[string]arvados.User) (bool, error) {
	if cfg.Verbose {
		log.Printf("Processing record for user %q", record.UserID)
	}

	wantedActiveStatus := strconv.FormatBool(record.Active)
	wantedAdminStatus := strconv.FormatBool(record.Admin)
	createRequired := false
	updateRequired := false
	// Check if user exists, set its active & admin status.
	var user arvados.User
	recordUUID := userIDToUUID[record.UserID]
	user, ok := allUsers[recordUUID]
	if !ok {
		if cfg.Verbose {
			log.Printf("User %q does not exist, creating", record.UserID)
		}
		createRequired = true
		err := CreateUser(cfg.Client, &user, map[string]string{
			cfg.UserID:   record.UserID,
			"first_name": record.FirstName,
			"last_name":  record.LastName,
			"is_active":  wantedActiveStatus,
			"is_admin":   wantedAdminStatus,
		})
		if err != nil {
			return false, fmt.Errorf("error creating user %q: %s", record.UserID, err)
		}
	}
	if record.Active != user.IsActive {
		updateRequired = true
		if record.Active {
			if cfg.Verbose {
				log.Printf("User %q is inactive, activating", record.UserID)
			}
			// Here we assume the 'setup' is done elsewhere if needed.
			err := UpdateUser(cfg.Client, user.UUID, &user, map[string]string{
				"is_active": wantedActiveStatus,
				"is_admin":  wantedAdminStatus, // Just in case it needs to be changed.
			})
			if err != nil {
				return false, fmt.Errorf("error updating user %q: %s", record.UserID, err)
			}
		} else {
			if cfg.Verbose {
				log.Printf("User %q is active, deactivating", record.UserID)
			}
			err := UnsetupUser(cfg.Client, user.UUID, &user)
			if err != nil {
				return false, fmt.Errorf("error deactivating user %q: %s", record.UserID, err)
			}
		}
	}
	// Inactive users cannot be admins.
	if user.IsActive && record.Admin != user.IsAdmin {
		if cfg.Verbose {
			log.Printf("User %q is active, changing admin status to %v", record.UserID, record.Admin)
		}
		updateRequired = true
		err := UpdateUser(cfg.Client, user.UUID, &user, map[string]string{
			"is_admin": wantedAdminStatus,
		})
		if err != nil {
			return false, fmt.Errorf("error updating user %q: %s", record.UserID, err)
		}
	}
	allUsers[record.UserID] = user
	if createRequired {
		log.Printf("Created user %q", record.UserID)
	}
	if updateRequired {
		log.Printf("Updated user %q", record.UserID)
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
		userID := strings.ToLower(strings.TrimSpace(record[0]))
		firstName := strings.TrimSpace(record[1])
		lastName := strings.TrimSpace(record[2])
		active := strings.TrimSpace(record[3])
		admin := strings.TrimSpace(record[4])
		if userID == "" || firstName == "" || lastName == "" || active == "" || admin == "" {
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
			UserID:    userID,
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
