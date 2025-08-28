// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

describe('Trash tests', function () {
    let adminUser;

    before(function () {
        cy.getUser('admin', 'Admin', 'User', true, true)
            .as('adminUser').then(function () {
                adminUser = this.adminUser;
            });
    });

    it('trashes and untrashes projects', function() {
        // Create test project
        cy.createProject({
            owningUser: adminUser,
            projectName: `trashTestProject`,
        }).as('testProject');

        cy.getAll('@testProject')
            .then(function ([testProject]) {
                cy.loginAs(adminUser);
                cy.doSidePanelNavigation('Home Projects');
                cy.doMPVTabSelect("Data");

                // Project Trash Tests

                // Trash with context menu
                cy.doDataExplorerContextAction(testProject.name, 'Move to trash');

                // Verify trashed and breadcrumbs correct
                cy.assertDataExplorerContains(testProject.name, false);
                cy.assertBreadcrumbs(["Home Projects"]);

                // Restore with context menu
                cy.get('[data-cy=side-panel-tree]').contains('Trash').click();
                cy.assertBreadcrumbs(["Trash"]);
                cy.doDataExplorerSearch(testProject.name);
                cy.doDataExplorerContextAction(testProject.name, 'Restore');

                // Verify navigated to project
                cy.assertBreadcrumbs(["Home Projects", testProject.name]);
                cy.assertUrlPathname(`/projects/${testProject.uuid}`);
                // Verify present in home project
                cy.get('[data-cy=side-panel-tree]').contains('Home Projects').click();
                cy.assertBreadcrumbs(["Home Projects"]);
                cy.doMPVTabSelect("Data");
                cy.assertDataExplorerContains(testProject.name, true);

                // Test delete from toolbar
                cy.doDataExplorerSelect(testProject.name);
                cy.doToolbarAction("Move to trash");

                // Verify trashed and breadcrumbs correct
                cy.assertDataExplorerContains(testProject.name, false);
                cy.assertBreadcrumbs(["Home Projects"]);

                // Restore with toolbar
                cy.get('[data-cy=side-panel-tree]').contains('Trash').click();
                cy.assertBreadcrumbs(["Trash"]);
                cy.doDataExplorerSearch(testProject.name);
                cy.doDataExplorerSelect(testProject.name);
                cy.get(`[aria-label="Restore"]`, { timeout: 5000 }).eq(0).click();
                cy.waitForDom();

                // Verify navigated to project
                cy.assertBreadcrumbs(["Home Projects", testProject.name]);
                cy.assertUrlPathname(`/projects/${testProject.uuid}`);
                // Verify present in home project
                cy.get('[data-cy=side-panel-tree]').contains('Home Projects').click();
                cy.assertBreadcrumbs(["Home Projects"]);
                cy.assertDataExplorerContains(testProject.name, true);
            });
    });

    it("trashes and untrashes collections", function() {
        // Create test collection
        cy.createCollection(adminUser.token, {
            owner_uuid: adminUser.user.uuid,
            name: `trashTestCollection ${Math.floor(Math.random() * 999999)}`,
        }).as('testCollection');

        cy.getAll('@testCollection')
            .then(function ([testCollection]) {
                cy.loginAs(adminUser);
                cy.doSidePanelNavigation('Home Projects');
                cy.doMPVTabSelect("Data");

                // Collection Trash Tests

                // Trash with context menu
                cy.doDataExplorerContextAction(testCollection.name, 'Move to trash');

                // Verify trashed and breadcrumbs correct
                cy.assertDataExplorerContains(testCollection.name, false);
                cy.assertBreadcrumbs(["Home Projects"]);

                // Restore with context menu
                cy.get('[data-cy=side-panel-tree]').contains('Trash').click();
                cy.assertBreadcrumbs(["Trash"]);
                cy.doDataExplorerSearch(testCollection.name);
                cy.doDataExplorerContextAction(testCollection.name, 'Restore');

                // Verify not in trash and in home project
                cy.assertDataExplorerContains(testCollection.name, false);
                cy.assertBreadcrumbs(["Trash"]);
                cy.get('[data-cy=side-panel-tree]').contains('Home Projects').click();
                cy.assertBreadcrumbs(["Home Projects"]);
                cy.assertDataExplorerContains(testCollection.name, true);

                // Test delete from toolbar
                cy.doDataExplorerSelect(testCollection.name);
                cy.doToolbarAction("Move to trash");

                // Verify trashed and breadcrumbs correct
                cy.assertDataExplorerContains(testCollection.name, false);
                cy.assertBreadcrumbs(["Home Projects"]);

                // Restore with toolbar
                cy.get('[data-cy=side-panel-tree]').contains('Trash').click();
                cy.assertBreadcrumbs(["Trash"]);
                cy.doDataExplorerSearch(testCollection.name);
                cy.doDataExplorerSelect(testCollection.name);
                cy.get(`[aria-label="Restore"]`, { timeout: 5000 }).eq(0).click();
                cy.waitForDom();

                // Verify not in trash and in home project
                cy.assertDataExplorerContains(testCollection.name, false);
                cy.assertBreadcrumbs(["Trash"]);
                cy.get('[data-cy=side-panel-tree]').contains('Home Projects').click();
                cy.assertBreadcrumbs(["Home Projects"]);
                cy.assertDataExplorerContains(testCollection.name, true);
            });
    });
});
