// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { tooltips } from '../support/msToolbarTooltips';

const testWFDefinition = "{\n    \"$graph\": [\n        {\n            \"class\": \"Workflow\",\n            \"doc\": \"Reverse the lines in a document, then sort those lines.\",\n            \"hints\": [\n                {\n                    \"acrContainerImage\": \"99b0201f4cade456b4c9d343769a3b70+261\",\n                    \"class\": \"http://arvados.org/cwl#WorkflowRunnerResources\"\n                }\n            ],\n            \"id\": \"#main\",\n            \"inputs\": [\n                {\n                    \"default\": null,\n                    \"doc\": \"The input file to be processed.\",\n                    \"id\": \"#main/input\",\n                    \"type\": \"File\"\n                },\n                {\n                    \"default\": true,\n                    \"doc\": \"If true, reverse (decending) sort\",\n                    \"id\": \"#main/reverse_sort\",\n                    \"type\": \"boolean\"\n                }\n            ],\n            \"outputs\": [\n                {\n                    \"doc\": \"The output with the lines reversed and sorted.\",\n                    \"id\": \"#main/output\",\n                    \"outputSource\": \"#main/sorted/output\",\n                    \"type\": \"File\"\n                }\n            ],\n            \"steps\": [\n                {\n                    \"id\": \"#main/rev\",\n                    \"in\": [\n                        {\n                            \"id\": \"#main/rev/input\",\n                            \"source\": \"#main/input\"\n                        }\n                    ],\n                    \"out\": [\n                        \"#main/rev/output\"\n                    ],\n                    \"run\": \"#revtool.cwl\"\n                },\n                {\n                    \"id\": \"#main/sorted\",\n                    \"in\": [\n                        {\n                            \"id\": \"#main/sorted/input\",\n                            \"source\": \"#main/rev/output\"\n                        },\n                        {\n                            \"id\": \"#main/sorted/reverse\",\n                            \"source\": \"#main/reverse_sort\"\n                        }\n                    ],\n                    \"out\": [\n                        \"#main/sorted/output\"\n                    ],\n                    \"run\": \"#sorttool.cwl\"\n                }\n            ]\n        },\n        {\n            \"baseCommand\": \"rev\",\n            \"class\": \"CommandLineTool\",\n            \"doc\": \"Reverse each line using the `rev` command\",\n            \"hints\": [\n                {\n                    \"class\": \"ResourceRequirement\",\n                    \"ramMin\": 8\n                }\n            ],\n            \"id\": \"#revtool.cwl\",\n            \"inputs\": [\n                {\n                    \"id\": \"#revtool.cwl/input\",\n                    \"inputBinding\": {},\n                    \"type\": \"File\"\n                }\n            ],\n            \"outputs\": [\n                {\n                    \"id\": \"#revtool.cwl/output\",\n                    \"outputBinding\": {\n                        \"glob\": \"output.txt\"\n                    },\n                    \"type\": \"File\"\n                }\n            ],\n            \"stdout\": \"output.txt\"\n        },\n        {\n            \"baseCommand\": \"sort\",\n            \"class\": \"CommandLineTool\",\n            \"doc\": \"Sort lines using the `sort` command\",\n            \"hints\": [\n                {\n                    \"class\": \"ResourceRequirement\",\n                    \"ramMin\": 8\n                }\n            ],\n            \"id\": \"#sorttool.cwl\",\n            \"inputs\": [\n                {\n                    \"id\": \"#sorttool.cwl/reverse\",\n                    \"inputBinding\": {\n                        \"position\": 1,\n                        \"prefix\": \"-r\"\n                    },\n                    \"type\": \"boolean\"\n                },\n                {\n                    \"id\": \"#sorttool.cwl/input\",\n                    \"inputBinding\": {\n                        \"position\": 2\n                    },\n                    \"type\": \"File\"\n                }\n            ],\n            \"outputs\": [\n                {\n                    \"id\": \"#sorttool.cwl/output\",\n                    \"outputBinding\": {\n                        \"glob\": \"output.txt\"\n                    },\n                    \"type\": \"File\"\n                }\n            ],\n            \"stdout\": \"output.txt\"\n        }\n    ],\n    \"cwlVersion\": \"v1.0\"\n}"

function createContainerRequest(user, name, docker_image, command, reuse = false, state = "Uncommitted", ownerUuid) {
    return cy.setupDockerImage(docker_image).then(function (dockerImage) {
        return cy.createContainerRequest(user.token, {
            name: name,
            command: command,
            container_image: dockerImage.portable_data_hash, // for some reason, docker_image doesn't work here
            output_path: "stdout.txt",
            priority: 1,
            runtime_constraints: {
                vcpus: 1,
                ram: 1,
            },
            use_existing: reuse,
            state: state,
            mounts: {
                '/var/lib/cwl/workflow.json': {
                    kind: "tmp",
                    path: "/tmp/foo",
                },
            },
            owner_uuid: ownerUuid || undefined,
        });
    });
}

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

    it('uses selector popover to select the correct items', () => {
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
        createContainerRequest(
            adminUser,
            `test_container_request_1 ${Math.floor(Math.random() * 999999)}`,
            "arvados/jobs",
            ["echo", "hello world"],
            false,
            "Committed"
        ).as('testProcess1');
        createContainerRequest(
            adminUser,
            `test_container_request_2 ${Math.floor(Math.random() * 999999)}`,
            "arvados/jobs",
            ["echo", "hello world"],
            false,
            "Committed"
        ).as('testProcess2');
        createContainerRequest(
            adminUser,
            `test_container_request_3 ${Math.floor(Math.random() * 999999)}`,
            "arvados/jobs",
            ["echo", "hello world"],
            false,
            "Committed"
        ).as('testWorkflow3');
        cy.getAll('@testProject1', '@testProject2', '@testProject3', '@testProcess1', '@testProcess2', '@testWorkflow3')
            .then(([testProject1, testProject2, testProject3, testProcess1, testProcess2, testWorkflow3]) => {
                cy.loginAs(adminUser);

                // Data tab
                cy.assertCheckboxes([testProject1.uuid, testProject2.uuid, testProject3.uuid], false);

                    //check that a thing can be checked
                    cy.doDataExplorerSelect(testProject1.name);
                    cy.assertCheckboxes([testProject1.uuid], true);
                    cy.assertCheckboxes([testProject2.uuid, testProject3.uuid], false);

                    //check invert
                    cy.get('[data-cy=data-table-multiselect-popover]').click();
                    cy.get('[data-cy=multiselect-popover-Invert]').click();
                    cy.assertCheckboxes([testProject1.uuid], false);
                    cy.assertCheckboxes([testProject2.uuid, testProject3.uuid], true);
                    //check all
                    cy.get('[data-cy=data-table-multiselect-popover]').click();
                    cy.get('[data-cy=multiselect-popover-All]').click();
                    cy.assertCheckboxes([testProject1.uuid, testProject2.uuid, testProject3.uuid], true);
                    //check none
                    cy.get('[data-cy=data-table-multiselect-popover]').click();
                    cy.get('[data-cy=multiselect-popover-None]').click();
                    cy.assertCheckboxes([testProject1.uuid, testProject2.uuid, testProject3.uuid], false);

                // Workflow Runs tab
                cy.get('[data-cy=mpv-tabs]').contains("Workflow Runs").click();
                cy.assertCheckboxes([testProcess1.uuid], false);

                    //check that a thing can be checked
                    cy.doDataExplorerSelect(testProcess1.name);
                    cy.assertCheckboxes([testProcess1.uuid], true);
                    cy.assertCheckboxes([testProcess2.uuid, testWorkflow3.uuid], false);

                    //check invert
                    cy.get('[data-cy=data-table-multiselect-popover]').click();
                    cy.get('[data-cy=multiselect-popover-Invert]').click();
                    cy.assertCheckboxes([testProcess1.uuid], false);
                    cy.assertCheckboxes([testProcess2.uuid, testWorkflow3.uuid], true);
                    //check all
                    cy.get('[data-cy=data-table-multiselect-popover]').click();
                    cy.get('[data-cy=multiselect-popover-All]').click();
                    cy.assertCheckboxes([testProcess1.uuid, testProcess2.uuid, testWorkflow3.uuid], true);
                    //check none
                    cy.get('[data-cy=data-table-multiselect-popover]').click();
                    cy.get('[data-cy=multiselect-popover-None]').click();
                    cy.assertCheckboxes([testProcess1.uuid, testProcess2.uuid, testWorkflow3.uuid], false);

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
});

describe('For collection resources', () => {
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

    it('should behave correctly for a single collection', () => {
        cy.createCollection(adminUser.token, {
            name: `Test collection ${Math.floor(Math.random() * 999999)}`,
            owner_uuid: adminUser.user.uuid,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:bar\n",
        }).as("testCollection")
        cy.getAll('@testCollection').then(([testCollection]) => {
            cy.loginAs(adminUser);
            cy.doDataExplorerSelect(testCollection.name);

            // disabled until #22787 is resolved
            // View details
            // cy.get('[aria-label="View details"]').click();
            // cy.get('[data-cy=details-panel]').contains(testCollection.name).should('be.visible');
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

            //edit collection
            cy.get('[aria-label="Edit collection"]').click();
            cy.get("[data-cy=form-dialog]").within(() => {
                cy.contains("Edit Collection").should('be.visible');
                cy.get("[data-cy=form-cancel-btn]").click();
            });

            //Make a copy
            cy.get('[aria-label="Make a copy"]').click();
            cy.get("[data-cy=form-dialog]").within(() => {
                cy.contains("Make a copy").should('be.visible');
                cy.get("[data-cy=form-cancel-btn]").click();
            });

            //Add to favorites
            cy.get('[aria-label="Add to favorites"]').click();
            cy.get('[data-cy=favorite-star]').should('exist')
                .parents('[data-cy=data-table-row]')
                .contains(testCollection.name)

            //Add to public favorites
            cy.get('[aria-label="Add to public favorites"]').click()
            cy.get('[data-cy=public-favorite-star]').should('exist')
                .parents('[data-cy=data-table-row]')
                .contains(testCollection.name)

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

    it('should behave correctly for multiple collections', () => {
        cy.createProject({
            owningUser: adminUser,
            projectName: 'TestProject1',
        }).as('testProject1');
        cy.createCollection(adminUser.token, {
            name: `Test collection ${Math.floor(Math.random() * 999999)}`,
            owner_uuid: adminUser.user.uuid,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:bar\n",
        }).as("testCollection1")
        cy.createCollection(adminUser.token, {
            name: `Test collection ${Math.floor(Math.random() * 999999)}`,
            owner_uuid: adminUser.user.uuid,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:bar\n",
        }).as("testCollection2")
        cy.createCollection(adminUser.token, {
            name: `Test collection ${Math.floor(Math.random() * 999999)}`,
            owner_uuid: adminUser.user.uuid,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:bar\n",
        }).as("testCollection3")
        cy.createCollection(adminUser.token, {
            name: `Test collection ${Math.floor(Math.random() * 999999)}`,
            owner_uuid: adminUser.user.uuid,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:bar\n",
        }).as("testCollection4")
        cy.getAll('@testProject1', '@testCollection1', '@testCollection2', '@testCollection3', '@testCollection4')
            .then(([testProject1, testCollection1, testCollection2, testCollection3, testCollection4]) => {

                cy.loginAs(adminUser);
                cy.assertDataExplorerContains(testProject1.name, true);
                cy.assertDataExplorerContains(testCollection1.name, true);
                cy.assertDataExplorerContains(testCollection2.name, true);
                cy.assertDataExplorerContains(testCollection3.name, true);
                cy.assertDataExplorerContains(testCollection4.name, true);

                //assert toolbar buttons
                cy.doDataExplorerSelect(testCollection1.name);
                cy.assertToolbarButtons(tooltips.adminCollection);
                cy.doDataExplorerSelect(testCollection2.name);
                cy.assertToolbarButtons(tooltips.multiCollection);

                //check multi-collection move to
                cy.get(`[aria-label="Move to"]`, { timeout: 5000 }).click();
                cy.get('[data-cy=picker-dialog-project-search]').find('input').type(testProject1.name);
                cy.get('[data-cy=projects-tree-search-picker]').contains(testProject1.name).click();
                cy.get('[data-cy=form-submit-btn]').click();

                cy.assertDataExplorerContains(testProject1.name, true).click();
                cy.assertDataExplorerContains(testCollection1.name, true);
                cy.assertDataExplorerContains(testCollection2.name, true);

                //check multi-collection trash
                cy.doDataExplorerSelect(testCollection1.name);
                cy.doDataExplorerSelect(testCollection2.name);
                cy.doToolbarAction('Move to trash');
                cy.assertDataExplorerContains(testCollection1.name, false);
                cy.assertDataExplorerContains(testCollection2.name, false);
                cy.waitForDom()
                cy.contains('Trash').click();
                cy.assertDataExplorerContains(testCollection1.name, true);
                cy.assertDataExplorerContains(testCollection2.name, true);

                //check multi-collection unTrash
                cy.doDataExplorerSelect(testCollection1.name);
                cy.doDataExplorerSelect(testCollection2.name);
                cy.doToolbarAction('Restore');
                cy.assertDataExplorerContains(testCollection1.name, false);
                cy.assertDataExplorerContains(testCollection2.name, false);
                cy.contains(testProject1.name).click();
                cy.assertDataExplorerContains(testCollection1.name, true);
                cy.assertDataExplorerContains(testCollection2.name, true);

                //share with active user to test readonly permissions
                cy.shareWith(adminUser.token, activeUser.user.uuid, testProject1.uuid, 'can_read');

                //check read only project toolbar buttons
                cy.loginAs(activeUser);
                cy.contains('Shared with me').click();
                cy.doDataExplorerSelect(testProject1.name);
                cy.assertToolbarButtons(tooltips.readOnlyProject);
                cy.get("[data-cy=data-table-row]").contains(testProject1.name).click();
                cy.doDataExplorerSelect(testCollection1.name);
                cy.assertToolbarButtons(tooltips.readonlyCollection);
                cy.doDataExplorerSelect(testCollection2.name);
                cy.assertToolbarButtons(tooltips.readonlyMultiCollection);
            }
        );
    });
});

describe('For process resources', () => {
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

    it('should behave correctly for a single process', () => {
        createContainerRequest(
            adminUser,
            `test_container_request_1 ${Math.floor(Math.random() * 999999)}`,
            "arvados/jobs",
            ["echo", "hello world"],
            false,
            "Committed"
        ).as('testWorkflow');
        cy.getAll('@testWorkflow').then(([testWorkflow]) => {
            cy.loginAs(adminUser);
            cy.get('[data-cy=mpv-tabs]').contains("Workflow Runs").click();

            cy.doDataExplorerSelect(testWorkflow.name);
            cy.assertToolbarButtons(tooltips.adminRunningProcess);

            //Cancel process first to avoid unnecessary log polling
            cy.get('[aria-label="Cancel"]').click();
            cy.assertToolbarButtons(tooltips.adminOnHoldProcess);


            // disabled until #22787 is resolved
            // View details
            // cy.get('[aria-label="View details"]').click();
            // cy.get('[data-cy=details-panel]').contains(testWorkflow.name).should('be.visible');
            // cy.get('[data-cy=close-details-btn]').click();

            cy.window().then((win) => {
                cy.stub(win, 'open').as('windowOpen');
            });

            // Open in new tab
            cy.get('[aria-label="Open in new tab"]').click();
            cy.get('@windowOpen').should('be.called');

            //Copy and re-run process
            cy.get('[aria-label="Copy and re-run process"]').click();
            cy.get("[data-cy=form-dialog]").within(() => {
                cy.contains("Choose location for re-run").should('be.visible');
                cy.get("[data-cy=form-cancel-btn]").click();
            });

            //edit process
            cy.get('[aria-label="Edit process"]').click();
            cy.get("[data-cy=form-dialog]").within(() => {
                cy.contains("Edit Process").should('be.visible');
                cy.get("[data-cy=form-cancel-btn]").click();
            });

            //Outputs
            cy.get('[aria-label="Outputs"]').click();
            cy.contains('Output collection was trashed or deleted').should('exist');

            //Add to favorites
            cy.get('[aria-label="Add to favorites"]').click();
            cy.get('[data-cy=favorite-star]').should('exist')
                .parents('[data-cy=data-table-row]')
                .contains(testWorkflow.name)

            //Add to public favorites
            cy.get('[aria-label="Add to public favorites"]').click()
            cy.get('[data-cy=public-favorite-star]').should('exist')
                .parents('[data-cy=data-table-row]')
                .contains(testWorkflow.name)

            //API Details
            cy.get('[aria-label="API Details"]').click()
            cy.get('[role=dialog]').contains('API Details')
            cy.contains('Close').click()

            //Remove
            cy.get('[aria-label="Remove"]').click();
            cy.get('[data-cy=confirmation-dialog]').within(() => {
                cy.get('[data-cy=confirmation-dialog-ok-btn]').click();
            });
            cy.assertDataExplorerContains(testWorkflow.name, false);
        });
    });

    it('should behave correctly for multiple processes', () => {
        createContainerRequest(
            adminUser,
            `test_container_request_1 ${Math.floor(Math.random() * 999999)}`,
            "arvados/jobs",
            ["echo", "hello world"],
            false,
            "Committed"
        ).as('testProcess1');
        createContainerRequest(
            adminUser,
            `test_container_request_2 ${Math.floor(Math.random() * 999999)}`,
            "arvados/jobs",
            ["echo", "hello world"],
            false,
            "Committed"
        ).as('testProcess2');
        cy.getAll('@testProcess1', '@testProcess2').then(([testProcess1, testProcess2]) => {

            cy.loginAs(adminUser);
            cy.get('[data-cy=mpv-tabs]').contains("Workflow Runs").click();
            cy.assertDataExplorerContains(testProcess1.name, true);
            cy.assertDataExplorerContains(testProcess2.name, true);

            //assert toolbar buttons
            cy.doDataExplorerSelect(testProcess1.name);
            cy.assertToolbarButtons(tooltips.adminRunningProcess);
            cy.doDataExplorerSelect(testProcess2.name);
            cy.assertToolbarButtons(tooltips.multiProcess);

            //multiprocess remove
            cy.get('[aria-label="Remove"]').click();
            cy.get('[data-cy=confirmation-dialog]').within(() => {
                cy.get('[data-cy=confirmation-dialog-ok-btn]').click();
            });
            cy.assertDataExplorerContains(testProcess1.name, false);
            cy.assertDataExplorerContains(testProcess2.name, false);
        });
    });
});

describe('For multiple resource types', () => {
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

    it('shows the appropriate buttons in the multiselect toolbar', () => {
        cy.createProject({
            owningUser: adminUser,
            projectName: 'TestProject',
        }).as('testProject');
        cy.createCollection(adminUser.token, {
            name: `Test collection ${Math.floor(Math.random() * 999999)}`,
            owner_uuid: adminUser.user.uuid,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:bar\n",
        }).as("testCollection")
        createContainerRequest(
            adminUser,
            `test_container_request_1 ${Math.floor(Math.random() * 999999)}`,
            "arvados/jobs",
            ["echo", "hello world"],
            false,
            "Committed"
        ).as('testProcess');

        cy.getAll('@testProject', '@testCollection', '@testProcess')
            .then(([testProject, testCollection,  testProcess]) => {

            cy.loginAs(adminUser);
            //add resources to favorites so they are all in the same table
            cy.doDataExplorerSelect(testProject.name);
            cy.get('[aria-label="Add to favorites"]').click();
            //deselect project
            cy.doDataExplorerSelect(testProject.name);
            cy.doDataExplorerSelect(testCollection.name);
            cy.get('[aria-label="Add to favorites"]').click();
            cy.get('[data-cy=mpv-tabs]').contains("Workflow Runs").click();
            cy.doDataExplorerSelect(testProcess.name);
            cy.get('[aria-label="Add to favorites"]').click();

            cy.contains('My Favorites').click();

            cy.assertDataExplorerContains(testProject.name, true);
            cy.assertDataExplorerContains(testCollection.name, true);
            cy.assertDataExplorerContains(testProcess.name, true);

            cy.doDataExplorerSelect(testProject.name);
            cy.doDataExplorerSelect(testCollection.name);
            cy.assertToolbarButtons(tooltips.projectAndCollection);

            cy.get('[data-cy=data-table-multiselect-popover]').click();
            cy.get('[data-cy=multiselect-popover-None]').click();
            cy.doDataExplorerSelect(testProcess.name);
            cy.doDataExplorerSelect(testCollection.name);
            cy.assertToolbarButtons(tooltips.processAndCollection);

            cy.get('[data-cy=data-table-multiselect-popover]').click();
            cy.get('[data-cy=multiselect-popover-None]').click();
            cy.doDataExplorerSelect(testProcess.name);
            cy.doDataExplorerSelect(testProject.name);
            cy.assertToolbarButtons(tooltips.processAndProject);
        });
    });
});
    /*
    selecting/deselecting items should:
        select/deselect the correct items x
        display the correct toolbar items x
    select all/deselect all/invert selection in popover should:
        select/deselect the correct items x
        display the correct toolbar items x
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
    Data/Workflow runs tabs should have all of the functionality of the main process view x
    */
