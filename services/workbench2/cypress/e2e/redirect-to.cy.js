// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { storeRedirects, handleRedirects } from 'common/redirect-to.ts';

describe('redirect-to', () => {
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

    describe('storeRedirects', () => {
        beforeEach(() => {
            cy.createCollection(adminUser.token, {
                name: `Test collection ${Math.floor(Math.random() * 999999)}`,
                owner_uuid: activeUser.user.uuid,
                manifest_text: './subdir 37b51d194a7513e45b56f6524f2d51f2+3 0:3:foo\n. 37b51d194a7513e45b56f6524f2d51f2+3 0:3:bar\n',
            }).as('testCollection1');
        });

        it('should store redirectTo in the session storage', () => {
            cy.getAll('@testCollection1').then(function ([testCollection1]) {
                cy.loginAs(activeUser);
                cy.goToPath(`/collections/${testCollection1.uuid}`);
                // upload file
                cy.get('[data-cy=upload-button]').click();
                cy.fixture('files/5mb.bin', 'base64').then((content) => {
                    cy.get('[data-cy=drag-and-drop]').upload(content, '5mb_a.bin');
                    cy.get('[data-cy=form-submit-btn]').click();
                    cy.get('[data-cy=collection-files-panel]').contains('5mb_a.bin').should('exist');
                });

                // copy file to clipboard
                cy.get('[data-cy=file-item-options-btn]').eq(1).click();
                cy.contains('Copy link to clipboard').click();

                // verify that the link copied to clipboard contains 'redirectTo'
                cy.window().then(async (win) => {
                    expect((await win.navigator.clipboard.readText()).includes('redirectToPreview')).to.be.true;
                });
            });
        });
    });

    describe('handleRedirects', () => {
        beforeEach(() => {
            cy.createCollection(adminUser.token, {
                name: `Test collection ${Math.floor(Math.random() * 999999)}`,
                owner_uuid: activeUser.user.uuid,
                manifest_text: '',
            }).as('testCollection1');
        });

        it('should redirect to page when it is present in session storage', () => {
            let redirectUrl;
            let redirectPath;
            cy.getAll('@testCollection1').then(function ([testCollection1]) {
                cy.loginAs(activeUser);
                cy.goToPath(`/collections/${testCollection1.uuid}`);
                // upload file
                cy.get('[data-cy=upload-button]').click();
                cy.fixture('files/cat.png', 'base64').then((content) => {
                    cy.get('[data-cy=drag-and-drop]').upload(content, 'cat.png');
                    cy.get('[data-cy=form-submit-btn]').click();
                    cy.waitForDom().get('[data-cy=form-submit-btn]').should('not.exist');
                    // Confirm final collection state.
                    cy.get('[data-cy=collection-files-panel]').contains('cat.png').should('exist');
                    // copy file to clipboard
                    cy.get('[data-cy=file-item-options-btn]').click();
                    cy.contains('Copy link to clipboard').click();

                    // verify that the link copied to clipboard contains 'redirectTo'
                    cy.window().then(async (win) => {
                        redirectUrl = await win.navigator.clipboard.readText();
                        redirectPath = redirectUrl.split('redirectToPreview=')[1];

                        cy.get('[aria-label="Account Management"]').click();
                        cy.contains('Logout').click();

                        cy.visit(redirectUrl);
                        cy.contains('Please log in.', { timeout: 10000 }).should('be.visible');
                        cy.waitForLocalStorage('redirectToPreview').then((value) => {
                            // verify the redirect in the url is put into local storage
                            expect(value).to.equal(decodeURIComponent(redirectPath));
                            // verify that redirect is handled
                            // regex because chromium adds a trailing slash to the url
                            expect(win.location.href.replace(/\//g, "")).to.equal(redirectUrl.replace(/\//g, ""));
                        });
                    });
                });
            });
        });
    });
});