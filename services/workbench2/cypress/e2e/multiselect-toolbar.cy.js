// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { tooltips } from '../support/msToolbarTooltips';

describe('Multiselect Toolbar Baseline Tests', () => {
    let activeUser;
    let adminUser;

    before(function () {
        // Only set up common users once. These aren't set up as aliases because
        // aliases are cleaned up after every test. Also it doesn't make sense
        // to set the same users on beforeEach() over and over again, so we
        // separate a little from Cypress' 'Best Practices' here.
        cy.getUser('admin', 'Admin', 'User', true, true)
            .as('adminUser')
            .then(function () {
                adminUser = this.adminUser;
            });
        cy.getUser('user', 'Active', 'User', false, true)
            .as('activeUser')
            .then(function () {
                activeUser = this.activeUser;
            });
    });

    it('exists in DOM in neutral state', () => {
        cy.loginAs(activeUser);
        //multiselect toolbar should exist in details card and not in data explorer
        cy.get('[data-cy=user-details-card]')
            .should('exist')
            .within(() => {
                cy.get('[data-cy=multiselect-toolbar]').should('exist');
            });
        cy.get('[data-cy=title-wrapper]')
            .should('exist')
            .within(() => {
                cy.get('[data-cy=multiselect-button]').should('not.exist');
            });
    });
});

describe('For project resources', () => {
    let activeUser;
    let adminUser;

    before(function () {
        // Only set up common users once. These aren't set up as aliases because
        // aliases are cleaned up after every test. Also it doesn't make sense
        // to set the same users on beforeEach() over and over again, so we
        // separate a little from Cypress' 'Best Practices' here.
        cy.getUser('admin', 'Admin', 'User', true, true)
            .as('adminUser')
            .then(function () {
                adminUser = this.adminUser;
            });
        cy.getUser('user', 'Active', 'User', false, true)
            .as('activeUser')
            .then(function () {
                activeUser = this.activeUser;
            });
    });

    it('should behave correctly for a single project', () => {
        cy.createProject({
            owningUser: adminUser,
            projectName: 'TestProject',
        }).as('testProject');
        cy.getAll('@testProject').then(([testProject]) => {
            cy.loginAs(adminUser);
            cy.doDataExplorerSelect(testProject.name);

            // disabled until #22787 is resolved
            // View details
            // cy.get('[aria-label="View details"]').click();
            // cy.get('[data-cy=details-panel]').contains(testProject.name).should('be.visible');
            // cy.get('[data-cy=close-details-btn]').click();

            cy.window().then((win) => {
                cy.stub(win, 'open').as('windowOpen');
            });

            // Open in new tab
            cy.get('[aria-label="Open in new tab"]').click();
            cy.get('@windowOpen').should('be.called');

            //Share
            cy.get('[aria-label="Share"]').click();
            cy.get('.sharing-dialog').should('exist');
            cy.contains('button', 'Close').click();

            //edit project
            cy.get('[aria-label="Edit project"]').click();
            cy.get("[data-cy=form-dialog]").within(() => {
                cy.contains("Edit Project").should('be.visible');
                cy.get("[data-cy=form-cancel-btn]").click();
            });

            //new project
            cy.get('[aria-label="New project"]').click();
            cy.get("[data-cy=form-dialog]").within(() => {
                cy.contains("New Project").should('be.visible');
                cy.get("[data-cy=form-cancel-btn]").click();
            });

            //freeze project
            cy.get('[aria-label="Freeze project"]').click();
            cy.assertToolbarButtons(tooltips.adminFrozenProject);

            //unfreeze project
            cy.get('[aria-label="Unfreeze project"]').click();
            cy.assertToolbarButtons(tooltips.adminProject);

            //Add to favorites
            cy.get('[aria-label="Add to favorites"]').click();
            cy.get('[data-cy=favorite-star]').should('exist')
                .parents('[data-cy=data-table-row]')
                .contains(testProject.name)

            //Add to public favorites
            cy.get('[aria-label="Add to public favorites"]').click()
            cy.get('[data-cy=public-favorite-star]').should('exist')
                .parents('[data-cy=data-table-row]')
                .contains(testProject.name)

            //Open with 3rd party client
            cy.get('[aria-label="Open with 3rd party client"]').click()
            cy.get('[role=dialog]').contains('Open with 3rd party client')
            cy.contains('Close').click()

            //API Details
            cy.get('[aria-label="API Details"]').click()
            cy.get('[role=dialog]').contains('API Details')
            cy.contains('Close').click()

        });
    });

    // The following test is enabled on Electron only, as Chromium and Firefox
    // require permissions to access the clipboard.
    it("handles project Copy UUID", { browser: 'electron' }, () => {
        cy.createProject({
            owningUser: adminUser,
            projectName: 'ClipboardTestProject',
        }).as('clipboardTestProject');
        cy.getAll('@clipboardTestProject').then(([clipboardTestProject]) => {
            cy.loginAs(adminUser);
            cy.doDataExplorerSelect(clipboardTestProject.name);

            // Copy UUID
            cy.get('[aria-label="Copy UUID"]').click()
            cy.window({ timeout: 10000 }).then(win =>{
                console.log('this ia a load-bearing console.log');
                win.focus();
                win.navigator.clipboard.readText().then(text => {
                    expect(text).to.equal(clipboardTestProject.uuid);
                })}
            );
        });
    });

    // The following test is enabled on Electron only, as Chromium and Firefox
    // require permissions to access the clipboard.
    it("handles project Copy link to clipboard", { browser: 'electron' }, () => {
        cy.createProject({
            owningUser: adminUser,
            projectName: 'ClipboardTestProject',
        }).as('clipboardTestProject');
        cy.getAll('@clipboardTestProject').then(([clipboardTestProject]) => {
            cy.loginAs(adminUser);
            cy.doDataExplorerSelect(clipboardTestProject.name);

            // Copy link to clipboard
            cy.get('[aria-label="Copy link to clipboard"]').click()
            cy.window({ timeout: 10000 }).then(win =>{
                console.log('this ia a load-bearing console.log');
                win.focus();
                win.navigator.clipboard.readText().then(text => {
                expect(text).to.match(/https\:\/\/127\.0\.0\.1\:[0-9]+\/projects\/[a-z0-9]{5}-[a-z0-9]{5}-[a-z0-9]{15}/);
                })}
            );
        });
    });

    it('should behave correctly for multiple projects', () => {
        cy.createProject({
            owningUser: adminUser,
            projectName: 'TestProject1',
        }).as('testProject1');
        cy.createProject({
            owningUser: adminUser,
            projectName: 'TestProject2',
        }).as('testProject2');
        cy.createProject({
            owningUser: adminUser,
            projectName: 'TestProject3',
        }).as('testProject3');
        cy.createProject({
            owningUser: activeUser,
            projectName: 'TestProject4',
        }).as('testProject4');
        cy.createProject({
            owningUser: activeUser,
            projectName: 'TestProject5',
        }).as('testProject5');
        cy.getAll('@testProject1', '@testProject2', '@testProject3', '@testProject4', '@testProject5').then(
            ([testProject1, testProject2, testProject3, testProject4, testProject5]) => {
                //share with active user to test permissions
                cy.shareWith(adminUser.token, activeUser.user.uuid, testProject1.uuid, 'can_read');

                // non-admin actions
                cy.loginAs(activeUser);
                cy.assertDataExplorerContains(testProject4.name, true);
                cy.assertDataExplorerContains(testProject5.name, true);

                //assert toolbar buttons
                cy.doDataExplorerSelect(testProject4.name);
                cy.assertToolbarButtons(tooltips.nonAdminProject);
                cy.doDataExplorerSelect(testProject5.name);
                cy.assertToolbarButtons(tooltips.multiProject);

                //assert read only project toolbar buttons
                cy.contains('Shared with me').click();
                cy.doDataExplorerSelect(testProject1.name);
                cy.assertToolbarButtons(tooltips.readOnlyProject);

                // admin actions
                cy.loginAs(adminUser);
                cy.assertDataExplorerContains(testProject1.name, true);
                cy.assertDataExplorerContains(testProject2.name, true);
                cy.assertDataExplorerContains(testProject3.name, true);

                //assert admin project toolbar buttons
                cy.doDataExplorerSelect(testProject1.name);
                cy.assertToolbarButtons(tooltips.adminProject);
                cy.doDataExplorerSelect(testProject2.name);
                cy.assertToolbarButtons(tooltips.multiProject);

                //check multi-project move to
                cy.get(`[aria-label="Move to"]`, { timeout: 5000 }).click();
                cy.get('[data-cy=picker-dialog-project-search]').find('input').type(testProject3.name);
                cy.get('[data-cy=projects-tree-search-picker]').contains(testProject3.name).click();
                cy.get('[data-cy=form-submit-btn]').click();

                cy.assertDataExplorerContains(testProject3.name, true).click();
                cy.assertDataExplorerContains(testProject1.name, true);
                cy.assertDataExplorerContains(testProject2.name, true);

                //check multi-project trash
                cy.doDataExplorerSelect(testProject1.name);
                cy.doDataExplorerSelect(testProject2.name);
                cy.doToolbarAction('Move to trash');
                cy.assertDataExplorerContains(testProject1.name, false);
                cy.assertDataExplorerContains(testProject2.name, false);
                cy.contains('Trash').click();
                cy.assertDataExplorerContains(testProject1.name, true);
                cy.assertDataExplorerContains(testProject2.name, true);

                //check multi-project unTrash
                cy.doDataExplorerSelect(testProject1.name);
                cy.doDataExplorerSelect(testProject2.name);
                cy.doToolbarAction('Restore');
                cy.assertDataExplorerContains(testProject1.name, false);
                cy.assertDataExplorerContains(testProject2.name, false);
                cy.contains(testProject3.name).click();
                cy.assertDataExplorerContains(testProject1.name, true);
                cy.assertDataExplorerContains(testProject2.name, true);
            }
        );
    });

    /*
    selecting/deselecting items should:
        select/deselect the correct items
        display the correct toolbar items
    select all/deselect all/invert selection in popover should:
        select/deselect the correct items
        display the correct toolbar items
    For each resource type:
        the correct toolbar is displayed when:
            One of that resource is selected
            Multiple of that resource are selected
            Some of these tests already exist, project.cy.js L231 for example. These should be removed because it's better to have all of these tests in the same place.
        Moving
            single item
            multiple of the same resource type
        Trashing
            single item
            multiple of the same resource type
        Untrashing
            single item
            multiple of the same resource type
    For mixed resource selections:
        for project & collections:
            Trashing mixed selection
            Untrashing mixed selection
            Moving mixed selection
        for processes & any other resource:
            no multiselect options should exist
    Subprocess panel should have all of the functionality of the main process view
    Data/Workflow runs tabs should have all of the functionality of the main process view
    */
});
