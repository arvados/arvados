// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

describe('User Details Card tests', function () {
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

describe.only('Project Details Card tests', function () {
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

    it('should display the project details card', () => {
        const projName = `Test project (${Math.floor(999999 * Math.random())})`;
        cy.loginAs(adminUser);

        // Create project
        cy.get('[data-cy=side-panel-button]').click();
        cy.get('[data-cy=side-panel-new-project]').click();
        cy.get('[data-cy=form-dialog]')
            .should('contain', 'New Project')
            .within(() => {
                cy.get('[data-cy=name-field]').within(() => {
                    cy.get('input').type(projName);
                });
            });
        cy.get('[data-cy=form-submit-btn]').click();
        cy.get('[data-cy=form-dialog]').should('not.exist');

        cy.get('[data-cy=project-details-card]').should('be.visible');
        cy.get('[data-cy=project-details-card]').contains(projName).should('be.visible');
    });

    it ('should contain a context menu with the correct options', () => {
        cy.get('[data-cy=kebab-icon]').should('be.visible').click();

        //admin options
        cy.get('[data-cy=context-menu]').should('be.visible');
        cy.get('[data-cy=context-menu]').contains('API Details').should('be.visible');
        cy.get('[data-cy=context-menu]').contains('Copy to clipboard').should('be.visible');
        cy.get('[data-cy=context-menu]').contains('Edit project').should('be.visible');
        cy.get('[data-cy=context-menu]').contains('Move to').should('be.visible');
        cy.get('[data-cy=context-menu]').contains('New project').should('be.visible');
        cy.get('[data-cy=context-menu]').contains('Open in new tab').should('be.visible');
        cy.get('[data-cy=context-menu]').contains('Open with 3rd party client').should('be.visible');
        cy.get('[data-cy=context-menu]').contains('Share').should('be.visible');
        cy.get('[data-cy=context-menu]').contains('Add to favorites').should('be.visible');
        cy.get('[data-cy=context-menu]').contains('Freeze project').should('be.visible');
        cy.get('[data-cy=context-menu]').contains('Add to public favorites').should('be.visible');
        cy.get('[data-cy=context-menu]').contains('Move to trash').should('be.visible');
        cy.get('[data-cy=context-menu]').contains('View details').should('be.visible');

        //create project
        const projName = `Test project (${Math.floor(999999 * Math.random())})`;
        cy.loginAs(activeUser);

        cy.get('[data-cy=side-panel-button]').click();
        cy.get('[data-cy=side-panel-new-project]').click();
        cy.get('[data-cy=form-dialog]')
            .should('contain', 'New Project')
            .within(() => {
                cy.get('[data-cy=name-field]').within(() => {
                    cy.get('input').type(projName);
                });
            });
        cy.get('[data-cy=form-submit-btn]').click();

        cy.waitForDom()
        cy.get('[data-cy=kebab-icon]').should('be.visible').click({ force: true });

        //active user options
        cy.get('[data-cy=context-menu]').should('be.visible');
        cy.get('[data-cy=context-menu]').contains('API Details').should('be.visible');
        cy.get('[data-cy=context-menu]').contains('Copy to clipboard').should('be.visible');
        cy.get('[data-cy=context-menu]').contains('Edit project').should('be.visible');
        cy.get('[data-cy=context-menu]').contains('Move to').should('be.visible');
        cy.get('[data-cy=context-menu]').contains('New project').should('be.visible');
        cy.get('[data-cy=context-menu]').contains('Open in new tab').should('be.visible');
        cy.get('[data-cy=context-menu]').contains('Open with 3rd party client').should('be.visible');
        cy.get('[data-cy=context-menu]').contains('Share').should('be.visible');
        cy.get('[data-cy=context-menu]').contains('Add to favorites').should('be.visible');
        cy.get('[data-cy=context-menu]').contains('Freeze project').should('be.visible');
        cy.get('[data-cy=context-menu]').contains('Move to trash').should('be.visible');
        cy.get('[data-cy=context-menu]').contains('View details').should('be.visible');

    });

    it.only('should toggle description display', () => {

      const projName = `Test project (${Math.floor(999999 * Math.random())})`;
      const projDescription = 'Lorem ipsum dolor sit amet, consectetur adipiscing elit, whose wings are dull realities.';
      cy.loginAs(adminUser);

      // Create project
      cy.get('[data-cy=side-panel-button]').click();
      cy.get('[data-cy=side-panel-new-project]').click();
      cy.get('[data-cy=form-dialog]')
          .should('contain', 'New Project')
          .within(() => {
              cy.get('[data-cy=name-field]').within(() => {
                  cy.get('input').type(projName);
              });
          });
      cy.get('[data-cy=form-submit-btn]').click();

      //check for no description
      cy.get("[data-cy=no-description").should('be.visible');

      //add description
      cy.get("[data-cy=side-panel-tree]").contains("Home Projects").click();
      cy.get("[data-cy=project-panel] tbody tr").contains(projName).rightclick({ force: true });
      cy.get("[data-cy=context-menu]").contains("Edit").click();
      cy.get("[data-cy=form-dialog]").within(() => {
          cy.get("div[contenteditable=true]").click().type(projDescription);
          cy.get("[data-cy=form-submit-btn]").click();
      });
      cy.get("[data-cy=project-panel] tbody tr").contains(projName).click({ force: true });
      cy.get('[data-cy=project-details-card]').contains(projName).should('be.visible');

      //toggle description
      cy.get("[data-cy=toggle-description").click();
      cy.waitForDom();
      cy.get("[data-cy=project-description]").should('be.visible');
      cy.get("[data-cy=project-details-card]").contains(projDescription).should('be.visible');
      cy.get("[data-cy=toggle-description").click();
      cy.waitForDom();
      cy.get("[data-cy=project-description]").should('be.hidden');
    });
});