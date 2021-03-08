// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

describe('Sharing tests', function () {
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

    it('can share projects to other users', () => {
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

        cy.getAll('@mySharedWritableProject', '@mySharedReadonlyProject')
            .then(function ([mySharedWritableProject, mySharedReadonlyProject]) {
                cy.loginAs(activeUser);

                cy.contains('Shared with me').click();

                cy.get('main').contains(mySharedWritableProject.name).rightclick();
                cy.get('[data-cy=context-menu]').should('contain', 'Move to trash');
                cy.get('[data-cy=context-menu]').contains('Move to trash').click();

                cy.get('main').contains(mySharedReadonlyProject.name).rightclick();
                cy.get('[data-cy=context-menu]').should('not.contain', 'Move to trash');
            });
    });
});