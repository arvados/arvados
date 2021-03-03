// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

describe('Favorites tests', function () {
    let activeUser;
    let adminUser;

    before(function () {
        // Only set up common users once. These aren't set up as aliases because
        // aliases are cleaned up after every test. Also it doesn't make sense
        // to set the same users on beforeEach() over and over again, so we
        // separate a little from Cypress' 'Best Practices' here.
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
    })

    beforeEach(function () {
        cy.clearCookies()
        cy.clearLocalStorage()
    });

    const createSharedProjects = () => {
        cy.createGroup(adminUser.token, {
            name: `my-shared-writable-project ${Math.floor(Math.random() * 999999)}`,
            group_class: 'project',
        }).as('mySharedWritableProject').then(function (mySharedWritableProject) {
            cy.contains('Refresh').click();
            cy.get('main').contains(mySharedWritableProject.name).rightclick();
            cy.get('[data-cy=context-menu]').within(() => {
                cy.contains('Share').click();
            });
            cy.get('[id="select-permissions"]').as('selectPermissions');
            cy.get('@selectPermissions').click();
            cy.contains('Write').click();
            cy.get('.sharing-dialog').as('sharingDialog');
            cy.get('[data-cy=invite-people-field]').find('input').type(activeUser.user.email);
            cy.get('[role=tooltip]').click();
            cy.get('@sharingDialog').contains('Save').click();
        });

        cy.createGroup(adminUser.token, {
            name: `my-shared-readonly-project ${Math.floor(Math.random() * 999999)}`,
            group_class: 'project',
        }).as('mySharedReadonlyProject').then(function (mySharedReadonlyProject) {
            cy.contains('Refresh').click();
            cy.get('main').contains(mySharedReadonlyProject.name).rightclick();
            cy.get('[data-cy=context-menu]').within(() => {
                cy.contains('Share').click();
            });
            cy.get('.sharing-dialog').as('sharingDialog');
            cy.get('[data-cy=invite-people-field]').find('input').type(activeUser.user.email);
            cy.get('[role=tooltip]').click();
            cy.get('@sharingDialog').contains('Save').click();
        });

        cy.createGroup(activeUser.token, {
            name: `my-project ${Math.floor(Math.random() * 999999)}`,
            group_class: 'project',
        }).as('myProject1');
    };

    it('checks that Public favorites does not appear under shared with me', function () {
        cy.loginAs(adminUser);
        cy.contains('Shared with me').click();
        cy.get('main').contains('Public favorites').should('not.exist');
    });

    it('creates and removes a public favorite', function () {
        cy.loginAs(adminUser);
        cy.createGroup(adminUser.token, {
            name: `my-favorite-project`,
            group_class: 'project',
        }).as('myFavoriteProject').then(function () {
            cy.contains('Refresh').click();
            cy.get('main').contains('my-favorite-project').rightclick();
            cy.contains('Add to public favorites').click();
            cy.contains('Public Favorites').click();
            cy.get('main').contains('my-favorite-project').rightclick();
            cy.contains('Remove from public favorites').click();
            cy.get('main').contains('my-favorite-project').should('not.exist');
            cy.trashGroup(adminUser.token, this.myFavoriteProject.uuid);
        });
    });

    it('can copy collection to favorites', () => {
        cy.loginAs(adminUser);

        createSharedProjects();

        cy.createCollection(adminUser.token, {
            name: `Test collection ${Math.floor(Math.random() * 999999)}`,
            owner_uuid: activeUser.user.uuid,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:bar\n"
        })
            .as('testCollection');

        cy.getAll('@mySharedWritableProject', '@mySharedReadonlyProject', '@myProject1', '@testCollection')
            .then(function ([mySharedWritableProject, mySharedReadonlyProject, myProject1, testCollection]) {
                cy.loginAs(activeUser);

                cy.contains('Shared with me').click();

                cy.get('main').contains(mySharedWritableProject.name).rightclick();
                cy.get('[data-cy=context-menu]').within(() => {
                    cy.contains('Add to favorites').click();
                });

                cy.get('main').contains(mySharedReadonlyProject.name).rightclick();
                cy.get('[data-cy=context-menu]').within(() => {
                    cy.contains('Add to favorites').click();
                });

                cy.doSearch(`${activeUser.user.uuid}`);

                cy.get('main').contains(myProject1.name).rightclick();
                cy.get('[data-cy=context-menu]').within(() => {
                    cy.contains('Add to favorites').click();
                });

                cy.contains(testCollection.name).rightclick();
                cy.get('[data-cy=context-menu]').within(() => {
                    cy.contains('Move to').click();
                });

                cy.get('[data-cy=form-dialog]').within(function () {
                    cy.get('[data-cy=projects-tree-favourites-tree-picker]').find('i').click();
                    cy.contains(myProject1.name);
                    cy.contains(mySharedWritableProject.name);
                    cy.get('[data-cy=projects-tree-favourites-tree-picker]')
                        .should('not.contain', mySharedReadonlyProject.name);
                    cy.contains(mySharedWritableProject.name).click();
                    cy.get('[data-cy=form-submit-btn]').click();
                });

                cy.doSearch(`${mySharedWritableProject.uuid}`);
                cy.get('main').contains(testCollection.name);
            });
    });

    it('can copy selected into the collection', () => {
        cy.loginAs(adminUser);

        cy.createCollection(adminUser.token, {
            name: `Test source collection ${Math.floor(Math.random() * 999999)}`,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:bar\n"
        })
            .as('testSourceCollection').then(function (testSourceCollection) {
                cy.contains('Refresh').click();
                cy.get('main').contains(testSourceCollection.name).rightclick();
                cy.get('[data-cy=context-menu]').within(() => {
                    cy.contains('Share').click();
                });
                cy.get('[id="select-permissions"]').as('selectPermissions');
                cy.get('@selectPermissions').click();
                cy.contains('Write').click();
                cy.get('.sharing-dialog').as('sharingDialog');
                cy.get('[data-cy=invite-people-field]').find('input').type(activeUser.user.email);
                cy.get('[role=tooltip]').click();
                cy.get('@sharingDialog').contains('Save').click();
            });

        cy.createCollection(adminUser.token, {
            name: `Test target collection ${Math.floor(Math.random() * 999999)}`,
        })
            .as('testTargetCollection').then(function (testTargetCollection) {
                cy.contains('Refresh').click();
                cy.get('main').contains(testTargetCollection.name).rightclick();
                cy.get('[data-cy=context-menu]').within(() => {
                    cy.contains('Share').click();
                });
                cy.get('[id="select-permissions"]').as('selectPermissions');
                cy.get('@selectPermissions').click();
                cy.contains('Write').click();
                cy.get('.sharing-dialog').as('sharingDialog');
                cy.get('[data-cy=invite-people-field]').find('input').type(activeUser.user.email);
                cy.get('[role=tooltip]').click();
                cy.get('@sharingDialog').contains('Save').click();
            });

        cy.getAll('@testSourceCollection', '@testTargetCollection')
            .then(function ([testSourceCollection, testTargetCollection]) {
                cy.loginAs(activeUser);

                cy.get('.layout-pane-primary')
                    .contains('Projects').click();

                cy.get('main').contains(testTargetCollection.name).rightclick();
                cy.get('[data-cy=context-menu]').within(() => {
                    cy.contains('Add to favorites').click();
                });

                cy.get('main').contains(testSourceCollection.name).click();
                cy.get('[data-cy=collection-files-panel]').contains('bar');
                cy.get('[data-cy=collection-files-panel]').find('input[type=checkbox]').click({ force: true });
                cy.get('[data-cy=collection-files-panel-options-btn]').click();
                cy.get('[data-cy=context-menu]')
                    .contains('Copy selected into the collection').click();

                cy.get('[data-cy=projects-tree-favourites-tree-picker]')
                    .find('i')
                    .click();

                cy.get('[data-cy=projects-tree-favourites-tree-picker]')
                    .contains(testTargetCollection.name)
                    .click();

                cy.get('[data-cy=form-submit-btn]').click();

                cy.get('.layout-pane-primary')
                    .contains('Projects').click();

                cy.get('main').contains(testTargetCollection.name).click();

                cy.get('[data-cy=collection-files-panel]').contains('bar');
            });
    });

    it.only('can view favourites in workflow', () => {
        cy.loginAs(adminUser);

        createSharedProjects();

        cy.getAll('@mySharedWritableProject', '@mySharedReadonlyProject', '@myProject1')
            .then(function ([mySharedWritableProject, mySharedReadonlyProject, myProject1]) {
                cy.loginAs(activeUser);

                cy.createWorkflow(adminUser.token, {
                    name: `TestWorkflow${Math.floor(Math.random() * 999999)}.cwl`,
                    definition: "{\n    \"$graph\": [\n        {\n            \"class\": \"Workflow\",\n            \"doc\": \"Reverse the lines in a document, then sort those lines.\",\n            \"hints\": [\n                {\n                    \"acrContainerImage\": \"99b0201f4cade456b4c9d343769a3b70+261\",\n                    \"class\": \"http://arvados.org/cwl#WorkflowRunnerResources\"\n                }\n            ],\n            \"id\": \"#main\",\n            \"inputs\": [\n                {\n                    \"default\": null,\n                    \"doc\": \"The input file to be processed.\",\n                    \"id\": \"#main/input\",\n                    \"type\": \"File\"\n                },\n                {\n                    \"default\": true,\n                    \"doc\": \"If true, reverse (decending) sort\",\n                    \"id\": \"#main/reverse_sort\",\n                    \"type\": \"boolean\"\n                }\n            ],\n            \"outputs\": [\n                {\n                    \"doc\": \"The output with the lines reversed and sorted.\",\n                    \"id\": \"#main/output\",\n                    \"outputSource\": \"#main/sorted/output\",\n                    \"type\": \"File\"\n                }\n            ],\n            \"steps\": [\n                {\n                    \"id\": \"#main/rev\",\n                    \"in\": [\n                        {\n                            \"id\": \"#main/rev/input\",\n                            \"source\": \"#main/input\"\n                        }\n                    ],\n                    \"out\": [\n                        \"#main/rev/output\"\n                    ],\n                    \"run\": \"#revtool.cwl\"\n                },\n                {\n                    \"id\": \"#main/sorted\",\n                    \"in\": [\n                        {\n                            \"id\": \"#main/sorted/input\",\n                            \"source\": \"#main/rev/output\"\n                        },\n                        {\n                            \"id\": \"#main/sorted/reverse\",\n                            \"source\": \"#main/reverse_sort\"\n                        }\n                    ],\n                    \"out\": [\n                        \"#main/sorted/output\"\n                    ],\n                    \"run\": \"#sorttool.cwl\"\n                }\n            ]\n        },\n        {\n            \"baseCommand\": \"rev\",\n            \"class\": \"CommandLineTool\",\n            \"doc\": \"Reverse each line using the `rev` command\",\n            \"hints\": [\n                {\n                    \"class\": \"ResourceRequirement\",\n                    \"ramMin\": 8\n                }\n            ],\n            \"id\": \"#revtool.cwl\",\n            \"inputs\": [\n                {\n                    \"id\": \"#revtool.cwl/input\",\n                    \"inputBinding\": {},\n                    \"type\": \"File\"\n                }\n            ],\n            \"outputs\": [\n                {\n                    \"id\": \"#revtool.cwl/output\",\n                    \"outputBinding\": {\n                        \"glob\": \"output.txt\"\n                    },\n                    \"type\": \"File\"\n                }\n            ],\n            \"stdout\": \"output.txt\"\n        },\n        {\n            \"baseCommand\": \"sort\",\n            \"class\": \"CommandLineTool\",\n            \"doc\": \"Sort lines using the `sort` command\",\n            \"hints\": [\n                {\n                    \"class\": \"ResourceRequirement\",\n                    \"ramMin\": 8\n                }\n            ],\n            \"id\": \"#sorttool.cwl\",\n            \"inputs\": [\n                {\n                    \"id\": \"#sorttool.cwl/reverse\",\n                    \"inputBinding\": {\n                        \"position\": 1,\n                        \"prefix\": \"-r\"\n                    },\n                    \"type\": \"boolean\"\n                },\n                {\n                    \"id\": \"#sorttool.cwl/input\",\n                    \"inputBinding\": {\n                        \"position\": 2\n                    },\n                    \"type\": \"File\"\n                }\n            ],\n            \"outputs\": [\n                {\n                    \"id\": \"#sorttool.cwl/output\",\n                    \"outputBinding\": {\n                        \"glob\": \"output.txt\"\n                    },\n                    \"type\": \"File\"\n                }\n            ],\n            \"stdout\": \"output.txt\"\n        }\n    ],\n    \"cwlVersion\": \"v1.0\"\n}",
                    owner_uuid: myProject1.uuid,
                })
                    .as('testWorkflow');

                cy.contains('Shared with me').click();

                cy.get('main').contains(mySharedWritableProject.name).rightclick();
                cy.get('[data-cy=context-menu]').within(() => {
                    cy.contains('Add to favorites').click();
                });

                cy.get('main').contains(mySharedReadonlyProject.name).rightclick();
                cy.get('[data-cy=context-menu]').within(() => {
                    cy.contains('Add to favorites').click();
                });

                cy.doSearch(`${activeUser.user.uuid}`);

                cy.get('main').contains(myProject1.name).click();

                cy.get('[data-cy=side-panel-button]').click();

                cy.get('#aside-menu-list').contains('Run a process').click();

                cy.get('@testWorkflow')
                    .then((testWorkflow) => {
                        cy.get('main').contains(testWorkflow.name).click();
                        cy.get('[data-cy=run-process-next-button]').click();
                        cy.get('[readonly]').click();
                        cy.get('[data-cy=choose-a-file-dialog]').as('chooseFileDialog');
                        cy.get('[data-cy=projects-tree-favourites-tree-picker]').contains('Favorites').closest('ul').find('i').click();
                        cy.get('@chooseFileDialog').find(`[data-id=${mySharedWritableProject.uuid}]`);
                        cy.get('@chooseFileDialog').find(`[data-id=${mySharedReadonlyProject.uuid}]`);
                    });
            });
    });
});