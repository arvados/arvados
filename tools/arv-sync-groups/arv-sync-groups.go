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

	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
)

func main() {
	err := doMain()
	if err != nil {
		log.Fatalf("%v", err)
	}
}

func doMain() error {
	const groupTag string = "remote_group"
	userIDOpts := []string{"email", "username"}

	flags := flag.NewFlagSet("arv-sync-groups", flag.ExitOnError)

	srcPath := flags.String(
		"path",
		"",
		"Local file path containing a CSV-like format.")

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

	// Parse args; omit the first arg which is the command name
	flags.Parse(os.Args[1:])

	// Validations
	if *retries < 0 {
		return fmt.Errorf("Retry quantity must be >= 0")
	}

	if *srcPath == "" {
		return fmt.Errorf("Please provide a path to an input file")
	}
	fileInfo, err := os.Stat(*srcPath)
	switch {
	case os.IsNotExist(err):
		return fmt.Errorf("File not found: %s", *srcPath)
	case fileInfo.IsDir():
		return fmt.Errorf("Path provided is not a file: %s", *srcPath)
	}

	validUserID := false
	for _, opt := range userIDOpts {
		if *userID == opt {
			validUserID = true
		}
	}
	if !validUserID {
		return fmt.Errorf("User ID must be one of: %s",
			strings.Join(userIDOpts, ", "))
	}

	arv, err := arvadosclient.MakeArvadosClient()
	if err != nil {
		return fmt.Errorf("Error setting up arvados client %s", err.Error())
	}
	arv.Retries = *retries

	log.Printf("Group sync starting. Using '%s' as users id", *userID)

	// Get the complete user list to minimize API Server requests
	allUsers := make(map[string]interface{})
	userIDToUUID := make(map[string]string) // Index by email or username
	results := make([]interface{}, 0)
	err = ListAll(arv, "users", arvadosclient.Dict{}, &results)
	if err != nil {
		return fmt.Errorf("Error getting user list from the API Server %s",
			err.Error())
	}
	log.Printf("Found %d users", len(results))
	for _, item := range results {
		userMap := item.(map[string]interface{})
		allUsers[userMap["uuid"].(string)] = userMap
		userIDToUUID[userMap[*userID].(string)] = userMap["uuid"].(string)
		if *verbose {
			log.Printf("Seen user %s", userMap[*userID].(string))
		}
	}

	// Request all UUIDs for groups tagged as remote
	remoteGroupUUIDs := make(map[string]struct{})
	results = make([]interface{}, 0)
	err = ListAll(arv, "links", arvadosclient.Dict{
		"filters": [][]string{
			{"link_class", "=", "tag"},
			{"name", "=", groupTag},
			{"head_kind", "=", "arvados#group"},
		},
	}, &results)
	if err != nil {
		return fmt.Errorf("Error getting remote group UUIDs: %s", err.Error())
	}
	for _, item := range results {
		link := item.(map[string]interface{})
		remoteGroupUUIDs[link["head_uuid"].(string)] = struct{}{}
	}
	// Get remote groups and their members
	uuidList := make([]string, 0)
	for uuid := range remoteGroupUUIDs {
		uuidList = append(uuidList, uuid)
	}
	remoteGroups := make(map[string]arvadosclient.Dict)
	groupNameToUUID := make(map[string]string) // Index by group name
	results = make([]interface{}, 0)
	err = ListAll(arv, "groups", arvadosclient.Dict{
		"filters": [][]interface{}{
			{"uuid", "in", uuidList},
		},
	}, &results)
	if err != nil {
		return fmt.Errorf("Error getting remote groups by UUID: %s", err.Error())
	}
	for _, item := range results {
		group := item.(map[string]interface{})
		results := make([]interface{}, 0)
		err := ListAll(arv, "links", arvadosclient.Dict{
			"filters": [][]string{
				{"link_class", "=", "permission"},
				{"name", "=", "can_read"},
				{"tail_uuid", "=", group["uuid"].(string)},
				{"head_kind", "=", "arvados#user"},
			},
		}, &results)
		if err != nil {
			return fmt.Errorf("Error getting member links: %s", err.Error())
		}
		// Build a list of user ids (email or username) belonging to this group
		membersSet := make(map[string]struct{}, 0)
		for _, linkItem := range results {
			link := linkItem.(map[string]interface{})
			memberID := allUsers[link["head_uuid"].(string)].(map[string]interface{})[*userID].(string)
			membersSet[memberID] = struct{}{}
		}
		remoteGroups[group["uuid"].(string)] = arvadosclient.Dict{
			"object":           group,
			"previous_members": membersSet,
			"current_members":  make(map[string]struct{}), // Empty set
		}
		// FIXME: There's an index (group_name, group.owner_uuid), should we
		// ask for our own groups tagged as remote? (with own being 'system'?)
		groupNameToUUID[group["name"].(string)] = group["uuid"].(string)
	}
	log.Printf("Found %d remote groups", len(remoteGroups))

	groupsCreated := 0
	membersAdded := 0
	membersRemoved := 0

	f, err := os.Open(*srcPath)
	if err != nil {
		return fmt.Errorf("Error opening file %s: %s", *srcPath, err.Error())
	}
	defer f.Close()

	csvReader := csv.NewReader(f)
	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("Error reading CSV file: %s", err.Error())
		}
		groupName := record[0]
		groupMember := record[1] // User ID (username or email)
		if _, found := userIDToUUID[groupMember]; !found {
			// User not present on the system, skip.
			log.Printf("Warning: there's no user with %s '%s' on the system, skipping.", *userID, groupMember)
			continue
		}
		if _, found := groupNameToUUID[groupName]; !found {
			// Group doesn't exist, create and tag it before continuing
			group := make(map[string]interface{})
			err := arv.Create("groups", arvadosclient.Dict{
				"group": arvadosclient.Dict{
					"name": groupName,
				},
			}, &group)
			if err != nil {
				return fmt.Errorf("Error creating group named '%s': %s",
					groupName, err.Error())
			}
			groupUUID := group["uuid"].(string)
			link := make(map[string]interface{})
			err = arv.Create("links", arvadosclient.Dict{
				"link": arvadosclient.Dict{
					"link_class": "tag",
					"name":       groupTag,
					"head_uuid":  groupUUID,
				},
			}, &link)
			if err != nil {
				return fmt.Errorf("Error creating tag for group '%s': %s",
					groupName, err.Error())
			}
			// Update cached group data
			groupNameToUUID[groupName] = groupUUID
			remoteGroups[groupUUID] = arvadosclient.Dict{
				"object":           group,
				"previous_members": make(map[string]struct{}), // Empty set
				"current_members":  make(map[string]struct{}), // Empty set
			}
			groupsCreated = groupsCreated + 1
		}
		// Both group & user exist, check if user is a member
		groupUUID := groupNameToUUID[groupName]
		previousMembersSet := remoteGroups[groupUUID]["previous_members"].(map[string]struct{})
		currentMembersSet := remoteGroups[groupUUID]["current_members"].(map[string]struct{})
		if !(contains(previousMembersSet, groupMember) ||
			contains(currentMembersSet, groupMember)) {
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
				return fmt.Errorf("Error adding user '%s' to group '%s': %s",
					groupMember, groupName, err.Error())
			}
			membersAdded = membersAdded + 1
		}
		currentMembersSet[groupMember] = struct{}{}
	}

	// Remove previous members not listed on this run
	for groupUUID := range remoteGroups {
		previousMembersSet := remoteGroups[groupUUID]["previous_members"].(map[string]struct{})
		currentMembersSet := remoteGroups[groupUUID]["current_members"].(map[string]struct{})
		evictedMembersSet := subtract(previousMembersSet, currentMembersSet)
		groupName := remoteGroups[groupUUID]["object"].(map[string]interface{})["name"]
		if len(evictedMembersSet) > 0 {
			log.Printf("Removing %d users from group '%s'", len(evictedMembersSet), groupName)
		}
		for evictedUser := range evictedMembersSet {
			links := make([]interface{}, 0)
			err := ListAll(arv, "links", arvadosclient.Dict{
				"filters": [][]string{
					{"link_class", "=", "permission"},
					{"name", "=", "can_read"},
					{"tail_uuid", "=", groupUUID},
					{"head_uuid", "=", userIDToUUID[evictedUser]},
				},
			}, &links)
			if err != nil {
				return fmt.Errorf("Error getting links needed to remove user '%s' from group '%s': %s", evictedUser, groupName, err.Error())
			}
			for _, link := range links {
				linkUUID := link.(map[string]interface{})["uuid"].(string)
				l := make(map[string]interface{})
				err := arv.Delete("links", linkUUID, arvadosclient.Dict{}, &l)
				if err != nil {
					return fmt.Errorf("Error removing user '%s' from group '%s': %s", evictedUser, groupName, err.Error())
				}
			}
			membersRemoved = membersRemoved + 1
		}
	}
	log.Printf("Groups created: %d, members added: %d, members removed: %d", groupsCreated, membersAdded, membersRemoved)

	return nil
}

// ListAll : Adds all objects of type 'resource' to the 'output' list
func ListAll(arv *arvadosclient.ArvadosClient, resource string, parameters arvadosclient.Dict, output *[]interface{}) (err error) {
	// Default limit value
	if _, ok := parameters["limit"]; !ok {
		parameters["limit"] = 1000
	}
	offset := 0
	itemsAvailable := parameters["limit"].(int)
	for len(*output) < itemsAvailable {
		results := make(arvadosclient.Dict)
		parameters["offset"] = offset
		err = arv.List(resource, parameters, &results)
		if err != nil {
			return err
		}
		if value, ok := results["items"]; ok {
			items := value.([]interface{})
			for _, item := range items {
				*output = append(*output, item)
			}
			offset = int(results["offset"].(float64)) + len(items)
		}
		itemsAvailable = int(results["items_available"].(float64))
	}
	return nil
}

func contains(set map[string]struct{}, element string) bool {
	_, found := set[element]
	return found
}

func subtract(setA map[string]struct{}, setB map[string]struct{}) map[string]struct{} {
	result := make(map[string]struct{})
	for element := range setA {
		if !contains(setB, element) {
			result[element] = struct{}{}
		}
	}
	return result
}
