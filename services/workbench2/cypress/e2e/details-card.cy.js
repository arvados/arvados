// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

const testWFDefinition = "{\n    \"$graph\": [\n        {\n            \"class\": \"Workflow\",\n            \"doc\": \"Reverse the lines in a document, then sort those lines.\",\n            \"hints\": [\n                {\n                    \"acrContainerImage\": \"99b0201f4cade456b4c9d343769a3b70+261\",\n                    \"class\": \"http://arvados.org/cwl#WorkflowRunnerResources\"\n                }\n            ],\n            \"id\": \"#main\",\n            \"inputs\": [\n                {\n                    \"default\": null,\n                    \"doc\": \"The input file to be processed.\",\n                    \"id\": \"#main/input\",\n                    \"type\": \"File\"\n                },\n                {\n                    \"default\": true,\n                    \"doc\": \"If true, reverse (decending) sort\",\n                    \"id\": \"#main/reverse_sort\",\n                    \"type\": \"boolean\"\n                }\n            ],\n            \"outputs\": [\n                {\n                    \"doc\": \"The output with the lines reversed and sorted.\",\n                    \"id\": \"#main/output\",\n                    \"outputSource\": \"#main/sorted/output\",\n                    \"type\": \"File\"\n                }\n            ],\n            \"steps\": [\n                {\n                    \"id\": \"#main/rev\",\n                    \"in\": [\n                        {\n                            \"id\": \"#main/rev/input\",\n                            \"source\": \"#main/input\"\n                        }\n                    ],\n                    \"out\": [\n                        \"#main/rev/output\"\n                    ],\n                    \"run\": \"#revtool.cwl\"\n                },\n                {\n                    \"id\": \"#main/sorted\",\n                    \"in\": [\n                        {\n                            \"id\": \"#main/sorted/input\",\n                            \"source\": \"#main/rev/output\"\n                        },\n                        {\n                            \"id\": \"#main/sorted/reverse\",\n                            \"source\": \"#main/reverse_sort\"\n                        }\n                    ],\n                    \"out\": [\n                        \"#main/sorted/output\"\n                    ],\n                    \"run\": \"#sorttool.cwl\"\n                }\n            ]\n        },\n        {\n            \"baseCommand\": \"rev\",\n            \"class\": \"CommandLineTool\",\n            \"doc\": \"Reverse each line using the `rev` command\",\n            \"hints\": [\n                {\n                    \"class\": \"ResourceRequirement\",\n                    \"ramMin\": 8\n                }\n            ],\n            \"id\": \"#revtool.cwl\",\n            \"inputs\": [\n                {\n                    \"id\": \"#revtool.cwl/input\",\n                    \"inputBinding\": {},\n                    \"type\": \"File\"\n                }\n            ],\n            \"outputs\": [\n                {\n                    \"id\": \"#revtool.cwl/output\",\n                    \"outputBinding\": {\n                        \"glob\": \"output.txt\"\n                    },\n                    \"type\": \"File\"\n                }\n            ],\n            \"stdout\": \"output.txt\"\n        },\n        {\n            \"baseCommand\": \"sort\",\n            \"class\": \"CommandLineTool\",\n            \"doc\": \"Sort lines using the `sort` command\",\n            \"hints\": [\n                {\n                    \"class\": \"ResourceRequirement\",\n                    \"ramMin\": 8\n                }\n            ],\n            \"id\": \"#sorttool.cwl\",\n            \"inputs\": [\n                {\n                    \"id\": \"#sorttool.cwl/reverse\",\n                    \"inputBinding\": {\n                        \"position\": 1,\n                        \"prefix\": \"-r\"\n                    },\n                    \"type\": \"boolean\"\n                },\n                {\n                    \"id\": \"#sorttool.cwl/input\",\n                    \"inputBinding\": {\n                        \"position\": 2\n                    },\n                    \"type\": \"File\"\n                }\n            ],\n            \"outputs\": [\n                {\n                    \"id\": \"#sorttool.cwl/output\",\n                    \"outputBinding\": {\n                        \"glob\": \"output.txt\"\n                    },\n                    \"type\": \"File\"\n                }\n            ],\n            \"stdout\": \"output.txt\"\n        }\n    ],\n    \"cwlVersion\": \"v1.0\"\n}"
const ResourceKinds = {
    USER: 'user',
    PROJECT: 'project',
    WORKFLOW: 'workflow',
    COLLECTION: 'collection',
    PROCESS: 'process',
};


describe('Base Details Card tests', function () {
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
        cy.getUser('activeUser1', 'Active', 'User', false, true)
            .as('activeUser')
            .then(function () {
                activeUser = this.activeUser;
            });
        cy.on('uncaught:exception', (err, runnable) => {
            console.error(err);
        });
    });

    beforeEach(function () {
        cy.clearCookies();
        cy.clearLocalStorage();
    });

    Object.values(ResourceKinds).forEach(resourceKind => {
        it(`Should display the ${resourceKind} details card`, () => {
            const { name, createResource, navToResource, extraAssertions } = getCardTestParams(activeUser, adminUser, resourceKind);

            createResource();
            cy.loginAs(adminUser);
            cy.doSidePanelNavigation('Home Projects');
            navToResource();

            cy.get(`[data-cy=${resourceKind}-details-card]`).should('be.visible');
            cy.get(`[data-cy=${resourceKind}-details-card]`).contains(name).should('be.visible');
            cy.get(`[data-cy=${resourceKind}-details-card]`).within(() => {
                cy.get('[data-cy=multiselect-toolbar]').should('exist');
            });

            if (extraAssertions) extraAssertions();
        });
    });
});


const getCardTestParams = (activeUser, adminUser, resourceKind) => {
    let name;
    switch (resourceKind) {
        case ResourceKinds.USER:
            return {
                name: adminUser.user.full_name,
                createResource: () => {},
                navToResource: () => {},
            };

        case ResourceKinds.PROJECT:
            name = `Test project (${Math.floor(999999 * Math.random())})`;
            return {
                name,
                createResource: () => cy.createProject({ owningUser: adminUser, projectName: name }),
                navToResource: () => {
                    cy.doMPVTabSelect("Data");
                    cy.get('main').contains(name).click()
                },
            };

        case ResourceKinds.WORKFLOW:
            name = `TestWorkflow${Math.floor(Math.random() * 999999)}.cwl`;
            return {
                name,
                createResource: () =>
                    cy.createWorkflow(adminUser.token, {
                        name,
                        definition: testWFDefinition,
                    }),
                navToResource: () => {
                    cy.doMPVTabSelect("Data");
                    cy.get('main').contains(name).click();
                },
            };

        case ResourceKinds.COLLECTION:
            name = `Test collection ${Math.floor(Math.random() * 999999)}`;
            return {
                name,
                createResource: () =>
                    cy.createCollection(adminUser.token, {
                        name,
                        owner_uuid: adminUser.user.uuid,
                        manifest_text: '. 37b51d194a7513e45b56f6524f2d51f2+3 0:3:bar\n',
                    }),
                navToResource: () => {
                    cy.doMPVTabSelect("Data");
                    cy.get('main').contains(name).click();
                },
            };

        case ResourceKinds.PROCESS:
            name = `Test container request ${Math.floor(Math.random() * 999999)}`;
            return {
                name,
                createResource: () => createContainerRequest(adminUser, name, 'arvados/jobs', ['echo', 'hello world'], false, 'Committed'),
                navToResource: () => {
                    cy.doMPVTabSelect("Workflow Runs");
                    cy.get('main').contains(name).click();
                },
                extraAssertions: () => {
                    cy.get(`[data-cy=process-details-card]`).within(() => {
                        cy.get('[data-cy=process-cancel-button]').should('exist');
                        cy.get('[data-cy=process-status-chip]').should('exist');
                    });
                },
            };

        default:
            throw new Error(`Unknown resource kind: ${resourceKind}`);
    }
};

function createContainerRequest(user, name, docker_image, command, reuse = false, state = "Uncommitted", ownerUuid) {
        return cy.setupDockerImage(docker_image).then(function (dockerImage) {
            return cy.createContainerRequest(user.token, {
                name: name,
                command: command,
                container_image: dockerImage.portable_data_hash, // for some reason, docker_image doesn't work here
                output_path: '/var/spool/cwl',
                priority: 1,
                runtime_constraints: {
                    vcpus: 1,
                    ram: 1,
                },
                use_existing: reuse,
                state: state,
                mounts: {
                    '/var/lib/cwl/workflow.json': {
                        kind: 'json',
                        content: {},
                    },
                    '/var/spool/cwl': {
                        kind: 'tmp',
                        capacity: 1000000,
                    },
                },
                owner_uuid: ownerUuid || undefined,
            });
        });
    }
