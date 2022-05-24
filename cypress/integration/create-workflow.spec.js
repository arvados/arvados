// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

describe('Multi-file deletion tests', function () {
    let activeUser;
    let adminUser;

    before(function () {
        cy.getUser('admin', 'Admin', 'User', true, true)
            .as('adminUser').then(function () {
                adminUser = this.adminUser;
            }
            );
        cy.getUser('collectionuser1', 'Collection', 'User', false, true)
            .as('activeUser').then(function () {
                activeUser = this.activeUser;
            }
            );
    });

    beforeEach(function () {
        cy.clearCookies();
        cy.clearLocalStorage();
    });

    it('can create project with nested data', function () {
        cy.createGroup(adminUser.token, {
            group_class: "project",
            name: `Test project (${Math.floor(Math.random() * 999999)})`,
        }).as('project1');

        cy.get('@project1').then(() => {
            cy.createGroup(adminUser.token, {
                group_class: "project",
                name: `Test project (${Math.floor(Math.random() * 999999)})`,
                owner_uuid: this.project1.uuid,
            }).as('project2');
        })

        cy.get('@project2').then(() => {
            cy.createGroup(adminUser.token, {
                group_class: "project",
                name: `Test project (${Math.floor(Math.random() * 999999)})`,
                owner_uuid: this.project2.uuid,
            }).as('project3');
        });

        cy.get('@project3').then(() => {
            cy.createWorkflow(adminUser.token, {
                name: `TestWorkflow${Math.floor(Math.random() * 999999)}.cwl`,
                definition: "{\n    \"$graph\": [\n        {\n            \"class\": \"Workflow\",\n            \"doc\": \"Reverse the lines in a document, then sort those lines.\",\n            \"hints\": [\n                {\n                    \"acrContainerImage\": \"99b0201f4cade456b4c9d343769a3b70+261\",\n                    \"class\": \"http://arvados.org/cwl#WorkflowRunnerResources\"\n                }\n            ],\n            \"id\": \"#main\",\n            \"inputs\": [\n                {\n                    \"default\": null,\n                    \"doc\": \"The input file to be processed.\",\n                    \"id\": \"#main/input\",\n                    \"type\": \"File\"\n                },\n                {\n                    \"default\": true,\n                    \"doc\": \"If true, reverse (decending) sort\",\n                    \"id\": \"#main/reverse_sort\",\n                    \"type\": \"boolean\"\n                }\n            ],\n            \"outputs\": [\n                {\n                    \"doc\": \"The output with the lines reversed and sorted.\",\n                    \"id\": \"#main/output\",\n                    \"outputSource\": \"#main/sorted/output\",\n                    \"type\": \"File\"\n                }\n            ],\n            \"steps\": [\n                {\n                    \"id\": \"#main/rev\",\n                    \"in\": [\n                        {\n                            \"id\": \"#main/rev/input\",\n                            \"source\": \"#main/input\"\n                        }\n                    ],\n                    \"out\": [\n                        \"#main/rev/output\"\n                    ],\n                    \"run\": \"#revtool.cwl\"\n                },\n                {\n                    \"id\": \"#main/sorted\",\n                    \"in\": [\n                        {\n                            \"id\": \"#main/sorted/input\",\n                            \"source\": \"#main/rev/output\"\n                        },\n                        {\n                            \"id\": \"#main/sorted/reverse\",\n                            \"source\": \"#main/reverse_sort\"\n                        }\n                    ],\n                    \"out\": [\n                        \"#main/sorted/output\"\n                    ],\n                    \"run\": \"#sorttool.cwl\"\n                }\n            ]\n        },\n        {\n            \"baseCommand\": \"rev\",\n            \"class\": \"CommandLineTool\",\n            \"doc\": \"Reverse each line using the `rev` command\",\n            \"hints\": [\n                {\n                    \"class\": \"ResourceRequirement\",\n                    \"ramMin\": 8\n                }\n            ],\n            \"id\": \"#revtool.cwl\",\n            \"inputs\": [\n                {\n                    \"id\": \"#revtool.cwl/input\",\n                    \"inputBinding\": {},\n                    \"type\": \"File\"\n                }\n            ],\n            \"outputs\": [\n                {\n                    \"id\": \"#revtool.cwl/output\",\n                    \"outputBinding\": {\n                        \"glob\": \"output.txt\"\n                    },\n                    \"type\": \"File\"\n                }\n            ],\n            \"stdout\": \"output.txt\"\n        },\n        {\n            \"baseCommand\": \"sort\",\n            \"class\": \"CommandLineTool\",\n            \"doc\": \"Sort lines using the `sort` command\",\n            \"hints\": [\n                {\n                    \"class\": \"ResourceRequirement\",\n                    \"ramMin\": 8\n                }\n            ],\n            \"id\": \"#sorttool.cwl\",\n            \"inputs\": [\n                {\n                    \"id\": \"#sorttool.cwl/reverse\",\n                    \"inputBinding\": {\n                        \"position\": 1,\n                        \"prefix\": \"-r\"\n                    },\n                    \"type\": \"boolean\"\n                },\n                {\n                    \"id\": \"#sorttool.cwl/input\",\n                    \"inputBinding\": {\n                        \"position\": 2\n                    },\n                    \"type\": \"File\"\n                }\n            ],\n            \"outputs\": [\n                {\n                    \"id\": \"#sorttool.cwl/output\",\n                    \"outputBinding\": {\n                        \"glob\": \"output.txt\"\n                    },\n                    \"type\": \"File\"\n                }\n            ],\n            \"stdout\": \"output.txt\"\n        }\n    ],\n    \"cwlVersion\": \"v1.0\"\n}",
            })
                .as('testWorkflow');

            cy.createCollection(adminUser.token, {
                name: `Test collection ${Math.floor(Math.random() * 999999)}`,
                owner_uuid: this.project3.uuid,
                manifest_text: "./subdir 37b51d194a7513e45b56f6524f2d51f2+3 0:3:foo\n. 37b51d194a7513e45b56f6524f2d51f2+3 0:3:baz\n"
            })
                .as('testCollection');
        });

        cy.get('@testWorkflow').then(() => {
            cy.loginAs(adminUser);

            cy.get('[data-cy=side-panel-button]').click();
            cy.get('[data-cy=side-panel-run-process]').click();

            cy.get('.layout-pane')
                .contains(this.testWorkflow.name)
                .click();

            cy.get('[data-cy=run-process-next-button]').click();

            cy.get('[data-cy=new-process-panel]').contains('Run workflow').should('be.disabled');

            cy.get('[data-cy=new-process-panel]')
                .within(() => {
                    cy.get('[name=name]').type(`Workflow name (${Math.floor(Math.random() * 999999)})`);
                    cy.contains('input').next().click();
                });

            cy.get('[data-cy=choose-a-file-dialog]').as('chooseFileDialog');
            cy.get('@chooseFileDialog').contains('Projects').closest('ul').find('i').click();

            cy.get('@project1').then((project1) => {
                cy.get('@chooseFileDialog').find(`[data-id=${project1.uuid}]`).find('i').click();
            });

            cy.get('@project2').then((project2) => {
                cy.get('@chooseFileDialog').find(`[data-id=${project2.uuid}]`).find('i').click();
            });

            cy.get('@project3').then((project3) => {
                cy.get('@chooseFileDialog').find(`[data-id=${project3.uuid}]`).find('i').click();
            });

            cy.get('@testCollection').then((testCollection) => {
                cy.get('@chooseFileDialog').find(`[data-id=${testCollection.uuid}]`).find('i').click();
            });

            cy.get('@chooseFileDialog').contains('baz').click();

            cy.get('@chooseFileDialog').find('button').contains('Ok').click();

            cy.get('[data-cy=new-process-panel]')
                .find('button').contains('Run workflow').should('not.be.disabled');
        });
    });

    ['workflow_with_array_fields.yaml', 'workflow_with_default_array_fields.yaml'].forEach((yamlfile) =>
    it('can select multi files when creating workflow '+yamlfile, () => {
        cy.createProject({
            owningUser: activeUser,
            projectName: 'myProject1',
            addToFavorites: true
        });

        cy.createCollection(adminUser.token, {
            name: `Test collection ${Math.floor(Math.random() * 999999)}`,
            owner_uuid: activeUser.user.uuid,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:bar\n. 37b51d194a7513e45b56f6524f2d51f2+3 0:3:baz\n"
        })
            .as('testCollection');

        cy.createCollection(adminUser.token, {
            name: `Test collection ${Math.floor(Math.random() * 999999)}`,
            owner_uuid: activeUser.user.uuid,
            manifest_text: `. 37b51d194a7513e45b56f6524f2d51f2+3 0:3:buz\n`
        })
            .as('testCollection2');

        cy.getAll('@myProject1', '@testCollection', '@testCollection2')
            .then(function ([myProject1, testCollection, testCollection2]) {
                cy.readFile('cypress/fixtures/'+yamlfile).then(workflow => {
                    cy.createWorkflow(adminUser.token, {
                        name: `TestWorkflow${Math.floor(Math.random() * 999999)}.cwl`,
                        definition: workflow,
                        owner_uuid: myProject1.uuid,
                    })
                        .as('testWorkflow');
                });

                cy.loginAs(activeUser);

                cy.get('main').contains(myProject1.name).click();

                cy.get('[data-cy=side-panel-button]').click();

                cy.get('#aside-menu-list').contains('Run a workflow').click();

                cy.get('@testWorkflow')
                    .then((testWorkflow) => {
                        cy.get('main').contains(testWorkflow.name).click();
                        cy.get('[data-cy=run-process-next-button]').click();

                        cy.get('label').contains('foo').parent('div').find('input').click();
                        cy.get('div[role=dialog]')
                            .within(() => {
                                cy.get('p').contains('Projects').closest('div[role=button]')
                                    .within(() => {
                                        cy.get('svg[role=presentation]')
                                            .click({ multiple: true });
                                    });

                                cy.get(`[data-id=${testCollection.uuid}]`)
                                    .find('i').click();

                                cy.contains('bar').closest('[data-action=TOGGLE_ACTIVE]').parent().find('input[type=checkbox]').click();

                                cy.contains('baz').closest('[data-action=TOGGLE_ACTIVE]').parent().find('input[type=checkbox]').click();

                                cy.get('[data-cy=ok-button]').click();
                            });

                        cy.get('label').contains('bar').parent('div').find('input').click();
                        cy.get('div[role=dialog]')
                            .within(() => {
                                cy.get('p').contains('Projects').closest('div[role=button]')
                                    .within(() => {
                                        cy.get('svg[role=presentation]')
                                            .click({ multiple: true });
                                    });

                                cy.get(`[data-id=${testCollection.uuid}]`)
                                    .find('input[type=checkbox]').click();

                                cy.get(`[data-id=${testCollection2.uuid}]`)
                                    .find('input[type=checkbox]').click();

                                cy.get('[data-cy=ok-button]').click();
                            });
                    });

                cy.get('label').contains('foo').parent('div')
                    .within(() => {
                        cy.contains('baz');
                        cy.contains('bar');
                    });

                cy.get('label').contains('bar').parent('div')
                    .within(() => {
                        cy.contains(testCollection.name);
                        cy.contains(testCollection2.name);
                    });
            });
    }));
})
