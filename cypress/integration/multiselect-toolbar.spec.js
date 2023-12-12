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

    it('can manipulate a project resource', () => {
        cy.loginAs(activeUser);
        const projName = `Test project (${Math.floor(999999 * Math.random())})`;
        cy.get('[data-cy=side-panel-button]').click();
        cy.get('[data-cy=side-panel-new-project]').click();
        cy.get('[data-cy=form-dialog]')
            .should('contain', 'New Project')
            .within(() => {
                cy.get('[data-cy=name-field]').within(() => {
                    cy.get('input').type(projName);
                });
            })
        cy.get("[data-cy=form-submit-btn]").click();
        cy.waitForDom()
        cy.go('back')

        cy.get('[data-cy=data-table-row]').contains(projName).should('exist').parent().parent().parent().click()
        cy.get('[data-cy=multiselect-button]').should('have.length', 12).eq(3).trigger('mouseover');
        cy.get('body').contains('Edit project').should('exist')
        cy.get('[data-cy=multiselect-button]').eq(3).click()
        cy.get("[data-cy=form-dialog]").within(() => {
            cy.get("div[contenteditable=true]").click().type('this is a test');
            cy.get("[data-cy=form-submit-btn]").click();
        });
    });
});
