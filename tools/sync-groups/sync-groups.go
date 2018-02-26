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
	"strings"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
)

var version = "dev"

type resourceList interface {
	Len() int
	GetItems() []interface{}
}

// GroupInfo tracks previous and current members of a particular Group
type GroupInfo struct {
	Group           arvados.Group
	PreviousMembers map[string]bool
	CurrentMembers  map[string]bool
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

// GroupList implements resourceList interface
type GroupList struct {
	arvados.GroupList
}

// Len returns the amount of items this list holds
func (l GroupList) Len() int {
	return len(l.Items)
}

// GetItems returns the list of items
func (l GroupList) GetItems() (out []interface{}) {
	for _, item := range l.Items {
		out = append(out, item)
	}
	return
}

// LinkList implements resourceList interface
type LinkList struct {
	arvados.LinkList
}

// Len returns the amount of items this list holds
func (l LinkList) Len() int {
	return len(l.Items)
}

// GetItems returns the list of items
func (l LinkList) GetItems() (out []interface{}) {
	for _, item := range l.Items {
		out = append(out, item)
	}
	return
}

func main() {
	// Parse & validate arguments, set up arvados client.
	cfg, err := GetConfig()
	if err != nil {
		log.Fatalf("%v", err)
	}

	if err := doMain(&cfg); err != nil {
		log.Fatalf("%v", err)
	}
}

// ConfigParams holds configuration data for this tool
type ConfigParams struct {
	Path            string
	UserID          string
	Verbose         bool
	ParentGroupUUID string
	ParentGroupName string
	SysUserUUID     string
	Client          *arvados.Client
}

// ParseFlags parses and validates command line arguments
func ParseFlags(config *ConfigParams) error {
	// Acceptable attributes to identify a user on the CSV file
	userIDOpts := map[string]bool{
		"email":    true, // default
		"username": true,
	}

	flags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	// Set up usage message
	flags.Usage = func() {
		usageStr := `Synchronize remote groups into Arvados from a CSV format file with 2 columns:
  * 1st column: Group name
  * 2nd column: User identifier`
		fmt.Fprintf(os.Stderr, "%s\n\n", usageStr)
		fmt.Fprintf(os.Stderr, "Usage:\n%s [OPTIONS] <input-file.csv>\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flags.PrintDefaults()
	}

	// Set up option flags
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
	parentGroupUUID := flags.String(
		"parent-group-uuid",
		"",
		"Use given group UUID as a parent for the remote groups. Should be owned by the system user. If not specified, a group named '"+config.ParentGroupName+"' will be used (and created if nonexistant).")

	// Parse args; omit the first arg which is the command name
	flags.Parse(os.Args[1:])

	// Print version information if requested
	if *getVersion {
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

	config.Path = *srcPath
	config.ParentGroupUUID = *parentGroupUUID
	config.UserID = *userID
	config.Verbose = *verbose

	return nil
}

// SetParentGroup finds/create parent group of all remote groups
func SetParentGroup(cfg *ConfigParams) error {
	var parentGroup arvados.Group
	if cfg.ParentGroupUUID == "" {
		// UUID not provided, search for preexisting parent group
		var gl GroupList
		params := arvados.ResourceListParams{
			Filters: []arvados.Filter{{
				Attr:     "name",
				Operator: "=",
				Operand:  cfg.ParentGroupName,
			}, {
				Attr:     "owner_uuid",
				Operator: "=",
				Operand:  cfg.SysUserUUID,
			}},
		}
		if err := cfg.Client.RequestAndDecode(&gl, "GET", "/arvados/v1/groups", nil, params); err != nil {
			return fmt.Errorf("error searching for parent group: %s", err)
		}
		if len(gl.Items) == 0 {
			// Default parent group does not exist, create it.
			if cfg.Verbose {
				log.Println("Default parent group not found, creating...")
			}
			groupData := map[string]string{
				"name":       cfg.ParentGroupName,
				"owner_uuid": cfg.SysUserUUID,
			}
			if err := CreateGroup(cfg, &parentGroup, groupData); err != nil {
				return fmt.Errorf("error creating system user owned group named %q: %s", groupData["name"], err)
			}
		} else if len(gl.Items) == 1 {
			// Default parent group found.
			parentGroup = gl.Items[0]
		} else {
			// This should never happen, as there's an unique index for
			// (owner_uuid, name) on groups.
			return fmt.Errorf("bug: found %d groups owned by system user and named %q", len(gl.Items), cfg.ParentGroupName)
		}
		cfg.ParentGroupUUID = parentGroup.UUID
	} else {
		// UUID provided. Check if exists and if it's owned by system user
		if err := GetGroup(cfg, &parentGroup, cfg.ParentGroupUUID); err != nil {
			return fmt.Errorf("error searching for parent group with UUID %q: %s", cfg.ParentGroupUUID, err)
		}
		if parentGroup.OwnerUUID != cfg.SysUserUUID {
			return fmt.Errorf("parent group %q (%s) must be owned by system user", parentGroup.Name, cfg.ParentGroupUUID)
		}
	}
	return nil
}

// GetConfig sets up a ConfigParams struct
func GetConfig() (config ConfigParams, err error) {
	config.ParentGroupName = "Externally synchronized groups"

	// Command arguments
	err = ParseFlags(&config)
	if err != nil {
		return config, err
	}

	// Arvados Client setup
	config.Client = arvados.NewClientFromEnv()

	// Check current user permissions & get System user's UUID
	u, err := config.Client.CurrentUser()
	if err != nil {
		return config, fmt.Errorf("error getting the current user: %s", err)
	}
	if !u.IsActive || !u.IsAdmin {
		return config, fmt.Errorf("current user (%s) is not an active admin user", u.UUID)
	}
	config.SysUserUUID = u.UUID[:12] + "000000000000000"

	// Set up remote groups' parent
	if err = SetParentGroup(&config); err != nil {
		return config, err
	}

	return config, nil
}

func doMain(cfg *ConfigParams) error {
	// Try opening the input file early, just in case there's a problem.
	f, err := os.Open(cfg.Path)
	if err != nil {
		return fmt.Errorf("%s", err)
	}
	defer f.Close()

	log.Printf("%s %s started. Using %q as users id and parent group UUID %q", os.Args[0], version, cfg.UserID, cfg.ParentGroupUUID)

	// Get the complete user list to minimize API Server requests
	allUsers := make(map[string]arvados.User)
	userIDToUUID := make(map[string]string) // Index by email or username
	results, err := GetAll(cfg.Client, "users", arvados.ResourceListParams{}, &UserList{})
	if err != nil {
		return fmt.Errorf("error getting user list: %s", err)
	}
	log.Printf("Found %d users", len(results))
	for _, item := range results {
		u := item.(arvados.User)
		allUsers[u.UUID] = u
		uID, err := GetUserID(u, cfg.UserID)
		if err != nil {
			return err
		}
		userIDToUUID[uID] = u.UUID
		if cfg.Verbose {
			log.Printf("Seen user %q (%s)", u.Username, u.UUID)
		}
	}

	// Get remote groups and their members
	remoteGroups, groupNameToUUID, err := GetRemoteGroups(cfg, allUsers)
	if err != nil {
		return err
	}
	log.Printf("Found %d remote groups", len(remoteGroups))
	if cfg.Verbose {
		for groupUUID := range remoteGroups {
			log.Printf("- Group %q: %d users", remoteGroups[groupUUID].Group.Name, len(remoteGroups[groupUUID].PreviousMembers))
		}
	}

	membershipsRemoved := 0

	// Read the CSV file
	groupsCreated, membershipsAdded, membershipsSkipped, err := ProcessFile(cfg, f, userIDToUUID, groupNameToUUID, remoteGroups, allUsers)
	if err != nil {
		return err
	}

	// Remove previous members not listed on this run
	for groupUUID := range remoteGroups {
		gi := remoteGroups[groupUUID]
		evictedMembers := subtract(gi.PreviousMembers, gi.CurrentMembers)
		groupName := gi.Group.Name
		if len(evictedMembers) > 0 {
			log.Printf("Removing %d users from group %q", len(evictedMembers), groupName)
		}
		for evictedUser := range evictedMembers {
			if err := RemoveMemberFromGroup(cfg, allUsers[userIDToUUID[evictedUser]], gi.Group); err != nil {
				return err
			}
			membershipsRemoved++
		}
	}
	log.Printf("Groups created: %d. Memberships added: %d, removed: %d, skipped: %d", groupsCreated, membershipsAdded, membershipsRemoved, membershipsSkipped)

	return nil
}

// ProcessFile reads the CSV file and process every record
func ProcessFile(
	cfg *ConfigParams,
	f *os.File,
	userIDToUUID map[string]string,
	groupNameToUUID map[string]string,
	remoteGroups map[string]*GroupInfo,
	allUsers map[string]arvados.User,
) (groupsCreated, membersAdded, membersSkipped int, err error) {
	lineNo := 0
	csvReader := csv.NewReader(f)
	csvReader.FieldsPerRecord = 2
	for {
		record, e := csvReader.Read()
		if e == io.EOF {
			break
		}
		lineNo++
		if e != nil {
			err = fmt.Errorf("error parsing %q, line %d", cfg.Path, lineNo)
			return
		}
		groupName := strings.TrimSpace(record[0])
		groupMember := strings.TrimSpace(record[1]) // User ID (username or email)
		if groupName == "" || groupMember == "" {
			log.Printf("Warning: CSV record has at least one empty field (%s, %s). Skipping", groupName, groupMember)
			membersSkipped++
			continue
		}
		if _, found := userIDToUUID[groupMember]; !found {
			// User not present on the system, skip.
			log.Printf("Warning: there's no user with %s %q on the system, skipping.", cfg.UserID, groupMember)
			membersSkipped++
			continue
		}
		if _, found := groupNameToUUID[groupName]; !found {
			// Group doesn't exist, create it before continuing
			if cfg.Verbose {
				log.Printf("Remote group %q not found, creating...", groupName)
			}
			var newGroup arvados.Group
			groupData := map[string]string{
				"name":        groupName,
				"owner_uuid":  cfg.ParentGroupUUID,
				"group_class": "role",
			}
			if e := CreateGroup(cfg, &newGroup, groupData); e != nil {
				err = fmt.Errorf("error creating group named %q: %s", groupName, err)
				return
			}
			// Update cached group data
			groupNameToUUID[groupName] = newGroup.UUID
			remoteGroups[newGroup.UUID] = &GroupInfo{
				Group:           newGroup,
				PreviousMembers: make(map[string]bool), // Empty set
				CurrentMembers:  make(map[string]bool), // Empty set
			}
			groupsCreated++
		}
		// Both group & user exist, check if user is a member
		groupUUID := groupNameToUUID[groupName]
		gi := remoteGroups[groupUUID]
		if !gi.PreviousMembers[groupMember] && !gi.CurrentMembers[groupMember] {
			if cfg.Verbose {
				log.Printf("Adding %q to group %q", groupMember, groupName)
			}
			// User wasn't a member, but should be.
			if e := AddMemberToGroup(cfg, allUsers[userIDToUUID[groupMember]], gi.Group); e != nil {
				err = e
				return
			}
			membersAdded++
		}
		gi.CurrentMembers[groupMember] = true
	}
	return
}

// GetAll : Adds all objects of type 'resource' to the 'allItems' list
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
		for _, i := range page.GetItems() {
			allItems = append(allItems, i)
		}
		params.Offset += page.Len()
	}
	return allItems, nil
}

func subtract(setA map[string]bool, setB map[string]bool) map[string]bool {
	result := make(map[string]bool)
	for element := range setA {
		if !setB[element] {
			result[element] = true
		}
	}
	return result
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

// GetRemoteGroups fetches all remote groups with their members
func GetRemoteGroups(cfg *ConfigParams, allUsers map[string]arvados.User) (remoteGroups map[string]*GroupInfo, groupNameToUUID map[string]string, err error) {
	remoteGroups = make(map[string]*GroupInfo)
	groupNameToUUID = make(map[string]string) // Index by group name

	params := arvados.ResourceListParams{
		Filters: []arvados.Filter{{
			Attr:     "owner_uuid",
			Operator: "=",
			Operand:  cfg.ParentGroupUUID,
		}},
	}
	results, err := GetAll(cfg.Client, "groups", params, &GroupList{})
	if err != nil {
		return remoteGroups, groupNameToUUID, fmt.Errorf("error getting remote groups: %s", err)
	}
	for _, item := range results {
		group := item.(arvados.Group)
		// Group -> User filter
		g2uFilter := arvados.ResourceListParams{
			Filters: []arvados.Filter{{
				Attr:     "owner_uuid",
				Operator: "=",
				Operand:  cfg.SysUserUUID,
			}, {
				Attr:     "link_class",
				Operator: "=",
				Operand:  "permission",
			}, {
				Attr:     "name",
				Operator: "=",
				Operand:  "can_read",
			}, {
				Attr:     "tail_uuid",
				Operator: "=",
				Operand:  group.UUID,
			}, {
				Attr:     "head_uuid",
				Operator: "like",
				Operand:  "%-tpzed-%",
			}},
		}
		// User -> Group filter
		u2gFilter := arvados.ResourceListParams{
			Filters: []arvados.Filter{{
				Attr:     "owner_uuid",
				Operator: "=",
				Operand:  cfg.SysUserUUID,
			}, {
				Attr:     "link_class",
				Operator: "=",
				Operand:  "permission",
			}, {
				Attr:     "name",
				Operator: "=",
				Operand:  "can_write",
			}, {
				Attr:     "head_uuid",
				Operator: "=",
				Operand:  group.UUID,
			}, {
				Attr:     "tail_uuid",
				Operator: "like",
				Operand:  "%-tpzed-%",
			}},
		}
		g2uLinks, err := GetAll(cfg.Client, "links", g2uFilter, &LinkList{})
		if err != nil {
			return remoteGroups, groupNameToUUID, fmt.Errorf("error getting member (can_read) links for group %q: %s", group.Name, err)
		}
		u2gLinks, err := GetAll(cfg.Client, "links", u2gFilter, &LinkList{})
		if err != nil {
			return remoteGroups, groupNameToUUID, fmt.Errorf("error getting member (can_write) links for group %q: %s", group.Name, err)
		}
		// Build a list of user ids (email or username) belonging to this group
		membersSet := make(map[string]bool)
		u2gLinkSet := make(map[string]bool)
		for _, l := range u2gLinks {
			linkedMemberUUID := l.(arvados.Link).TailUUID
			u2gLinkSet[linkedMemberUUID] = true
		}
		for _, item := range g2uLinks {
			link := item.(arvados.Link)
			// We may have received an old link pointing to a removed account.
			if _, found := allUsers[link.HeadUUID]; !found {
				continue
			}
			// The matching User -> Group link may not exist if the link
			// creation failed on a previous run. If that's the case, don't
			// include this account on the "previous members" list.
			if _, found := u2gLinkSet[link.HeadUUID]; !found {
				continue
			}
			memberID, err := GetUserID(allUsers[link.HeadUUID], cfg.UserID)
			if err != nil {
				return remoteGroups, groupNameToUUID, err
			}
			membersSet[memberID] = true
		}
		remoteGroups[group.UUID] = &GroupInfo{
			Group:           group,
			PreviousMembers: membersSet,
			CurrentMembers:  make(map[string]bool), // Empty set
		}
		groupNameToUUID[group.Name] = group.UUID
	}
	return remoteGroups, groupNameToUUID, nil
}

// RemoveMemberFromGroup remove all links related to the membership
func RemoveMemberFromGroup(cfg *ConfigParams, user arvados.User, group arvados.Group) error {
	if cfg.Verbose {
		log.Printf("Getting group membership links for user %q (%s) on group %q (%s)", user.Username, user.UUID, group.Name, group.UUID)
	}
	var links []interface{}
	// Search for all group<->user links (both ways)
	for _, filterset := range [][]arvados.Filter{
		// Group -> User
		{{
			Attr:     "link_class",
			Operator: "=",
			Operand:  "permission",
		}, {
			Attr:     "tail_uuid",
			Operator: "=",
			Operand:  group.UUID,
		}, {
			Attr:     "head_uuid",
			Operator: "=",
			Operand:  user.UUID,
		}},
		// Group <- User
		{{
			Attr:     "link_class",
			Operator: "=",
			Operand:  "permission",
		}, {
			Attr:     "tail_uuid",
			Operator: "=",
			Operand:  user.UUID,
		}, {
			Attr:     "head_uuid",
			Operator: "=",
			Operand:  group.UUID,
		}},
	} {
		l, err := GetAll(cfg.Client, "links", arvados.ResourceListParams{Filters: filterset}, &LinkList{})
		if err != nil {
			userID, _ := GetUserID(user, cfg.UserID)
			return fmt.Errorf("error getting links needed to remove user %q from group %q: %s", userID, group.Name, err)
		}
		for _, link := range l {
			links = append(links, link)
		}
	}
	for _, item := range links {
		link := item.(arvados.Link)
		userID, _ := GetUserID(user, cfg.UserID)
		if cfg.Verbose {
			log.Printf("Removing %q permission link for %q on group %q", link.Name, userID, group.Name)
		}
		if err := DeleteLink(cfg, link.UUID); err != nil {
			return fmt.Errorf("error removing user %q from group %q: %s", userID, group.Name, err)
		}
	}
	return nil
}

// AddMemberToGroup create membership links
func AddMemberToGroup(cfg *ConfigParams, user arvados.User, group arvados.Group) error {
	var newLink arvados.Link
	linkData := map[string]string{
		"owner_uuid": cfg.SysUserUUID,
		"link_class": "permission",
		"name":       "can_read",
		"tail_uuid":  group.UUID,
		"head_uuid":  user.UUID,
	}
	if err := CreateLink(cfg, &newLink, linkData); err != nil {
		userID, _ := GetUserID(user, cfg.UserID)
		return fmt.Errorf("error adding group %q -> user %q read permission: %s", group.Name, userID, err)
	}
	linkData = map[string]string{
		"owner_uuid": cfg.SysUserUUID,
		"link_class": "permission",
		"name":       "can_write",
		"tail_uuid":  user.UUID,
		"head_uuid":  group.UUID,
	}
	if err := CreateLink(cfg, &newLink, linkData); err != nil {
		userID, _ := GetUserID(user, cfg.UserID)
		return fmt.Errorf("error adding user %q -> group %q write permission: %s", userID, group.Name, err)
	}
	return nil
}

// CreateGroup creates a group with groupData parameters, assigns it to dst
func CreateGroup(cfg *ConfigParams, dst *arvados.Group, groupData map[string]string) error {
	return cfg.Client.RequestAndDecode(dst, "POST", "/arvados/v1/groups", jsonReader("group", groupData), nil)
}

// GetGroup fetches a group by its UUID
func GetGroup(cfg *ConfigParams, dst *arvados.Group, groupUUID string) error {
	return cfg.Client.RequestAndDecode(&dst, "GET", "/arvados/v1/groups/"+groupUUID, nil, nil)
}

// CreateLink creates a link with linkData parameters, assigns it to dst
func CreateLink(cfg *ConfigParams, dst *arvados.Link, linkData map[string]string) error {
	return cfg.Client.RequestAndDecode(dst, "POST", "/arvados/v1/links", jsonReader("link", linkData), nil)
}

// DeleteLink deletes a link by its UUID
func DeleteLink(cfg *ConfigParams, linkUUID string) error {
	if linkUUID == "" {
		return fmt.Errorf("cannot delete link with invalid UUID: %q", linkUUID)
	}
	return cfg.Client.RequestAndDecode(&arvados.Link{}, "DELETE", "/arvados/v1/links/"+linkUUID, nil, nil)
}

// GetResourceList fetches res list using params
func GetResourceList(c *arvados.Client, dst *resourceList, res string, params interface{}) error {
	return c.RequestAndDecode(dst, "GET", "/arvados/v1/"+res, nil, params)
}
