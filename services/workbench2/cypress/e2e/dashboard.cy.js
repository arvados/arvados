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

describe('Main Dashboard', () => {
    let activeUser;
    let adminUser;

    const sectionsTitles = [
        'Favorites',
        'Recently Visited',
        'Recent Workflow Runs',
    ];

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

    it('Is the default starting page', () => {
        cy.loginAs(activeUser);
        cy.visit(`/`);
        cy.get('[data-cy=dashboard-root]').should('exist');
        cy.reload();
    });

    it('displays the appropriate sections', () => {
        cy.loginAs(activeUser);
        cy.get('[data-cy=tree-top-level-item]').contains('Dashboard').click();
        cy.get('[data-cy=dashboard-root]').should('exist');
        cy.get('[data-cy=breadcrumbs]').contains('Dashboard');
        cy.get('[data-cy=dashboard-root] [data-cy=dashboard-section]').should('have.length', sectionsTitles.length);
        sectionsTitles.forEach(title => {
            cy.get('[data-cy=dashboard-section]').contains(title).should('exist');
        });
    })
});

describe('Favorites section', () => {
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

    it('displays the favorites section', () => {
        cy.loginAs(activeUser);
        cy.get('[data-cy=tree-top-level-item]').contains('Dashboard').click();
        cy.get('[data-cy=dashboard-section]').contains('Favorites').should('exist');
    });

    it('handles favorite pins operations', () => {
        cy.createProject({
            owningUser: adminUser,
            projectName: 'TestProject1',
            addToFavorites: true,
        }).as('testProject1');
        cy.createProject({
            owningUser: adminUser,
            projectName: 'TestProject2',
            addToFavorites: true,
        }).as('testProject2');
        cy.createProject({
            owningUser: adminUser,
            projectName: 'TestProject3',
            addToFavorites: true,
        }).as('testProject3');
        cy.getAll('@testProject1', '@testProject2', '@testProject3').then(
            ([testProject1, testProject2, testProject3]) => {
                cy.loginAs(adminUser);
                cy.get('[data-cy=tree-top-level-item]').contains('Dashboard').click();
                cy.get('[data-cy=dashboard-section]').contains('Favorites').should('exist');

                //verify favorite pins
                cy.get('[data-cy=favorite-pin]').should('have.length', 3);
                cy.get('[data-cy=favorite-pin]').eq(0).contains('TestProject3')
                cy.get('[data-cy=favorite-pin]').eq(1).contains('TestProject2')
                cy.get('[data-cy=favorite-pin]').eq(2).contains('TestProject1')

                //remove favorite pin
                cy.get(`[data-cy=${testProject1.head_uuid}-star]`).click();
                cy.get('[data-cy=favorite-pin]').should('have.length', 2);
                cy.get('[data-cy=favorite-pin]').contains('TestProject1').should('not.exist');

                //add favorite pin
                cy.get('[data-cy=side-panel-tree]').contains('TestProject1').rightclick();
                cy.get('[data-cy=context-menu]').contains('Add to favorites').click();
                cy.get('[data-cy=favorite-pin]').should('have.length', 3);
                cy.get('[data-cy=favorite-pin]').contains('TestProject1');

                //verify ordered by last favorited
                cy.get('[data-cy=favorite-pin]').eq(0).contains('TestProject1');
                cy.get('[data-cy=favorite-pin]').eq(1).contains('TestProject3');
                cy.get('[data-cy=favorite-pin]').eq(2).contains('TestProject2');

                //opens context menu
                cy.get('[data-cy=favorite-pin]').contains('TestProject1').rightclick();
                cy.get('[data-cy=context-menu]').contains('TestProject1');
                cy.get('body').click();

                //navs to item
                cy.get('[data-cy=favorite-pin]').contains('TestProject1').click();
                cy.get('[data-cy=project-details-card]').contains('TestProject1').should('exist');
            });
        });
});

describe('Recently Visited section', () => {
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

    it('displays the recently visited section', () => {
        cy.loginAs(activeUser);
        cy.get('[data-cy=tree-top-level-item]').contains('Dashboard').click();
        cy.get('[data-cy=dashboard-section]').contains('Recently Visited').should('exist');
    });

    it('handles recently visited operations', () => {
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
        cy.getAll('@testProject1', '@testProject2', '@testProject3').then(
            ([testProject1, testProject2, testProject3]) => {
                cy.loginAs(adminUser);

                // visit some projects
                cy.get('[data-cy=side-panel-tree]').contains(testProject1.name).click();
                cy.get('[data-cy=project-details-card]').contains(testProject1.name).should('exist');
                cy.get('[data-cy=side-panel-tree]').contains(testProject2.name).click();
                cy.get('[data-cy=project-details-card]').contains(testProject2.name).should('exist');
                cy.get('[data-cy=side-panel-tree]').contains(testProject3.name).click();
                cy.get('[data-cy=project-details-card]').contains(testProject3.name).should('exist');
                    
                // verify recently visited
                cy.get('[data-cy=tree-top-level-item]').contains('Dashboard').click();
                cy.get('[data-cy=dashboard-section]').contains('Recently Visited').should('exist');
                cy.get('[data-cy=dashboard-item-row]').should('have.length', 3);
                cy.get('[data-cy=dashboard-item-row]').eq(0).contains(testProject3.name);
                cy.get('[data-cy=dashboard-item-row]').eq(1).contains(testProject2.name);
                cy.get('[data-cy=dashboard-item-row]').eq(2).contains(testProject1.name);

                // opens context menu
                cy.get('[data-cy=dashboard-item-row]').contains(testProject1.name).rightclick();
                cy.get('[data-cy=context-menu]').contains(testProject1.name);
                cy.get('body').click();

                // navs to item
                cy.get('[data-cy=dashboard-item-row]').contains(testProject1.name).click();
                cy.get('[data-cy=project-details-card]').contains(testProject1.name).should('exist');

                // verify recently visited order has changed
                cy.get('[data-cy=tree-top-level-item]').contains('Dashboard').click();
                cy.get('[data-cy=dashboard-section]').contains('Recently Visited').should('exist');
                cy.get('[data-cy=dashboard-item-row]').should('have.length', 3);
                cy.get('[data-cy=dashboard-item-row]').eq(0).contains(testProject1.name);
                cy.get('[data-cy=dashboard-item-row]').eq(1).contains(testProject3.name);
                cy.get('[data-cy=dashboard-item-row]').eq(2).contains(testProject2.name);
            });
        });
});

// describe('Recent Workflow Runs section', () => {
//     let activeUser;
//     let adminUser;

//     before(function () {
//         // Only set up common users once. These aren't set up as aliases because
//         // aliases are cleaned up after every test. Also it doesn't make sense
//         // to set the same users on beforeEach() over and over again, so we
//         // separate a little from Cypress' 'Best Practices' here.
//         cy.getUser('admin', 'Admin', 'User', true, true)
//             .as('adminUser')
//             .then(function () {
//                 adminUser = this.adminUser;
//             });
//         cy.getUser('user', 'Active', 'User', false, true)
//             .as('activeUser')
//             .then(function () {
//                 activeUser = this.activeUser;
//             });
//     });

//     it('displays the recent workflow runs section', () => {
//         cy.loginAs(activeUser);
//         cy.get('[data-cy=tree-top-level-item]').contains('Dashboard').click();
//         cy.get('[data-cy=dashboard-section]').contains('Recent Workflow Runs').should('exist');
//     });
// });
