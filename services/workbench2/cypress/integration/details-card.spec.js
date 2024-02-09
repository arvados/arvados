// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

describe('Details Card tests', function () {
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

    it('should display the user details card', () => {
        cy.loginAs(adminUser);

        cy.get('[data-cy=user-details-card]').should('be.visible');
        cy.get('[data-cy=user-details-card]').contains(adminUser.user.full_name).should('be.visible');
    });

    it('should contain a context menu with the correct options', () => {
        cy.loginAs(adminUser);

        cy.get('[data-cy=kebab-icon]').should('be.visible').click();

        //admin options
        cy.get('[data-cy=context-menu]').should('be.visible');
        cy.get('[data-cy=context-menu]').contains('API Details').should('be.visible');
        cy.get('[data-cy=context-menu]').contains('Account Settings').should('be.visible');
        cy.get('[data-cy=context-menu]').contains('Attributes').should('be.visible');
        cy.get('[data-cy=context-menu]').contains('Project').should('be.visible');
        cy.get('[data-cy=context-menu]').contains('Deactivate User').should('be.visible');

        cy.loginAs(activeUser);

        cy.get('[data-cy=kebab-icon]').should('be.visible').click();

        //active user options
        cy.get('[data-cy=context-menu]').should('be.visible');
        cy.get('[data-cy=context-menu]').contains('API Details').should('be.visible');
        cy.get('[data-cy=context-menu]').contains('Account Settings').should('be.visible');
        cy.get('[data-cy=context-menu]').contains('Attributes').should('be.visible');
        cy.get('[data-cy=context-menu]').contains('Project').should('be.visible');
    });
});
