// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

describe('Multiselect Toolbar Tests', () => {
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

    beforeEach(function () {
        cy.clearCookies();
        cy.clearLocalStorage();
    });

    it('exists in DOM', () => {
        cy.loginAs(activeUser);
        cy.get('[data-cy=multiselect-toolbar]').should('exist');
        cy.get('[data-cy=multiselect-button]').should('not.exist');
        cy.get('[data-cy=multiselect-alt-button]').should('not.exist');
    });
});
