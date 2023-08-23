// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

describe('Page not found tests', function() {
    let adminUser;

    before(function() {
        cy.getUser('admin', 'Admin', 'User', true, true)
            .as('adminUser').then(function() {
                adminUser = this.adminUser;
            }
        );
    });

    beforeEach(function() {
        cy.clearCookies()
        cy.clearLocalStorage()
    });

    it('shows not found page', function() {
        // when
        cy.loginAs(adminUser);
        cy.goToPath(`/this/is/an/invalid/route`);

        // then
        cy.get('[data-cy=not-found-page]').should('exist');
        cy.get('[data-cy=not-found-content]').should('exist');
    });


    it('shows not found popup', function() {
        // given
        [
            '/projects/zzzzz-j7d0g-nonexistingproj',
            '/projects/zzzzz-tpzed-nonexistinguser',
            '/processes/zzzzz-xvhdp-nonexistingproc',
            '/collections/zzzzz-4zz18-nonexistingcoll'
        ].forEach(function(path) {
            // Using de slower loginAs() method to avoid bumping into dialog
            // dismissal issues that are not related to this test.
            cy.loginAs(adminUser);

            // when
            cy.goToPath(path);

            // then
            cy.get('[data-cy=not-found-page]').should('not.exist');
            cy.get('[data-cy=not-found-content]').should('exist');
        });
    });
})