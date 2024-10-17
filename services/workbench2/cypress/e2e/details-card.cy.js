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

    it('shows the appropriate buttons in the multiselect toolbar', () => {
        const msButtonTooltips = ['View details', 'User account', 'API Details'];

        cy.loginAs(activeUser);

        cy.get('[data-cy=multiselect-button]').should('have.length', msButtonTooltips.length);

        for (let i = 0; i < msButtonTooltips.length; i++) {
            cy.get('[data-cy=multiselect-button]').eq(i).trigger('mouseover');
            cy.get('body').contains(msButtonTooltips[i]).should('exist');
            cy.get('[data-cy=multiselect-button]').eq(i).trigger('mouseout');
        }
    });
});

describe('Project Details Card tests', function () {
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
        cy.waitForDom().get('[data-cy=form-dialog]').should('not.exist');

        cy.get('[data-cy=project-details-card]').should('be.visible');
        cy.get('[data-cy=project-details-card]').contains(projName).should('be.visible');
    });

    it('shows the appropriate buttons in the multiselect toolbar', () => {
        const msButtonTooltips = [
            'View details',
            'Open in new tab',
            'Copy link to clipboard',
            'Open with 3rd party client',
            'API Details',
            'Share',
            'New project',
            'Edit project',
            'Move to',
            'Move to trash',
            'Freeze project',
            'Add to favorites',
        ];

        const projName = `Test project (${Math.floor(999999 * Math.random())})`;
        cy.loginAs(activeUser);

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
        cy.get('[data-cy=form-submit-btn]').should('exist').click();
        cy.waitForDom().get('[data-cy=form-dialog]').should('not.exist');

        for (let i = 0; i < msButtonTooltips.length; i++) {
            cy.get('[data-cy=multiselect-button]').eq(i).should('exist');
            cy.get('[data-cy=multiselect-button]').eq(i).trigger('mouseover');
            cy.waitForDom();
            cy.get('body').within(() => {
                cy.contains(msButtonTooltips[i]).should('exist');
            });
            cy.get('[data-cy=multiselect-button]').eq(i).trigger('mouseout');
        }
    });

    it('should toggle description display', () => {
        const projName = `Test project (${Math.floor(999999 * Math.random())})`;

        //a single line description shouldn't change the height of the card
        const projDescription = 'Science! True daughter of Old Time thou art! Who alterest all things with thy peering eyes.';
        //a multi-line description should change the height of the card
        const multiLineProjDescription = '{enter}Why preyest thou thus upon the poet’s heart,{enter}Vulture, whose wings are dull realities?';

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
        cy.get('[data-cy=no-description').should('be.visible');

        //add description
        cy.get('[data-cy=side-panel-tree]').contains('Home Projects').click();
        cy.get('[data-cy=project-panel] tbody tr').contains(projName).rightclick({ force: true });
        cy.get('[data-cy=context-menu]').contains('Edit').click();
        cy.get('[data-cy=form-dialog]').within(() => {
            cy.get('div[contenteditable=true]').click().type(projDescription);
            cy.get('[data-cy=form-submit-btn]').click();
        });
        cy.waitForDom();
        cy.get('[data-cy=project-panel]').should('be.visible');
        cy.get('[data-cy=project-panel] tbody tr').contains(projName).click({ force: true });
        cy.get('[data-cy=project-details-card]').contains(projName).should('be.visible');

        cy.get('[data-cy=project-details-card]').contains(projDescription).should('not.be.visible');
        cy.get('[data-cy=project-details-card]').invoke('height').should('be.lt', 80);
        cy.get('[data-cy=toggle-description]').click();
        cy.waitForDom();
        cy.get('[data-cy=project-details-card]').contains(projDescription).should('be.visible');
        cy.get('[data-cy=project-details-card]').invoke('height').should('be.gt', 80);

        // modify description to be multi-line
        cy.get('[data-cy=side-panel-tree]').contains('Home Projects').click();
        cy.get('[data-cy=project-panel] tbody tr').contains(projName).rightclick({ force: true });
        cy.get('[data-cy=context-menu]').contains('Edit').click();
        cy.get('[data-cy=form-dialog]').within(() => {
            cy.get('div[contenteditable=true]').click().type(multiLineProjDescription);
            cy.get('[data-cy=form-submit-btn]').click();
        });
        cy.get('[data-cy=project-panel] tbody tr').contains(projName).click({ force: true });
        cy.get('[data-cy=project-details-card]').contains(projName).should('be.visible');

        // card height should change if description is multi-line
        cy.get('[data-cy=project-details-card]').contains(projDescription).should('not.be.visible');
        cy.get('[data-cy=project-details-card]').invoke('height').should('be.lt', 80);
        cy.get('[data-cy=toggle-description]').click();
        cy.waitForDom();
        cy.get('[data-cy=project-details-card]').invoke('height').should('be.gt', 130);
        cy.get('[data-cy=toggle-description]').click();
        cy.waitForDom();
        cy.get('[data-cy=project-details-card]').invoke('height').should('be.lt', 80);
    });

    // The following test is enabled on Electron only, as Chromium and Firefox
    // require permissions to access the clipboard.
    it('should display key/value pairs',  { browser: 'electron' }, () => {
        const projName = `Test project (${Math.floor(999999 * Math.random())})`;
        cy.loginAs(adminUser);

        // Create project wih key/value pairs
        cy.get('[data-cy=side-panel-button]').click();
        cy.get('[data-cy=side-panel-new-project]').click();
        cy.get('[data-cy=form-dialog]')
            .should('contain', 'New Project')
            .within(() => {
                cy.get('[data-cy=name-field]').within(() => {
                    cy.get('input').type(projName);
                });
            });

        cy.get('[data-cy=key-input]').should('be.visible').find('input').click().type('Animal');
        cy.get('[data-cy=value-input]').should('be.visible').find('input').click().type('Dog');
        cy.get('[data-cy=property-add-btn]').should('be.visible').click();

        cy.get('[data-cy=key-input]').should('be.visible').find('input').click().type('Importance');
        cy.get('[data-cy=value-input]').should('be.visible').find('input').click().type('Critical');
        cy.get('[data-cy=property-add-btn]').should('be.visible').click();

        cy.get('[data-cy=key-input]').should('be.visible').find('input').click().type('very long key');
        cy.get('[data-cy=value-input]').should('be.visible').find('input').click().type('very loooooooooooooooooooooooooooooooooooooooooooooooooooooong value');
        cy.get('[data-cy=property-add-btn]').should('be.visible').click();

        cy.get('[data-cy=key-input]').should('be.visible').find('input').click().type('very long key 2');
        cy.get('[data-cy=value-input]').should('be.visible').find('input').click().type('very loooooooooooooooooooooooooooooooooooooooooooooooooooooong value 2');
        cy.get('[data-cy=property-add-btn]').should('be.visible').click();

        cy.get('[data-cy=form-submit-btn]').click();

        //toggle chips
        cy.get('[data-cy=project-details-card]').invoke('height').should('be.lt', 95);
        cy.get('[data-cy=toggle-description]').click();
        cy.waitForDom();
        cy.get('[data-cy=project-details-card]').invoke('height').should('be.gt', 96);
        cy.get('[data-cy=toggle-description').click();
        cy.waitForDom();
        cy.get('[data-cy=project-details-card]').invoke('height').should('be.lt', 95);

        //check for key/value pairs in project details card
        // only run in electron because other browsers require permission for clipboard
        if (Cypress.browser.name === 'electron') {
            cy.get('[data-cy=toggle-description]').click();
            cy.waitForDom();
            cy.get('[data-cy=project-details-card]').contains('Animal').should('be.visible');
            cy.get('[data-cy=project-details-card]').contains('Importance').should('be.visible').click();
            cy.waitForDom();
                cy.window().then((win) => {
                    win.navigator.clipboard.readText().then((text) => {
                        //wait is necessary due to known issue with cypress@13.7.1
                        cy.wait(1000);
                        expect(text).to.match(new RegExp(`Importance: Critical`));
                    });
                });
        }
    });
});
