[comment]: # (Copyright © The Arvados Authors. All rights reserved.)
[comment]: # ()
[comment]: # (SPDX-License-Identifier: CC-BY-SA-3.0)

# Manual testing plan

## Manual testing of SDKs that don’t have good coverage

## Release candidate builds with ~1 ~2

## Workbench2

Need to go through this whole testing plan with both an admin and non-admin account. Admin only operations are indicated.

### Login

- Test login using username/password
- Test login using OpenID Connect
- Test login as federated user
  - Login to using remote account on centralized federation (LoginCluster)
  - Login to using remote account on peer federation

### Left side navigation

- Click on each top level icon (home projects, favorites/public favorite, shared, all processes, instance types, shell access, groups, trash) and confirm that the appropriate page loads with no errors
- Check that the left side panel can be resized
- Check that that the toggle side panel button works as expected
- Check that the +NEW button is disabled unless a project is displayed

### Home projects top panel

- Project name should match the logged in user
- Check for expected toolbar buttons
  - Details
  - User account
  - API Details
- Check the buttons have the expected behavior

### Project view

#### Top panel

Check that it shows the project name

Check that it shows the first line of the project description. Check that there is an arrow that expands to show the full description.

Check that it shows project properties. Check that there is an arrow that expands to show all the properties if they don’t fit on a single line.

Check that it renders the toolbar of project operations (listed below).

Check that the each operation behaves correctly and operates on the project.

Check that if the window is narrow, the rightmost toolbar icons spill into an overflow menu

Check that breadcrumbs include the project name and each parent project.

#### Data tab

Should show projects, workflows, and collections (in that order)

Clicking on the name rendered in blue text should navigate to the item

Clicking anywhere else but the name should toggle between selected and not selected

- Unless clicking on the checkbox, clicking on the row clears any other selected items
- When a row is selected, the toolbar moves from the top panel to the data table panel
- Clicking on the check box to the left when a different item is selected selects both items
  - The toolbar updates to show only the operations that can be applied to both items
- Clicking “View details” should open the right info panel. Check that it shows details for the currently selected item.

Check that the toolbar operations are sorted and grouped consistently across different types of items.

Check that the toolbar operations are appropriate to the type of item selected.

Expected toolbar when project is selected:

- View details
- Open in new tab
- Copy link to clipboard
- Open with 3rd party client
- API details
- —-
- Share
- New project
- Edit project
- Move to
- Move to trash
- —-
- Freeze project
- Add to favorites
- Add to public favorites (admin only)

Expected toolbar when workflow is selected:

- View details
- Open in new tab
- Copy link to clipboard
- API details
- —-
- Run workflow
- Delete workflow

Expected toolbar when collection is selected

- View details
- Open in new tab
- Copy link to clipboard
- Open with 3rd party client
- API details
- —-
- Share
- Edit collection
- Move to
- Make a copy
- Move to trash
- —-
- Add to favorites
- Add to public favorites (admin only)

Check that all the toolbar operations work as expected.

Check that right-clicking on a row selects the row and then opens the appropriate context menu.

- Check that the operations apply to the item that was clicked on.
- Check that the operations in the right-click context menu match the toolbar.

Check that clicking on each action in the context menu works as expected.

Check that entering text into the search box refreshes the list with search results

Check that clicking on the three bars in the upper right opens a menu to select columns

Check that enabling/disabling data columns works. Check that all columns are filled in appropriately for each item, or blank (“-”) where no such data applies.

Check that clicking on the “Name” column sorts by name.

- Check sort by “Date created”
- Check sort by “Last modified”
- Check sort by “Trash at”
- Check sort by “Delete at”

Check that clicking on “Go to the next page” loads the next page of items.

Check that the getting the number of items doesn’t block loading the table contents.

#### Workflows tab

Check it shows processes (workflow runs) only.

Check that it shows the number of completed, failed, queued and running processes, as well as the total, at the top of the data table.

Check that it shows the name, status, type, runtime and last modified times.

Check that entering text into the search box refreshes the list with search results

Check that the toolbar and context menu behave as expected:

- View details
- Open in new tab
- Outputs
- API details
- —-
- Edit process
- Copy and re-run process
- Remove
- —-
- Add to favorites
- Add to public favorites (admin only)

Check that selecting more that one item updates the toolbar to “Remove”.

Check that clicking on “Go to the next page” loads the next page of items.

Check that the getting the number of items doesn’t block loading the table contents.

Check that process status is rendered correctly.

Check that filtering by process status shows only rows with the intended status.

Check that runtime is calculated/rendered correctly.

### My favorites

Check that all items marked as “favorite” appear.

Check that clicking on an item shows the appropriate toolbar.

Check that clicking “Remove from favorites” on an item refreshes the favorites list and the item is no longer present.

### Public favorites

Check that all items marked as “public favorite” appear.

Check that clicking on an item shows the appropriate toolbar.

Check that clicking “Remove from public favorites” on an item refreshes the public favorites list and the item is no longer present. (admin only)

### Shared with me

Check that it shows all the things that don’t belong to the current user.

Check that selecting an item shows the appropriate toolbar.

Check that clicking on “Go to the next page” loads the next page of items.

Check that the getting the number of items doesn’t block loading the table contents.

### All processes

Check that it shows all processes visible to the user, regardless of owner project.

Check that selecting an item shows the appropriate toolbar.

Check that clicking on “Go to the next page” loads the next page of items.

Check that the getting the number of items doesn’t block loading the table contents.

### Instance types

Check that all the available instance types are listed and formatted properly.

### Shell Access

Check that shell nodes are listed.

Check that the ssh command line is valid.

Check that webshell works properly.

## Groups — standalone and peer federation

1.  Create group
2.  Log in as non-admin user.
3.  Log in as a second non-admin user in a private window for testing sharing.
4.  check that users cannot see one another
5.  Add user to group
6.  Check that users can see one another

## Collections

1.  Create a collection & upload a file
2.  Add a file
3.  Rename a file
4.  Remove a file
5.  Download one of the files
6.  Make a sharing link to the collection & check usage from private window
7.  Mark collection as a favorite, check that it shows up in favorites
8.  Rename collection
9.  Edit description
10. Add property
11. Search for collection by property
12. Search for collection by name
13. Search for collection by filename
14. Search for collection by keyword in description
15. Trash collection
16. Check that collection can be found in the trash
17. Untrash collection

## Projects

1.  Create a project
2.  Rename a project
3.  Edit description
4.  Create a collection inside the project
5.  Move a collection into the project
6.  Add read-only sharing permission to the project & check access from other user
7.  Add read-write sharing permission to project & check access from other user
8.  Add manage sharing permission to project & check access from other user
9.  Mark project as favorite, check that it shows up in favorites
10. Search for project by name
11. Search for project by keyword in description
12. Trash project
13. Check that project can be found in the trash
14. Untrash project

## Workflows

1.  Upload workflow with arvados-cwl-runnner —create-workflow
2.  Browse workflow
3.  Select workflow to run
4.  Choose input file
5.  Watch it run
    1.  Check logging
    2.  Check live updates
    3.  Check links to input & output
6.  Check that it shows up in All Processes

## Federation

### Peer federation

2 or more clusters are configured with a ‘Remoteclusters’ entry in config.yml.

### Login cluster federation

2 or more clusters are configured with a ‘Remoteclusters’ entry in config.yml. One of the clusters is the ‘login cluster’, which means the **other** clusters have a section like this in their config (clsr1 is the login cluster):

    Clusters:
      clsr2:
        Login:
          LoginCluster: clsr1

#### Groups

1.  Login cluster: create group
2.  Satellite cluster: Log in as non-admin user.
3.  Satellite cluster: Log in as a second non-admin user in a private window for testing sharing.
4.  Satellite cluster: check that users cannot see one another
5.  Login cluster: add both users to group
6.  Satellite cluster: Check that users can see one another
7.  Satellite cluster: create group
8.  Satellite cluster: add both users to group
9.  Satellite cluster: Check that both users can share with the group created on the satellite cluster

## Misc

1.  As admin, create a “public favorite” and make sure users see it.
2.  As admin, deactivate a user. Make sure that user can’t log back in
3.  Add a cluster for multi-site search.
4.  Upload ssh key & check view
5.  Create git repo & check view
6.  As admin, add virtual machine access & check view
