
// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { BANNER_LOCAL_STORAGE_KEY } from '../../src/views-components/baner/banner';

describe('Banner / tooltip tests', function () {
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
            });
        cy.getUser('collectionuser1', 'Collection', 'User', false, true)
            .as('activeUser').then(function () {
                activeUser = this.activeUser;
            });
    });

    beforeEach(function () {
        cy.on('uncaught:exception', (err, runnable, promise) => {
            Cypress.log({ message: `Application Error: ${err}`});
            if (promise) {
                return false;
            }
        });
    });
    
    it('should re-show the banner', () => {
        cy.loginAs(adminUser);

        cy.getAll('@adminUser').then(([adminUser]) => {
            cy.createCollection(adminUser.token, {
                name: `BannerTooltipTest${Math.floor(Math.random() * 999999)}`,
                owner_uuid: adminUser.user.uuid,
            }, true).as('bannerCollection');
        });

        cy.getAll('@bannerCollection').then(function ([bannerCollection]) {
            cy.intercept({ method: 'GET', hostname: "127.0.0.1", url: '**/arvados/v1/config?nocache=*' }, (req) => {
                req.continue((res) => {
                    if (res.body.Workbench) res.body.Workbench.BannerUUID = bannerCollection.uuid;
                });
            });    
            
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
        })
        cy.getAll('@bannerCollection').then((bannerCollection)=>{
            console.log('bannerCollection', bannerCollection[0]);
            window.localStorage.setItem(BANNER_LOCAL_STORAGE_KEY, bannerCollection)});
        
        //manual reload instead of loginAs() to preserve localstorage
        cy.reload();
        cy.waitForDom();

        //check that banner appears on reload
        cy.waitForDom().get('[data-cy=confirmation-dialog]', {timeout: 10000}).should('be.visible');
        cy.get('[data-cy=confirmation-dialog-ok-btn]').click();
        cy.waitForDom().get('[data-cy=confirmation-dialog]', {timeout: 10000}).should('not.exist');

        //check that banner appears on "Restore Banner"
        cy.get('[title=Notifications]').click();
        cy.get('li').contains('Restore Banner').click();

        cy.waitForDom().get('[data-cy=confirmation-dialog-ok-btn]', {timeout: 10000}).should('be.visible');
        cy.waitForDom().get('[data-cy=confirmation-dialog]', {timeout: 10000}).should('be.visible');
        cy.get('[data-cy=confirmation-dialog-ok-btn]').click();
        cy.waitForDom().get('[data-cy=confirmation-dialog]', {timeout: 10000}).should('not.exist');
    });


    it('should show tooltips and remove tooltips as localStorage key is present', () => {
        cy.loginAs(adminUser);
        cy.waitForDom();

        cy.getAll('@adminUser').then(([adminUser]) => {
            cy.createCollection(adminUser.token, {
                name: `BannerTooltipTest${Math.floor(Math.random() * 999999)}`,
                owner_uuid: adminUser.user.uuid,
            }, true).as('bannerCollection');
        });

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
