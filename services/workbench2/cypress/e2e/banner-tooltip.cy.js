
// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

describe('Banner / tooltip tests', function () {
    let activeUser;
    let adminUser;
    let collectionUUID;

    before(function () {
        // Only set up common users once. These aren't set up as aliases because
        // aliases are cleaned up after every test. Also it doesn't make sense
        // to set the same users on beforeEach() over and over again, so we
        // separate a little from Cypress' 'Best Practices' here.
        cy.getUser('admin', 'Admin', 'User', true, true)
            .as('adminUser').then(function () {
                adminUser = this.adminUser;
            });
        cy.getUser('collectionuser1', 'Collection', 'User', false, true)
            .as('activeUser').then(function () {
                activeUser = this.activeUser;
            });

        cy.getAll('@adminUser').then(([adminUser]) => {
            // This collection will not be deleted after each test, we'll
            // clean it up manually.
            cy.createCollection(adminUser.token, {
                name: `BannerTooltipTest${Math.floor(Math.random() * 999999)}`,
                owner_uuid: adminUser.user.uuid,
            }, true).as('bannerCollection');
        });

        cy.getAll('@bannerCollection').then(function ([bannerCollection]) {
            collectionUUID = bannerCollection.uuid;

            cy.loginAs(adminUser);

            cy.goToPath(`/collections/${bannerCollection.uuid}`);

            cy.get('[data-cy=upload-button]').click();

            cy.fixture('files/banner.html').as('banner');
            cy.fixture('files/tooltips.txt').as('tooltips');

            cy.getAll('@banner', '@tooltips').then(([banner, tooltips]) => {
                cy.get('[data-cy=drag-and-drop]').upload(banner, 'banner.html', false);
                cy.get('[data-cy=drag-and-drop]').upload(tooltips, 'tooltips.json', false);
            });

            cy.get('[data-cy=form-submit-btn]').click();
            cy.get('[data-cy=form-submit-btn]').should('not.exist');
            cy.get('[data-cy=collection-files-right-panel]')
                .should('contain', 'banner.html');
            cy.get('[data-cy=collection-files-right-panel]')
                .should('contain', 'tooltips.json');
        });
    });

    beforeEach(function () {
        cy.on('uncaught:exception', (err, runnable, promise) => {
            Cypress.log({ message: `Application Error: ${err}`});
            if (promise) {
                return false;
            }
        });
        cy.intercept({ method: 'GET', url: '**/arvados/v1/config?nocache=*' }, (req) => {
            req.on('response', (res) => {
                res.body.Workbench.BannerUUID = collectionUUID;
            });
        });
    });

    after(function () {
        // Delete banner collection after all test used it.
        cy.deleteResource(adminUser.token, "collections", collectionUUID);
    });

    it('should re-show the banner', () => {
        cy.loginAs(adminUser);
        cy.waitForDom();

        cy.waitForDom().get('[data-cy=confirmation-dialog]', {timeout: 10000}).should('be.visible');
        cy.get('[data-cy=confirmation-dialog-ok-btn]').click();
        cy.waitForDom().get('[data-cy=confirmation-dialog]', {timeout: 10000}).should('not.exist');

        cy.get('[title=Notifications]').click();
        cy.get('li').contains('Restore Banner').click();

        cy.waitForDom().get('[data-cy=confirmation-dialog-ok-btn]', {timeout: 10000}).should('be.visible');
    });


    it('should show tooltips and remove tooltips as localStorage key is present', () => {
        cy.loginAs(adminUser);
        cy.waitForDom();

        cy.waitForDom().get('[data-cy=confirmation-dialog]', {timeout: 10000}).should('be.visible');
        cy.get('[data-cy=confirmation-dialog-ok-btn]').click();
        cy.waitForDom().get('[data-cy=confirmation-dialog]', {timeout: 10000}).should('not.exist');

        cy.contains('This allows you to navigate through the app').should('not.exist'); // This content comes from tooltips.txt
        cy.get('[data-cy=side-panel-tree]').trigger('mouseover');
        cy.get('[data-cy=side-panel-tree]').trigger('mouseenter');
        cy.contains('This allows you to navigate through the app').should('be.visible');

        cy.get('[title=Notifications]').click();
        cy.get('li').contains('Disable tooltips').click();

        cy.contains('This allows you to navigate through the app').should('not.exist');
        cy.get('[data-cy=side-panel-tree]').trigger('mouseover');
        cy.get('[data-cy=side-panel-tree]').trigger('mouseenter');
        cy.contains('This allows you to navigate through the app').should('not.exist');
    });
});
