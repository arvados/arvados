// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

describe('Collection panel tests', function() {
    let activeUser;
    let adminUser;

    before(function() {
        // Only set up common users once. These aren't set up as aliases because
        // aliases are cleaned up after every test. Also it doesn't make sense
        // to set the same users on beforeEach() over and over again, so we
        // separate a little from Cypress' 'Best Practices' here.
        cy.getUser('admin', 'Admin', 'User', true, true)
            .as('adminUser').then(function() {
                adminUser = this.adminUser;
            }
        );
        cy.getUser('collectionuser1', 'Collection', 'User', false, true)
            .as('activeUser').then(function() {
                activeUser = this.activeUser;
            }
        );
    })

    beforeEach(function() {
        cy.clearCookies()
        cy.clearLocalStorage()
    })

    it('creates and removes a public favorite', function() {
        cy.loginAs(adminUser);
            cy.createGroup(adminUser.token, {
                name: `my-favorite-project`,
                group_class: 'project',
            }).as('myFavoriteProject').then(function() {
                cy.contains('Refresh').click();
                cy.get('main').contains('my-favorite-project').rightclick();
                cy.contains('Add to public favorites').click();
                cy.contains('Public Favorites').click();
                cy.get('main').contains('my-favorite-project').rightclick();
                cy.contains('Remove from public favorites').click();
                cy.get('main').contains('my-favorite-project').should('not.exist');
                cy.trashGroup(adminUser.token, this.myFavoriteProject.uuid);
            });
    })
})
