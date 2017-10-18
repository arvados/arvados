// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
)

type resourceList interface {
	items() []interface{}
	itemsAvailable() int
	offset() int
}

type groupInfo struct {
	Group           group
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
	Items          []user `json:"items"`
	ItemsAvailable int    `json:"items_available"`
	Offset         int    `json:"offset"`
}

func (l userList) items() []interface{} {
	var out []interface{}
	for _, item := range l.Items {
		out = append(out, item)
	}
	return out
}

func (l userList) itemsAvailable() int {
	return l.ItemsAvailable
}

func (l userList) offset() int {
	return l.Offset
}

type group struct {
	UUID      string `json:"uuid,omitempty"`
	Name      string `json:"name,omitempty"`
	OwnerUUID string `json:"owner_uuid,omitempty"`
}

// groupList implements resourceList interface
type groupList struct {
	Items          []group `json:"items"`
	ItemsAvailable int     `json:"items_available"`
	Offset         int     `json:"offset"`
}

func (l groupList) items() []interface{} {
	var out []interface{}
	for _, item := range l.Items {
		out = append(out, item)
	}
	return out
}

func (l groupList) itemsAvailable() int {
	return l.ItemsAvailable
}

func (l groupList) offset() int {
	return l.Offset
}

type link struct {
	UUID      string `json:"uuid, omiempty"`
	Name      string `json:"name,omitempty"`
	LinkClass string `json:"link_class,omitempty"`
	HeadUUID  string `json:"head_uuid,omitempty"`
	HeadKind  string `json:"head_kind,omitempty"`
	TailUUID  string `json:"tail_uuid,omitempty"`
	TailKind  string `json:"tail_kind,omitempty"`
}

// linkList implements resourceList interface
type linkList struct {
	Items          []link `json:"items"`
	ItemsAvailable int    `json:"items_available"`
	Offset         int    `json:"offset"`
}

func (l linkList) items() []interface{} {
	var out []interface{}
	for _, item := range l.Items {
		out = append(out, item)
	}
	return out
}

func (l linkList) itemsAvailable() int {
	return l.ItemsAvailable
}

func (l linkList) offset() int {
	return l.Offset
}

func main() {
	err := doMain()
	if err != nil {
		log.Fatalf("%v", err)
	}
}

func doMain() error {
	const groupTag string = "remote_group"
	const remoteGroupParentName string = "Externally synchronized groups"
	userIDOpts := []string{"email", "username"}

	flags := flag.NewFlagSet("arv-sync-groups", flag.ExitOnError)

	srcPath := flags.String(
		"path",
		"",
		"Local file path containing a CSV format.")

	userID := flags.String(
		"user-id",
		"email",
		"Attribute by which every user is identified. "+
			"Valid values are: email (the default) and username.")

	verbose := flags.Bool(
		"verbose",
		false,
		"Log informational messages. By default is deactivated.")

	retries := flags.Int(
		"retries",
		3,
		"Maximum number of times to retry server requests that encounter "+
			"temporary failures (e.g., server down).  Default 3.")

	parentGroupUUID := flags.String(
		"parent-group-uuid",
		"",
		"Use given group UUID as a parent for the remote groups. Should "+
			"be owned by the system user. If not specified, a group named '"+
			remoteGroupParentName+"' will be used (and created if nonexistant).")

	// Parse args; omit the first arg which is the command name
	flags.Parse(os.Args[1:])

	// Validations
	if *retries < 0 {
		return fmt.Errorf("retry quantity must be >= 0")
	}

	if *srcPath == "" {
		return fmt.Errorf("please provide a path to an input file")
	}

	// Try opening the input file early, just in case there's problems.
	f, err := os.Open(*srcPath)
	if err != nil {
		return fmt.Errorf("%s", err)
	}
	defer f.Close()

	validUserID := false
	for _, opt := range userIDOpts {
		if *userID == opt {
			validUserID = true
		}
	}
	if !validUserID {
		return fmt.Errorf("user ID must be one of: %s",
			strings.Join(userIDOpts, ", "))
	}

	// Arvados Client setup
	ac := arvados.NewClientFromEnv()
	arv, err := arvadosclient.New(ac)
	if err != nil {
		return fmt.Errorf("error setting up arvados client %s", err)
	}
	arv.Retries = *retries

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
	var parentGroup group
	if *parentGroupUUID == "" {
		// UUID not provided, search for preexisting parent group
		var gl groupList
		err := arv.List("groups", arvadosclient.Dict{
			"filters": [][]string{
				{"name", "=", remoteGroupParentName},
				{"owner_uuid", "=", sysUserUUID}},
		}, &gl)
		if err != nil {
			return fmt.Errorf("error searching for parent group: %s", err)
		}
		if len(gl.Items) == 0 {
			// Default parent group not existant, create one.
			if *verbose {
				log.Println("Default parent group not found, creating...")
			}
			err := arv.Create("groups", arvadosclient.Dict{
				"group": arvadosclient.Dict{
					"name":       remoteGroupParentName,
					"owner_uuid": sysUserUUID},
			}, &parentGroup)
			if err != nil {
				return fmt.Errorf("error creating system user owned group named %q: %s", remoteGroupParentName, err)
			}
		} else if len(gl.Items) == 1 {
			// Default parent group found.
			parentGroup = gl.Items[0]
		} else {
			// This should never happen, as there's an unique index for
			// (owner_uuid, name) on groups.
			return fmt.Errorf("found %d groups owned by system user and named %q", len(gl.Items), remoteGroupParentName)
		}
	} else {
		// UUID provided. Check if exists and if it's owned by system user
		err := arv.Get("groups", *parentGroupUUID, arvadosclient.Dict{}, &parentGroup)
		if err != nil {
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
	results, err := ListAll(arv, "users", arvadosclient.Dict{}, &userList{})
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

	// Request all UUIDs for groups tagged as remote
	remoteGroupUUIDs := make(map[string]bool)
	results, err = ListAll(arv, "links", arvadosclient.Dict{
		"filters": [][]string{
			{"link_class", "=", "tag"},
			{"name", "=", groupTag},
			{"head_kind", "=", "arvados#group"},
		},
	}, &linkList{})
	if err != nil {
		return fmt.Errorf("error getting remote group UUIDs: %s", err)
	}
	for _, item := range results {
		link := item.(link)
		remoteGroupUUIDs[link.HeadUUID] = true
	}
	// Get remote groups and their members
	var uuidList []string
	for uuid := range remoteGroupUUIDs {
		uuidList = append(uuidList, uuid)
	}
	remoteGroups := make(map[string]*groupInfo)
	groupNameToUUID := make(map[string]string) // Index by group name
	results, err = ListAll(arv, "groups", arvadosclient.Dict{
		"filters": [][]interface{}{
			{"uuid", "in", uuidList},
			{"owner_uuid", "=", parentGroup.UUID},
		},
	}, &groupList{})
	if err != nil {
		return fmt.Errorf("error getting remote groups by UUID: %s", err)
	}
	for _, item := range results {
		group := item.(group)
		results, err := ListAll(arv, "links", arvadosclient.Dict{
			"filters": [][]string{
				{"link_class", "=", "permission"},
				{"name", "=", "can_read"},
				{"tail_uuid", "=", group.UUID},
				{"head_kind", "=", "arvados#user"},
			},
		}, &linkList{})
		if err != nil {
			return fmt.Errorf("error getting member links: %s", err)
		}
		// Build a list of user ids (email or username) belonging to this group
		membersSet := make(map[string]bool)
		for _, item := range results {
			link := item.(link)
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
	membersAdded := 0
	membersRemoved := 0

	csvReader := csv.NewReader(f)
	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading %q: %s", *srcPath, err)
		}
		groupName := record[0]
		groupMember := record[1] // User ID (username or email)
		if _, found := userIDToUUID[groupMember]; !found {
			// User not present on the system, skip.
			log.Printf("Warning: there's no user with %s %q on the system, skipping.", *userID, groupMember)
			continue
		}
		if _, found := groupNameToUUID[groupName]; !found {
			// Group doesn't exist, create and tag it before continuing
			if *verbose {
				log.Printf("Remote group %q not found, creating...", groupName)
			}
			var group group
			err := arv.Create("groups", arvadosclient.Dict{
				"group": arvadosclient.Dict{
					"name":       groupName,
					"owner_uuid": parentGroup.UUID,
				},
			}, &group)
			if err != nil {
				return fmt.Errorf("error creating group named %q: %s",
					groupName, err)
			}
			link := make(map[string]interface{})
			err = arv.Create("links", arvadosclient.Dict{
				"link": arvadosclient.Dict{
					"link_class": "tag",
					"name":       groupTag,
					"head_uuid":  group.UUID,
				},
			}, &link)
			if err != nil {
				return fmt.Errorf("error creating tag for group %q: %s",
					groupName, err)
			}
			// Update cached group data
			groupNameToUUID[groupName] = group.UUID
			remoteGroups[group.UUID] = &groupInfo{
				Group:           group,
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
			link := make(map[string]interface{})
			err := arv.Create("links", arvadosclient.Dict{
				"link": arvadosclient.Dict{
					"link_class": "permission",
					"name":       "can_read",
					"tail_uuid":  groupUUID,
					"head_uuid":  userIDToUUID[groupMember],
				},
			}, &link)
			if err != nil {
				return fmt.Errorf("error adding user %q to group %q: %s",
					groupMember, groupName, err)
			}
			membersAdded++
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
			links, err := ListAll(arv, "links", arvadosclient.Dict{
				"filters": [][]string{
					{"link_class", "=", "permission"},
					{"name", "=", "can_read"},
					{"tail_uuid", "=", groupUUID},
					{"head_uuid", "=", userIDToUUID[evictedUser]},
				},
			}, &linkList{})
			if err != nil {
				return fmt.Errorf("error getting links needed to remove user %q from group %q: %s", evictedUser, groupName, err)
			}
			for _, item := range links {
				link := item.(link)
				var l map[string]interface{}
				if *verbose {
					log.Printf("Removing %q from group %q", evictedUser, gi.Group.Name)
				}
				err := arv.Delete("links", link.UUID, arvadosclient.Dict{}, &l)
				if err != nil {
					return fmt.Errorf("error removing user %q from group %q: %s", evictedUser, groupName, err)
				}
			}
			membersRemoved++
		}
	}
	log.Printf("Groups created: %d, members added: %d, members removed: %d", groupsCreated, membersAdded, membersRemoved)

	return nil
}

// ListAll : Adds all objects of type 'resource' to the 'output' list
func ListAll(arv *arvadosclient.ArvadosClient, resource string, parameters arvadosclient.Dict, rl resourceList) (allItems []interface{}, err error) {
	if _, ok := parameters["limit"]; !ok {
		// Default limit value: use the maximum page size the server allows
		parameters["limit"] = 1<<31 - 1
	}
	offset := 0
	itemsAvailable := parameters["limit"].(int)
	for len(allItems) < itemsAvailable {
		parameters["offset"] = offset
		err = arv.List(resource, parameters, &rl)
		if err != nil {
			return allItems, err
		}
		for _, i := range rl.items() {
			allItems = append(allItems, i)
		}
		offset = rl.offset() + len(rl.items())
		itemsAvailable = rl.itemsAvailable()
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
