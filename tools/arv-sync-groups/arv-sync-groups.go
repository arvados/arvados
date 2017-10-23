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

type resourceList interface {
	Len() int
	GetItems() []interface{}
}

type groupInfo struct {
	Group           Group
	PreviousMembers map[string]bool
	CurrentMembers  map[string]bool
}

type user struct {
	UUID     string `json:"uuid,omitempty"`
	Email    string `json:"email,omitempty"`
	Username string `json:"username,omitempty"`
}

func (u user) GetID(idSelector string) (string, error) {
	switch idSelector {
	case "email":
		return u.Email, nil
	case "username":
		return u.Username, nil
	default:
		return "", fmt.Errorf("cannot identify user by %q selector", idSelector)
	}
}

// userList implements resourceList interface
type userList struct {
	Items []user `json:"items"`
}

func (l userList) Len() int {
	return len(l.Items)
}

func (l userList) GetItems() (out []interface{}) {
	for _, item := range l.Items {
		out = append(out, item)
	}
	return
}

// Group is an arvados#group record
type Group struct {
	UUID      string `json:"uuid,omitempty"`
	Name      string `json:"name,omitempty"`
	OwnerUUID string `json:"owner_uuid,omitempty"`
}

// GroupList implements resourceList interface
type GroupList struct {
	Items []Group `json:"items"`
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

// Link is an arvados#link record
type Link struct {
	UUID      string `json:"uuid,omiempty"`
	OwnerUUID string `json:"owner_uuid,omitempty"`
	Name      string `json:"name,omitempty"`
	LinkClass string `json:"link_class,omitempty"`
	HeadUUID  string `json:"head_uuid,omitempty"`
	HeadKind  string `json:"head_kind,omitempty"`
	TailUUID  string `json:"tail_uuid,omitempty"`
	TailKind  string `json:"tail_kind,omitempty"`
}

// LinkList implements resourceList interface
type linkList struct {
	Items []Link `json:"items"`
}

// Len returns the amount of items this list holds
func (l linkList) Len() int {
	return len(l.Items)
}

// GetItems returns the list of items
func (l linkList) GetItems() (out []interface{}) {
	for _, item := range l.Items {
		out = append(out, item)
	}
	return
}

func main() {
	if err := doMain(); err != nil {
		log.Fatalf("%v", err)
	}
}

func doMain() error {
	const remoteGroupParentName string = "Externally synchronized groups"
	// Acceptable attributes to identify a user on the CSV file
	userIDOpts := map[string]bool{
		"email":    true, // default
		"username": true,
	}

	// Command arguments
	flags := flag.NewFlagSet("arv-sync-groups", flag.ExitOnError)
	srcPath := flags.String(
		"path",
		"",
		"Local file path containing a CSV format: GroupName,UserID")
	userID := flags.String(
		"user-id",
		"email",
		"Attribute by which every user is identified. Valid values are: email (the default) and username.")
	verbose := flags.Bool(
		"verbose",
		false,
		"Log informational messages. Off by default.")
	parentGroupUUID := flags.String(
		"parent-group-uuid",
		"",
		"Use given group UUID as a parent for the remote groups. Should be owned by the system user. If not specified, a group named '"+remoteGroupParentName+"' will be used (and created if nonexistant).")

	// Parse args; omit the first arg which is the command name
	flags.Parse(os.Args[1:])

	// Validations
	if *srcPath == "" {
		return fmt.Errorf("please provide a path to an input file")
	}
	if !userIDOpts[*userID] {
		var options []string
		for opt := range userIDOpts {
			options = append(options, opt)
		}
		return fmt.Errorf("user ID must be one of: %s", strings.Join(options, ", "))
	}

	// Try opening the input file early, just in case there's problems.
	f, err := os.Open(*srcPath)
	if err != nil {
		return fmt.Errorf("%s", err)
	}
	defer f.Close()

	// Arvados Client setup
	ac := arvados.NewClientFromEnv()

	// Check current user permissions & get System user's UUID
	u, err := ac.CurrentUser()
	if err != nil {
		return fmt.Errorf("error getting the current user: %s", err)
	}
	if !u.IsActive || !u.IsAdmin {
		return fmt.Errorf("current user (%s) is not an active admin user", u.UUID)
	}
	sysUserUUID := u.UUID[:12] + "000000000000000"

	// Find/create parent group
	var parentGroup Group
	if *parentGroupUUID == "" {
		// UUID not provided, search for preexisting parent group
		var gl GroupList
		params := arvados.ResourceListParams{
			Filters: []arvados.Filter{{
				Attr:     "name",
				Operator: "=",
				Operand:  remoteGroupParentName,
			}, {
				Attr:     "owner_uuid",
				Operator: "=",
				Operand:  sysUserUUID,
			}},
		}
		if err := ac.RequestAndDecode(&gl, "GET", "/arvados/v1/groups", nil, params); err != nil {
			return fmt.Errorf("error searching for parent group: %s", err)
		}
		if len(gl.Items) == 0 {
			// Default parent group not existant, create one.
			if *verbose {
				log.Println("Default parent group not found, creating...")
			}
			groupData := map[string]string{
				"name":       remoteGroupParentName,
				"owner_uuid": sysUserUUID,
			}
			if err := ac.RequestAndDecode(&parentGroup, "POST", "/arvados/v1/groups", jsonReader("group", groupData), nil); err != nil {
				return fmt.Errorf("error creating system user owned group named %q: %s", groupData["name"], err)
			}
		} else if len(gl.Items) == 1 {
			// Default parent group found.
			parentGroup = gl.Items[0]
		} else {
			// This should never happen, as there's an unique index for
			// (owner_uuid, name) on groups.
			return fmt.Errorf("bug: found %d groups owned by system user and named %q", len(gl.Items), remoteGroupParentName)
		}
	} else {
		// UUID provided. Check if exists and if it's owned by system user
		if err := ac.RequestAndDecode(&parentGroup, "GET", "/arvados/v1/groups/"+*parentGroupUUID, nil, nil); err != nil {
			return fmt.Errorf("error searching for parent group with UUID %q: %s", *parentGroupUUID, err)
		}
		if parentGroup.OwnerUUID != sysUserUUID {
			return fmt.Errorf("parent group %q (%s) must be owned by system user", parentGroup.Name, *parentGroupUUID)
		}
	}

	log.Printf("Group sync starting. Using %q as users id and parent group UUID %q", *userID, parentGroup.UUID)

	// Get the complete user list to minimize API Server requests
	allUsers := make(map[string]user)
	userIDToUUID := make(map[string]string) // Index by email or username
	results, err := ListAll(ac, "users", arvados.ResourceListParams{}, &userList{})
	if err != nil {
		return fmt.Errorf("error getting user list: %s", err)
	}
	log.Printf("Found %d users", len(results))
	for _, item := range results {
		u := item.(user)
		allUsers[u.UUID] = u
		uID, err := u.GetID(*userID)
		if err != nil {
			return err
		}
		userIDToUUID[uID] = u.UUID
		if *verbose {
			log.Printf("Seen user %q (%s)", u.Username, u.Email)
		}
	}

	// Get remote groups and their members
	remoteGroups := make(map[string]*groupInfo)
	groupNameToUUID := make(map[string]string) // Index by group name
	params := arvados.ResourceListParams{
		Filters: []arvados.Filter{{
			Attr:     "owner_uuid",
			Operator: "=",
			Operand:  parentGroup.UUID,
		}},
	}
	results, err = ListAll(ac, "groups", params, &GroupList{})
	if err != nil {
		return fmt.Errorf("error getting remote groups: %s", err)
	}
	for _, item := range results {
		group := item.(Group)
		params := arvados.ResourceListParams{
			Filters: []arvados.Filter{{
				Attr:     "owner_uuid",
				Operator: "=",
				Operand:  sysUserUUID,
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
				Attr:     "head_kind",
				Operator: "=",
				Operand:  "arvados#user",
			}},
		}
		results, err := ListAll(ac, "links", params, &linkList{})
		if err != nil {
			return fmt.Errorf("error getting member links for group %q: %s", group.Name, err)
		}
		// Build a list of user ids (email or username) belonging to this group
		membersSet := make(map[string]bool)
		for _, item := range results {
			link := item.(Link)
			// We may receive an old link pointing to a removed user
			if _, found := allUsers[link.HeadUUID]; !found {
				continue
			}
			memberID, err := allUsers[link.HeadUUID].GetID(*userID)
			if err != nil {
				return err
			}
			membersSet[memberID] = true
		}
		remoteGroups[group.UUID] = &groupInfo{
			Group:           group,
			PreviousMembers: membersSet,
			CurrentMembers:  make(map[string]bool), // Empty set
		}
		groupNameToUUID[group.Name] = group.UUID
	}
	log.Printf("Found %d remote groups", len(remoteGroups))

	groupsCreated := 0
	membershipsAdded := 0
	membershipsRemoved := 0
	membershipsSkipped := 0

	csvReader := csv.NewReader(f)
	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading %q: %s", *srcPath, err)
		}
		groupName := strings.TrimSpace(record[0])
		groupMember := strings.TrimSpace(record[1]) // User ID (username or email)
		if groupName == "" || groupMember == "" {
			log.Printf("Warning: CSV record has at least one field empty (%s, %s). Skipping", groupName, groupMember)
			membershipsSkipped++
			continue
		}
		if _, found := userIDToUUID[groupMember]; !found {
			// User not present on the system, skip.
			log.Printf("Warning: there's no user with %s %q on the system, skipping.", *userID, groupMember)
			membershipsSkipped++
			continue
		}
		if _, found := groupNameToUUID[groupName]; !found {
			// Group doesn't exist, create it before continuing
			if *verbose {
				log.Printf("Remote group %q not found, creating...", groupName)
			}
			var newGroup Group
			groupData := map[string]string{
				"name":       groupName,
				"owner_uuid": parentGroup.UUID,
			}
			if err := ac.RequestAndDecode(&newGroup, "POST", "/arvados/v1/groups", jsonReader("group", groupData), nil); err != nil {
				return fmt.Errorf("error creating group named %q: %s", groupName, err)
			}
			// Update cached group data
			groupNameToUUID[groupName] = newGroup.UUID
			remoteGroups[newGroup.UUID] = &groupInfo{
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
			if *verbose {
				log.Printf("Adding %q to group %q", groupMember, groupName)
			}
			// User wasn't a member, but should.
			var newLink Link
			linkData := map[string]string{
				"owner_uuid": sysUserUUID,
				"link_class": "permission",
				"name":       "can_read",
				"tail_uuid":  groupUUID,
				"head_uuid":  userIDToUUID[groupMember],
			}
			if err := ac.RequestAndDecode(&newLink, "POST", "/arvados/v1/links", jsonReader("link", linkData), nil); err != nil {
				return fmt.Errorf("error adding group %q -> user %q read permission: %s", groupName, groupMember, err)
			}
			linkData = map[string]string{
				"owner_uuid": sysUserUUID,
				"link_class": "permission",
				"name":       "manage",
				"tail_uuid":  userIDToUUID[groupMember],
				"head_uuid":  groupUUID,
			}
			if err = ac.RequestAndDecode(&newLink, "POST", "/arvados/v1/links", jsonReader("link", linkData), nil); err != nil {
				return fmt.Errorf("error adding user %q -> group %q manage permission: %s", groupMember, groupName, err)
			}
			membershipsAdded++
		}
		gi.CurrentMembers[groupMember] = true
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
			if *verbose {
				log.Printf("Getting group membership links for user %q (%s) on group %q (%s)", evictedUser, userIDToUUID[evictedUser], groupName, groupUUID)
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
					Operand:  groupUUID,
				}, {
					Attr:     "head_uuid",
					Operator: "=",
					Operand:  userIDToUUID[evictedUser],
				}},
				// Group <- User
				{{
					Attr:     "link_class",
					Operator: "=",
					Operand:  "permission",
				}, {
					Attr:     "tail_uuid",
					Operator: "=",
					Operand:  userIDToUUID[evictedUser],
				}, {
					Attr:     "head_uuid",
					Operator: "=",
					Operand:  groupUUID,
				}},
			} {
				l, err := ListAll(ac, "links", arvados.ResourceListParams{Filters: filterset}, &linkList{})
				if err != nil {
					return fmt.Errorf("error getting links needed to remove user %q from group %q: %s", evictedUser, groupName, err)
				}
				for _, link := range l {
					links = append(links, link)
				}
			}
			for _, item := range links {
				link := item.(Link)
				if *verbose {
					log.Printf("Removing permission link for %q on group %q", evictedUser, gi.Group.Name)
				}
				if err := ac.RequestAndDecode(&link, "DELETE", "/arvados/v1/links/"+link.UUID, nil, nil); err != nil {
					return fmt.Errorf("error removing user %q from group %q: %s", evictedUser, groupName, err)
				}
			}
			membershipsRemoved++
		}
	}
	log.Printf("Groups created: %d. Memberships added: %d, removed: %d, skipped: %d", groupsCreated, membershipsAdded, membershipsRemoved, membershipsSkipped)

	return nil
}

// ListAll : Adds all objects of type 'resource' to the 'allItems' list
func ListAll(c *arvados.Client, res string, params arvados.ResourceListParams, page resourceList) (allItems []interface{}, err error) {
	// Use the maximum page size the server allows
	limit := 1<<31 - 1
	params.Limit = &limit
	params.Offset = 0
	params.Order = "uuid"
	for {
		if err = c.RequestAndDecode(&page, "GET", "/arvados/v1/"+res, nil, params); err != nil {
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
