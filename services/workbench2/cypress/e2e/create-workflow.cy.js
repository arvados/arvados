// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

const testWFDefinition = "{\n    \"$graph\": [\n        {\n            \"class\": \"Workflow\",\n            \"doc\": \"Reverse the lines in a document, then sort those lines.\",\n            \"hints\": [\n                {\n                    \"acrContainerImage\": \"99b0201f4cade456b4c9d343769a3b70+261\",\n                    \"class\": \"http://arvados.org/cwl#WorkflowRunnerResources\"\n                }\n            ],\n            \"id\": \"#main\",\n            \"inputs\": [\n                {\n                    \"default\": null,\n                    \"doc\": \"The input file to be processed.\",\n                    \"id\": \"#main/input\",\n                    \"type\": \"File\"\n                },\n                {\n                    \"default\": true,\n                    \"doc\": \"If true, reverse (decending) sort\",\n                    \"id\": \"#main/reverse_sort\",\n                    \"type\": \"boolean\"\n                }\n            ],\n            \"outputs\": [\n                {\n                    \"doc\": \"The output with the lines reversed and sorted.\",\n                    \"id\": \"#main/output\",\n                    \"outputSource\": \"#main/sorted/output\",\n                    \"type\": \"File\"\n                }\n            ],\n            \"steps\": [\n                {\n                    \"id\": \"#main/rev\",\n                    \"in\": [\n                        {\n                            \"id\": \"#main/rev/input\",\n                            \"source\": \"#main/input\"\n                        }\n                    ],\n                    \"out\": [\n                        \"#main/rev/output\"\n                    ],\n                    \"run\": \"#revtool.cwl\"\n                },\n                {\n                    \"id\": \"#main/sorted\",\n                    \"in\": [\n                        {\n                            \"id\": \"#main/sorted/input\",\n                            \"source\": \"#main/rev/output\"\n                        },\n                        {\n                            \"id\": \"#main/sorted/reverse\",\n                            \"source\": \"#main/reverse_sort\"\n                        }\n                    ],\n                    \"out\": [\n                        \"#main/sorted/output\"\n                    ],\n                    \"run\": \"#sorttool.cwl\"\n                }\n            ]\n        },\n        {\n            \"baseCommand\": \"rev\",\n            \"class\": \"CommandLineTool\",\n            \"doc\": \"Reverse each line using the `rev` command\",\n            \"hints\": [\n                {\n                    \"class\": \"ResourceRequirement\",\n                    \"ramMin\": 8\n                }\n            ],\n            \"id\": \"#revtool.cwl\",\n            \"inputs\": [\n                {\n                    \"id\": \"#revtool.cwl/input\",\n                    \"inputBinding\": {},\n                    \"type\": \"File\"\n                }\n            ],\n            \"outputs\": [\n                {\n                    \"id\": \"#revtool.cwl/output\",\n                    \"outputBinding\": {\n                        \"glob\": \"output.txt\"\n                    },\n                    \"type\": \"File\"\n                }\n            ],\n            \"stdout\": \"output.txt\"\n        },\n        {\n            \"baseCommand\": \"sort\",\n            \"class\": \"CommandLineTool\",\n            \"doc\": \"Sort lines using the `sort` command\",\n            \"hints\": [\n                {\n                    \"class\": \"ResourceRequirement\",\n                    \"ramMin\": 8\n                }\n            ],\n            \"id\": \"#sorttool.cwl\",\n            \"inputs\": [\n                {\n                    \"id\": \"#sorttool.cwl/reverse\",\n                    \"inputBinding\": {\n                        \"position\": 1,\n                        \"prefix\": \"-r\"\n                    },\n                    \"type\": \"boolean\"\n                },\n                {\n                    \"id\": \"#sorttool.cwl/input\",\n                    \"inputBinding\": {\n                        \"position\": 2\n                    },\n                    \"type\": \"File\"\n                }\n            ],\n            \"outputs\": [\n                {\n                    \"id\": \"#sorttool.cwl/output\",\n                    \"outputBinding\": {\n                        \"glob\": \"output.txt\"\n                    },\n                    \"type\": \"File\"\n                }\n            ],\n            \"stdout\": \"output.txt\"\n        }\n    ],\n    \"cwlVersion\": \"v1.0\"\n}"

describe('Create workflow tests', function () {
    let activeUser;
    let adminUser;

    before(function () {
        cy.getUser('admin', 'Admin', 'User', true, true)
            .as('adminUser').then(function () {
                adminUser = this.adminUser;
            }
            );
        cy.getUser('activeuser', 'Active', 'User', false, true)
            .as('activeUser').then(function () {
                activeUser = this.activeUser;
            }
            );
            
    });

    function createNestedHelper(testRemainder) {
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
                definition: testWFDefinition,
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
            cy.get('[data-cy=run-wf-project-picker-ok-button]').click();

            cy.get('[data-cy=new-process-panel]')
                .within(() => {
                    cy.get('[name=name]').type(`Workflow name (${Math.floor(Math.random() * 999999)})`);
                    cy.contains('input').next().click();
                });

            testRemainder();

            cy.get('[data-cy=new-process-panel]')
                .find('button').contains('Run workflow').should('not.be.disabled');
        });
    }

    it('can create project with nested data', function () {
        this.createNestedHelper = createNestedHelper;
        this.createNestedHelper(() => {
            cy.get('[data-cy=choose-a-file-dialog]').as('chooseFileDialog');
            cy.get('@chooseFileDialog').contains('Home Projects')
                .parents('[data-cy=tree-li]')
                .find('[data-cy=side-panel-arrow-icon]')
                .click();

            cy.get('@project1').then((project1) => {
                cy.get('@chooseFileDialog').find(`[data-id=${project1.uuid}]`);
                cy.get('@chooseFileDialog')
                    .find(`[data-id=${project1.uuid}]`)
                    .find('[data-action=TOGGLE_ACTIVE]')
                    .click();
                cy.get('[data-cy=picker-dialog-details]')
                    .contains("Project");
                cy.get('[data-cy=picker-dialog-details]')
                    .contains(project1.uuid);
                cy.get('@chooseFileDialog')
                    .find(`[data-id=${project1.uuid}]`)
                    .find('[data-action=TOGGLE_OPEN]')
                    .click();
            });

            cy.get('@project2').then((project2) => {
                cy.get('@chooseFileDialog').find(`[data-id=${project2.uuid}]`);
                cy.get('@chooseFileDialog')
                    .find(`[data-id=${project2.uuid}]`)
                    .find('[data-action=TOGGLE_ACTIVE]')
                    .click();
                cy.get('[data-cy=picker-dialog-details]')
                    .contains("Project");
                cy.get('[data-cy=picker-dialog-details]')
                    .contains(project2.uuid);
                cy.get('@chooseFileDialog')
                    .find(`[data-id=${project2.uuid}]`)
                    .find('[data-action=TOGGLE_OPEN]')
                    .click();
            });

            cy.get('@project3').then((project3) => {
                cy.get('@chooseFileDialog').find(`[data-id=${project3.uuid}]`);
                cy.get('@chooseFileDialog')
                    .find(`[data-id=${project3.uuid}]`)
                    .find('[data-action=TOGGLE_ACTIVE]')
                    .click();
                cy.get('[data-cy=picker-dialog-details]')
                    .contains("Project");
                cy.get('[data-cy=picker-dialog-details]')
                    .contains(project3.uuid);
                cy.get('@chooseFileDialog')
                    .find(`[data-id=${project3.uuid}]`)
                    .find('[data-action=TOGGLE_OPEN]')
                    .click();
            });

            cy.get('@testCollection').then((testCollection) => {
                cy.get('@chooseFileDialog').find(`[data-id=${testCollection.uuid}]`);
                cy.get('@chooseFileDialog')
                    .find(`[data-id=${testCollection.uuid}]`)
                    .find('[data-action=TOGGLE_ACTIVE]')
                    .click();
                cy.get('[data-cy=picker-dialog-details]')
                    .contains("Collection");
                cy.get('[data-cy=picker-dialog-details]')
                    .contains(testCollection.uuid);
                cy.get('@chooseFileDialog')
                    .find(`[data-id=${testCollection.uuid}]`)
                    .find('[data-action=TOGGLE_OPEN]')
                    .click();
            });

            cy.get('@chooseFileDialog').contains('baz').click();
            cy.get('[data-cy=picker-dialog-details]')
                .contains("File");

            cy.get('@chooseFileDialog').find('button').contains('Ok').click();
        });
    });

    it('can search for nested project by name', function () {
        this.createNestedHelper = createNestedHelper;
        this.createNestedHelper(() => {
            cy.get('[data-cy=choose-a-file-dialog]').as('chooseFileDialog');

            cy.get('@project3').then((project3) => {
                cy.get('[data-cy=picker-dialog-project-search]')
                    .find('[data-cy=search-input]')
                    .type(project3.name)

                cy.waitForDom();

                cy.get('@chooseFileDialog')
                    .find(`[data-id=${project3.uuid}]`)
                    .find('[data-action=TOGGLE_OPEN]')
                    .click();

                cy.get('@testCollection').then((testCollection) => {
                    cy.get('@chooseFileDialog').find(`[data-id=${testCollection.uuid}]`).find('i').click();
                });

                cy.get('@chooseFileDialog').contains('baz').click();

                cy.get('@chooseFileDialog').find('button').contains('Ok').click();
            });
        });
    });

    it('can search for nested project by uuid', function () {
        this.createNestedHelper = createNestedHelper;
        this.createNestedHelper(() => {
            cy.get('[data-cy=choose-a-file-dialog]').as('chooseFileDialog');

            cy.get('@project3').then((project3) => {
                cy.get('[data-cy=picker-dialog-project-search]')
                    .find('[data-cy=search-input]')
                    .type(project3.uuid)

                cy.waitForDom();

                cy.get('@chooseFileDialog')
                    .find(`[data-id=${project3.uuid}]`)
                    .find('[data-action=TOGGLE_OPEN]')
                    .click();

                cy.get('@testCollection').then((testCollection) => {
                    cy.get('@chooseFileDialog').find(`[data-id=${testCollection.uuid}]`).find('i').click();
                });

                cy.get('@chooseFileDialog').contains('baz').click();

                cy.get('@chooseFileDialog').find('button').contains('Ok').click();
            });
        });
    });


    it('can search for collection by name', function () {
        this.createNestedHelper = createNestedHelper;
        this.createNestedHelper(() => {
            cy.get('[data-cy=choose-a-file-dialog]').as('chooseFileDialog');

            cy.get('@testCollection').then((testCollection) => {
                cy.get('[data-cy=picker-dialog-collection-search]')
                    .find('[data-cy=search-input]')
                    .type(testCollection.name)

                cy.waitForDom();

                cy.get('@testCollection').then((testCollection) => {
                    cy.get('@chooseFileDialog')
                        .find(`[data-id=${testCollection.uuid}]`)
                        .find('[data-action=TOGGLE_OPEN]')
                        .click();
                });

                cy.get('@chooseFileDialog').contains('baz').click();

                cy.get('@chooseFileDialog').find('button').contains('Ok').click();
            });
        });
    });

    it('can search for collection by uuid', function () {
        this.createNestedHelper = createNestedHelper;
        this.createNestedHelper(() => {
            cy.get('[data-cy=choose-a-file-dialog]').as('chooseFileDialog');

            cy.get('@testCollection').then((testCollection) => {
                cy.get('[data-cy=picker-dialog-collection-search]')
                    .find('[data-cy=search-input]')
                    .type(testCollection.uuid)

                cy.waitForDom();

                cy.get('@testCollection').then((testCollection) => {
                    cy.get('@chooseFileDialog')
                        .find(`[data-id=${testCollection.uuid}]`)
                        .find('[data-action=TOGGLE_OPEN]')
                        .click();
                });

                cy.get('@chooseFileDialog').contains('baz').click();

                cy.get('@chooseFileDialog').find('button').contains('Ok').click();
            });
        });
    });

    it('can search for collection by PDH', function () {
        this.createNestedHelper = createNestedHelper;
        this.createNestedHelper(() => {
            cy.get('[data-cy=choose-a-file-dialog]').as('chooseFileDialog');

            cy.get('@testCollection').then((testCollection) => {
                cy.get('[data-cy=picker-dialog-collection-search]')
                    .find('[data-cy=search-input]')
                    .type(testCollection.portable_data_hash)

                cy.waitForDom();

                cy.get('@testCollection').then((testCollection) => {
                    cy.get('@chooseFileDialog')
                        .find(`[data-id=${testCollection.uuid}]`)
                        .find('[data-action=TOGGLE_OPEN]')
                        .click();
                });

                cy.get('@chooseFileDialog').contains('baz').click();

                cy.get('@chooseFileDialog').find('button').contains('Ok').click();
            });
        });
    });

    it('can pick a parent project from the project picker when starting from +NEW button', function () {
        cy.createGroup(adminUser.token, {
            group_class: 'project',
            name: `Test project (${Math.floor(Math.random() * 999999)})`,
        }).as('project1');

        cy.createGroup(adminUser.token, {
            group_class: 'project',
            name: `Test project (${Math.floor(Math.random() * 999999)})`,
        }).as('project2');

        cy.get('@project1').then(() => {
            cy.createWorkflow(adminUser.token, {
                name: `TestWorkflow${Math.floor(Math.random() * 999999)}.cwl`,
                definition: testWFDefinition,
            }).as('testWorkflow');
        });

        cy.getAll('@project1', '@project2', '@testWorkflow').then(([project1, project2, testWorkflow]) => {
            cy.loginAs(adminUser);

            cy.get('[data-cy=side-panel-button]').click();
            cy.get('[data-cy=side-panel-run-process]').click();

            cy.get('.layout-pane').contains(this.testWorkflow.name).click();

            cy.get('[data-cy=run-process-next-button]').click();

            cy.get('[data-cy=new-process-panel]').contains('Run workflow').should('be.disabled');
            cy.get('[data-cy=run-wf-project-picker-ok-button]').click();

            cy.get('[data-cy=new-process-panel]').within(() => {
                cy.get('[name=name]').type(`Workflow name (${Math.floor(Math.random() * 999999)})`);
            });

            //check that the default owner project is correct
            cy.get(`input[value="Admin User (root project)"]`).should('exist');
            cy.get('[data-cy=run-wf-project-input]').click();
            cy.get('[data-cy=project-picker-details]').contains('Admin User (root project)');
            //selecting a project should update the details element
            cy.get('[data-cy=projects-tree-home-tree-picker]').contains(project1.name).click();
            cy.get('[data-cy=project-picker-details]').contains(project1.name);
            cy.get('[data-cy=projects-tree-home-tree-picker]').contains(project2.name).click();
            cy.get('[data-cy=project-picker-details]').contains(project2.name);
            cy.get('[data-cy=projects-tree-home-tree-picker]').contains(project1.name).click();
            cy.get('[data-cy=project-picker-details]').contains(project1.name);
            //canceling should reset the details element
            cy.get('[data-cy=run-wf-project-picker-cancel-button]').click();
            cy.get(`input[value="Admin User (root project)"]`).should('exist');
            //we should be able to change the selection with the 'OK' button
            cy.get('[data-cy=run-wf-project-input]').click();
            cy.get('[data-cy=projects-tree-home-tree-picker]').contains(project1.name).click();
            cy.get('[data-cy=project-picker-details]').contains(project1.name);
            cy.get('[data-cy=run-wf-project-picker-ok-button]').click();
            cy.get(`input[value="${project1.name}"]`).should('exist');
            cy.get('[data-cy=run-wf-project-input]').click();
            cy.get('[data-cy=projects-tree-home-tree-picker]').contains(project2.name).click();
            cy.get('[data-cy=project-picker-details]').contains(project2.name);
            cy.get('[data-cy=run-wf-project-picker-ok-button]').click();
            cy.get(`input[value="${project2.name}"]`).should('exist');
            //should be able to re-select root project
            cy.get('[data-cy=run-wf-project-input]').click();
            cy.get('[data-cy=projects-tree-home-tree-picker]').contains("Home Projects").click();
            // wait for tree node to expand
            cy.waitForDom();
            cy.wait(1000);
            cy.get('[data-cy=projects-tree-home-tree-picker]').contains("Home Projects").should('exist', {timeout: 10000}).click();
            cy.get('[data-cy=project-picker-details]').contains('Admin User (root project)');
            cy.get('[data-cy=run-wf-project-picker-ok-button]').click();
            cy.get(`input[value="Admin User (root project)"]`).should('exist');
        });
    });

    it('can pick a parent project from the project picker starting from toolbar or context menu', function () {
        cy.createGroup(adminUser.token, {
            group_class: 'project',
            name: `Test project (${Math.floor(Math.random() * 999999)})`,
        }).as('project1');

        cy.createGroup(adminUser.token, {
            group_class: 'project',
            name: `Test project (${Math.floor(Math.random() * 999999)})`,
        }).as('project2');

        cy.get('@project1').then(() => {
            cy.createWorkflow(adminUser.token, {
                name: `TestWorkflow${Math.floor(Math.random() * 999999)}.cwl`,
                definition: testWFDefinition,
            }).as('testWorkflow');
        });

        cy.getAll('@project1', '@project2', '@testWorkflow').then(([project1, project2, testWorkflow]) => {
            cy.loginAs(adminUser);

            cy.get('.layout-pane').contains(this.testWorkflow.name).rightclick();
            cy.get('[data-cy=context-menu]').within(() => {
                cy.contains('Run Workflow').click();
            });

            //check that the default owner project is correct
            cy.get('[data-cy=project-picker-details]').contains('Admin User (root project)');
            //selecting a project should update the details element
            cy.get('[data-cy=projects-tree-home-tree-picker]').contains(project1.name).click();
            cy.get('[data-cy=project-picker-details]').contains(project1.name);
            cy.get('[data-cy=projects-tree-home-tree-picker]').contains(project2.name).click();
            cy.get('[data-cy=project-picker-details]').contains(project2.name);
            cy.get('[data-cy=projects-tree-home-tree-picker]').contains(project1.name).click();
            cy.get('[data-cy=project-picker-details]').contains(project1.name);
            //canceling should reset the details element
            cy.get('[data-cy=run-wf-project-picker-cancel-button]').click();
            cy.get(`input[value="Admin User (root project)"]`).should('exist');
            //we should be able to change the selection with the 'OK' button
            cy.get('[data-cy=run-wf-project-input]').click();
            cy.get('[data-cy=projects-tree-home-tree-picker]').contains(project1.name).click();
            cy.get('[data-cy=project-picker-details]').contains(project1.name);
            cy.get('[data-cy=run-wf-project-picker-ok-button]').click();
            cy.get(`input[value="${project1.name}"]`).should('exist');
            cy.get('[data-cy=run-wf-project-input]').click();
            cy.get('[data-cy=projects-tree-home-tree-picker]').contains(project2.name).click();
            cy.get('[data-cy=project-picker-details]').contains(project2.name);
            cy.get('[data-cy=run-wf-project-picker-ok-button]').click();
            cy.get(`input[value="${project2.name}"]`).should('exist');
            //should be able to re-select root project
            cy.get('[data-cy=run-wf-project-input]').click();
            cy.get('[data-cy=projects-tree-home-tree-picker]').contains("Home Projects").click();
            // wait for tree node to expand
            cy.waitForDom();
            cy.wait(1000);
            cy.get('[data-cy=projects-tree-home-tree-picker]').contains("Home Projects").should('exist', {timeout: 10000}).click();
            cy.get('[data-cy=project-picker-details]').contains('Admin User (root project)');
            cy.get('[data-cy=run-wf-project-picker-ok-button]').click();
            cy.get(`input[value="Admin User (root project)"]`).should('exist');
        });
    });

    it('respects write permissions in the project picker', function () {
        cy.loginAs(adminUser);

        cy.createGroup(adminUser.token, {
            name: `my-shared-writable-project ${Math.floor(Math.random() * 999999)}`,
            group_class: 'project',
        }).as('mySharedWritableProject').then(function (mySharedWritableProject) {
            cy.createWorkflow(adminUser.token, {
                name: `TestWorkflow${Math.floor(Math.random() * 999999)}.cwl`,
                definition: testWFDefinition,
                owner_uuid: mySharedWritableProject.uuid,
                }).as('parentWritableWF');
            cy.contains('Refresh').click();
            cy.get('main').contains(mySharedWritableProject.name).rightclick();
            cy.get('[data-cy=context-menu]').within(() => {
                cy.contains('Share').click({ waitForAnimations: false });
            });
            cy.get('[data-cy=permission-select]').as('permissionSelect');
            cy.get('@permissionSelect').click();
            cy.contains('Write').click();
            cy.get('.sharing-dialog').as('sharingDialog');
            cy.get('[data-cy=invite-people-field]').find('input').type(activeUser.user.email);
            cy.get('[data-cy="loading-spinner"]').should('not.exist');
            cy.get('[data-cy="users-tab-label"]').click();
            cy.get('[data-cy=sharing-suggestion]').click();
            cy.get('@sharingDialog').within(() => {
                cy.get('[data-cy=add-invited-people]').click();
                cy.contains('Close').click({ waitForAnimations: false });
            });
        });

        cy.createGroup(adminUser.token, {
            name: `my-shared-readonly-project ${Math.floor(Math.random() * 999999)}`,
            group_class: 'project',
        }).as('mySharedReadonlyProject').then(function (mySharedReadonlyProject) {
            cy.createWorkflow(adminUser.token, {
                name: `(readonly) TestWorkflow${Math.floor(Math.random() * 999999)}.cwl`,
                definition: testWFDefinition,
                owner_uuid: mySharedReadonlyProject.uuid,
                }).as('parentReadonlyWF');
            cy.contains('Refresh').click();
            cy.get('main').contains(mySharedReadonlyProject.name).rightclick();
            cy.get('[data-cy=context-menu]').within(() => {
                cy.contains('Share').click({ waitForAnimations: false });
            });
            cy.get('.sharing-dialog').as('sharingDialog');
            cy.get('[data-cy=invite-people-field]').find('input').type(activeUser.user.email);
            cy.get('[data-cy="loading-spinner"]').should('not.exist');
            cy.get('[data-cy="users-tab-label"]').click();
            cy.get('[data-cy=sharing-suggestion]').click();
            cy.get('@sharingDialog').within(() => {
                cy.get('[data-cy=add-invited-people]').click();
                cy.contains('Close').click({ waitForAnimations: false });
            });
        });

        cy.loginAs(activeUser);
        cy.createGroup(activeUser.token, {
            name: `non-admin-readonly-project ${Math.floor(Math.random() * 999999)}`,
            group_class: 'project',
        }).as('nonAdminReadonlyProject').then(function (nonAdminReadonlyProject) {
            cy.createWorkflow(activeUser.token, {
                name: `(non-admin, readonly) TestWorkflow${Math.floor(Math.random() * 999999)}.cwl`,
                definition: testWFDefinition,
                owner_uuid: nonAdminReadonlyProject.uuid,
                }).as('nonAdminReadonlyWF');
            cy.contains('Refresh').click();
            cy.get('main').contains(nonAdminReadonlyProject.name).rightclick();
            cy.get('[data-cy=context-menu]').within(() => {
                cy.contains('Share').click({ waitForAnimations: false });
            });
            cy.get('.sharing-dialog').as('sharingDialog');
            cy.get('[data-cy=invite-people-field]').find('input').type(adminUser.user.email);
            cy.get('[data-cy="loading-spinner"]').should('not.exist');
            cy.get('[data-cy="users-tab-label"]').click();
            cy.get('[data-cy=sharing-suggestion]').click();
            cy.get('@sharingDialog').within(() => {
                cy.get('[data-cy=add-invited-people]').click();
                cy.contains('Close').click({ waitForAnimations: false });
            });
        });

        cy.getAll('@parentWritableWF', '@parentReadonlyWF', '@mySharedWritableProject', '@mySharedReadonlyProject', '@nonAdminReadonlyProject', '@nonAdminReadonlyWF')
        .then(([parentWritableWF, parentReadonlyWF, mySharedWritableProject, mySharedReadonlyProject, nonAdminReadonlyProject, nonAdminReadonlyWF]) => {
            // already logged in as activeUser from previous step

            // a non-admin can run a wf in a writable project
            cy.contains('Shared with me').click();
            cy.contains(mySharedWritableProject.name).click();
            cy.contains(parentWritableWF.name).click();
            cy.get('[data-cy=workflow-details-panel-run-btn]').click();
            cy.get('[data-cy=project-picker-details]').contains(mySharedWritableProject.name);
            cy.get('[data-cy=run-wf-project-picker-ok-button]').click();
            cy.get(`input[value="${mySharedWritableProject.name}"]`).should('exist');

            // a non-admin cannot run a wf in a non-writable project, it defaults to the user's root project instead
            cy.contains('Shared with me').click();
            cy.contains(mySharedReadonlyProject.name).click();
            cy.contains(parentReadonlyWF.name).click();
            cy.get('[data-cy=workflow-details-panel-run-btn]').click();
            cy.get('[data-cy=project-picker-details]').contains("Active User (root project)");
            cy.get('[data-cy=run-wf-project-picker-ok-button]').click();
            cy.get(`input[value="Active User (root project)"]`).should('exist');

            //using +NEW button in Home Projects should default to user's root project
            cy.contains('Home Projects').click();
            cy.get('[data-cy=side-panel-button]').click();
            cy.get('[data-cy=side-panel-run-process]').click();
            cy.contains(parentWritableWF.name).click();
            cy.get('[data-cy=run-process-next-button]').click();
            cy.get('[data-cy=project-picker-details]').contains("Active User (root project)");
            cy.get('[data-cy=run-wf-project-picker-ok-button]').click();

            //using +NEW button to run a wf in a writable project should default to that writable project
            cy.contains('Shared with me').click();
            cy.contains(mySharedWritableProject.name).click();
            cy.get('[data-cy=side-panel-button]').click();
            cy.get('[data-cy=side-panel-run-process]').click();
            cy.contains(parentWritableWF.name).click();
            cy.get('[data-cy=run-process-next-button]').click();
            cy.get('[data-cy=project-picker-details]').contains(mySharedWritableProject.name);
            cy.get('[data-cy=run-wf-project-picker-ok-button]').click();

            // admin should be able to launch wf in shared readonly project
            cy.loginAs(adminUser);
            cy.contains('Shared with me').click();
            cy.contains(nonAdminReadonlyProject.name).click();
            cy.get('[data-cy=side-panel-button]').click();
            cy.get('[data-cy=side-panel-run-process]').click();
            cy.contains(nonAdminReadonlyWF.name).click();
            cy.get('[data-cy=run-process-next-button]').click();
            cy.get('[data-cy=project-picker-details]').contains(nonAdminReadonlyProject.name);
            cy.get('[data-cy=run-wf-project-picker-ok-button]').click();
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

                cy.waitForDom();

                cy.get('[data-cy=side-panel-button]').click();

                cy.get('#aside-menu-list').contains('Run a workflow').click();

                cy.get('@testWorkflow')
                    .then((testWorkflow) => {
                        cy.get('main').contains(testWorkflow.name).click();
                        cy.get('[data-cy=run-process-next-button]').click();
                        cy.get('[data-cy=run-wf-project-picker-ok-button]').click();

                        cy.get('label').contains('foo').parent('div').find('input').click();
                        cy.get('div[role=dialog]')
                            .within(() => {
                                // must use .then to avoid selecting instead of expanding https://github.com/cypress-io/cypress/issues/5529
                                cy.get('p').contains('Home Projects').closest('ul')
                                    .find('i')
                                    .then(el => el.click());

                                cy.get(`[data-id=${testCollection.uuid}]`)
                                    .find('i').click();

                                cy.wait(1000);
                                cy.contains('bar').closest('[data-action=TOGGLE_ACTIVE]').parent().find('input[type=checkbox]').click();
                                cy.contains('baz').closest('[data-action=TOGGLE_ACTIVE]').parent().find('input[type=checkbox]').click();

                                cy.get('[data-cy=ok-button]').click();
                            });

                        cy.get('label').contains('bar').parent('div').find('input').click();
                        cy.get('div[role=dialog]')
                            .within(() => {
                                // must use .then to avoid selecting instead of expanding https://github.com/cypress-io/cypress/issues/5529
                                cy.get('p').contains('Home Projects').closest('ul')
                                    .find('i')
                                    .then(el => el.click());

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

    it('allows selecting collection subdirectories and reselects existing selections', () => {
        cy.createProject({
            owningUser: activeUser,
            projectName: 'myProject1',
            addToFavorites: true
        });

        cy.createCollection(adminUser.token, {
            name: `Test collection ${Math.floor(Math.random() * 999999)}`,
            owner_uuid: activeUser.user.uuid,
            manifest_text: "./subdir/dir1 d41d8cd98f00b204e9800998ecf8427e+0 0:0:\\056\n./subdir/dir2 d41d8cd98f00b204e9800998ecf8427e+0 0:0:\\056\n"
        })
            .as('testCollection');

        cy.getAll('@myProject1', '@testCollection')
            .then(function ([myProject1, testCollection]) {
                cy.readFile('cypress/fixtures/workflow_directory_array.yaml').then(workflow => {
                    cy.createWorkflow(adminUser.token, {
                        name: `TestWorkflow${Math.floor(Math.random() * 999999)}.cwl`,
                        definition: workflow,
                        owner_uuid: myProject1.uuid,
                    })
                        .as('testWorkflow');
                });

                cy.loginAs(activeUser);

                cy.get('main').contains(myProject1.name).click();

                cy.waitForDom();

                cy.get('[data-cy=side-panel-button]').click();

                cy.get('#aside-menu-list').contains('Run a workflow').click();

                cy.get('@testWorkflow')
                    .then((testWorkflow) => {
                        cy.get('main').contains(testWorkflow.name).click();
                        cy.get('[data-cy=run-process-next-button]').click();
                        cy.get('[data-cy=run-wf-project-picker-ok-button]').click();

                        cy.get('label').contains('directoryInputName').parent('div').find('input').click();
                        cy.get('div[role=dialog]')
                            .within(() => {
                                // must use .then to avoid selecting instead of expanding https://github.com/cypress-io/cypress/issues/5529
                                cy.get('p').contains('Home Projects').closest('ul')
                                    .find('i')
                                    .then(el => el.click());

                                cy.get(`[data-id=${testCollection.uuid}]`)
                                    .find('i').click();

                                cy.get(`[data-id="${testCollection.uuid}/subdir"]`)
                                    .find('i').click();

                                cy.contains('dir1').closest('[data-action=TOGGLE_ACTIVE]').parent().find('input[type=checkbox]').click();
                                cy.contains('dir2').closest('[data-action=TOGGLE_ACTIVE]').parent().find('input[type=checkbox]').click();

                                cy.get('[data-cy=ok-button]').click();
                            });

                        // Verify subdirectories were selected
                        cy.get('label').contains('directoryInputName').parent('div')
                            .within(() => {
                                cy.contains('dir1');
                                cy.contains('dir2');
                            });

                        // Reopen tree picker and verify subdirectories are preselected
                        cy.get('label').contains('directoryInputName').parent('div').find('input').click();
                        cy.waitForDom().get('div[role=dialog]')
                            .within(() => {
                                cy.contains('dir1').closest('[data-action=TOGGLE_ACTIVE]').parent().find('input[type=checkbox]').should('be.checked');
                                cy.contains('dir2').closest('[data-action=TOGGLE_ACTIVE]').parent().find('input[type=checkbox]').should('be.checked');
                            });
                    });

            });
    })

    it('handles secret inputs', () => {
        cy.createProject({
            owningUser: activeUser,
            projectName: 'myProject1',
            addToFavorites: true
        });

        cy.setupDockerImage("arvados/jobs").as("dockerImg");

        cy.getAll('@myProject1').then(function ([myProject1]) {
                cy.readFile('cypress/fixtures/workflow_with_secret_input.yaml').then(workflow => {
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
                        cy.get('[data-cy=run-wf-project-picker-ok-button]').click();

                        var foo = cy.get('label').contains('foo').parent('div').find('input');
                        foo.type("secret_value_xyz");
                        foo.should('have.attr', 'type').and('equal', 'password');

                        var bar = cy.get('label').contains('bar').parent('div').find('input');
                        bar.type("exposed_value_xyz");
                        bar.should('have.attr', 'type').and('equal', 'text');
                    });
            cy.get('[data-cy=new-process-panel]').contains('Run workflow').click();

            cy.get('[data-cy=process-io-card]').should('contain', 'exposed_value_xyz');
            cy.get('[data-cy=process-io-card]').should('contain', 'Cannot display secret');
            cy.get('[data-cy=process-io-card]').should('not.contain', 'secret_value_xyz');

            cy.url().then((url) => {
                let uuid = url.split('/').pop();
                cy.getResource(activeUser.token, "container_requests", uuid).then((res) => {
                    expect(res.mounts["/var/lib/cwl/cwl.input.json"].content.bar).to.equal('exposed_value_xyz');
                    expect(res.mounts["/var/lib/cwl/cwl.input.json"].content.foo).to.deep.equal({$include: '/secrets/s0'});
                });
            });

        });
    });

    it('handles optional inputs', () => {
        cy.intercept({ method: "POST", url: "**/arvados/v1/container_requests" }, (req, res) => {
            const inputs = req.body.container_request.mounts["/var/lib/cwl/cwl.input.json"].content;
            expect(inputs).to.deep.equal({
                int_input: null,
                empty_string_input: null,
                string_input: "foo"
            });

            //handle expected 422 error
            req.reply({
                statusCode: 200,
                body: { message: 'Expected 422 error' },
            });
        }).as("mockedRequest");

        cy.createProject({
            owningUser: adminUser,
            projectName: 'myProject1',
        });

        cy.setupDockerImage("arvados/jobs").as("dockerImg");

        cy.getAll('@myProject1').then(function ([myProject1]) {
                cy.readFile('cypress/fixtures/workflow-with-optional-inputs.yaml').then(workflow => {
                    cy.createWorkflow(adminUser.token, {
                        name: `TestWorkflow${Math.floor(Math.random() * 999999)}.cwl`,
                        definition: workflow,
                        owner_uuid: myProject1.uuid,
                    })
                        .as('testWorkflow');
                });

                cy.loginAs(adminUser);

                cy.get('main').contains(myProject1.name).click();

                cy.get('[data-cy=side-panel-button]').click();

                cy.get('#aside-menu-list').contains('Run a workflow').click();

                cy.get('@testWorkflow')
                    .then((testWorkflow) => {
                        cy.get('main').contains(testWorkflow.name).click();
                        cy.get('[data-cy=run-process-next-button]').click();

                        var int_input = cy.get('label').contains('int_input').parent('div').find('input');
                        var string_input = cy.get('label').contains('string_input').parent('div').find('input');
                        var empty_string_input = cy.get('label').contains('empty_string_input').parent('div').find('input');

                        string_input.type("foo");

                        //both inputs are optional, so they should be null instead of empty strings
                        int_input.type("123{backspace}{backspace}{backspace}")
                        empty_string_input.type("bar{backspace}{backspace}{backspace}");
                    });

                cy.get('[data-cy=new-process-panel]').contains('Run workflow').click();

                cy.wait('@mockedRequest')
        });
    });
})
