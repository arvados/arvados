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
    })

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
                    cy.get('[data-cy=projects-tree-favourites-tree-pciker]').find('i').click();
                    cy.contains(myProject1.name);
                    cy.contains(mySharedWritableProject.name);
                    cy.get('[data-cy=projects-tree-favourites-tree-pciker]')
                        .should('not.contain', mySharedReadonlyProject.name);
                    cy.contains(mySharedWritableProject.name).click();
                    cy.get('[data-cy=form-submit-btn]').click();
                });

                cy.doSearch(`${mySharedWritableProject.uuid}`);
                cy.get('main').contains(testCollection.name);
            });
    });

    it('can copy selected into the collection', () => {
        cy.loginAs(activeUser);

        cy.createCollection(adminUser.token, {
            name: `Test source collection ${Math.floor(Math.random() * 999999)}`,
            owner_uuid: activeUser.user.uuid,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:bar\n"
        })
            .as('testSourceCollection');

        cy.createCollection(adminUser.token, {
            name: `Test target collection ${Math.floor(Math.random() * 999999)}`,
            owner_uuid: activeUser.user.uuid
        })
            .as('testTargetCollection');

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
                cy.get('[data-cy=collection-files-panel]').find('input[type=checkbox]').click();
                cy.get('[data-cy=collection-files-panel-options-btn]').click();
                cy.get('[data-cy=context-menu]')
                    .contains('Copy selected into the collection').click();

                cy.get('[data-cy=projects-tree-favourites-tree-pciker]')
                    .find('i')
                    .click();

                cy.get('[data-cy=projects-tree-favourites-tree-pciker]')
                    .contains(testTargetCollection.name)
                    .click();

                cy.get('[data-cy=form-submit-btn]').click();

                cy.get('.layout-pane-primary')
                    .contains('Projects').click();

                cy.get('main').contains(testTargetCollection.name).click();

                cy.get('[data-cy=collection-files-panel]').contains('bar');
            });
    });
});