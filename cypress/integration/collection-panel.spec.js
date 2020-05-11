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

    it('shows a collection by URL', function() {
        cy.loginAs(activeUser);
        cy.createCollection(adminUser.token, {
            name: 'Test collection',
            owner_uuid: activeUser.user.uuid,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:bar\n"})
        .as('testCollection').then(function() {
            cy.visit(`/collections/${this.testCollection.uuid}`);
            cy.get('[data-cy=collection-info-panel]')
                .should('contain', this.testCollection.name)
                .and('contain', this.testCollection.uuid);
            cy.get('[data-cy=collection-files-panel]')
                .should('contain', 'bar');
        })
    })

    // it('')
})