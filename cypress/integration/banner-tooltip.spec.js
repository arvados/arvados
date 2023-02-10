// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

describe('Collection panel tests', function () {
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
            }
            );
        cy.getUser('collectionuser1', 'Collection', 'User', false, true)
            .as('activeUser').then(function () {
                activeUser = this.activeUser;
            }
            )

        cy.getAll('@adminUser')
            .then(function([adminUser]) {
                cy.createCollection(adminUser.token, {
                    name: `BannerTooltipTest${Math.floor(Math.random() * 999999)}`,
                    owner_uuid: adminUser.user.uuid,
                }).as('bannerCollection');

                cy.getAll('@bannerCollection')
                    .then(function ([bannerCollection]) {

                        collectionUUID=bannerCollection.uuid;
        
                        cy.loginAs(adminUser);
        
                        cy.goToPath(`/collections/${bannerCollection.uuid}`);
        
                        cy.get('[data-cy=upload-button]').click();
        
                        cy.fixture('files/banner.html').as('banner');
                        cy.fixture('files/tooltips.txt').as('tooltips');
                        
                        cy.getAll('@banner', '@tooltips')
                            .then(([banner, tooltips]) => {
                                console.log(tooltips)
                                cy.get('[data-cy=drag-and-drop]').upload(btoa(banner), 'banner.html');
                                cy.get('[data-cy=drag-and-drop]').upload(btoa(tooltips), 'tooltips.json');
                            });
  
                        cy.get('[data-cy=form-submit-btn]').click();
                    });
            });
            cy.on('uncaught:exception', (err, runnable) => {console.error(err)});
    });

    beforeEach(function () {
        cy.clearCookies();
        cy.clearLocalStorage();
        cy.intercept({ method: 'GET', hostname: 'localhost', url: '**/arvados/v1/config?nocache=*' }, (req) => {
            req.reply((res) => {
                res.body.Workbench.BannerUUID = collectionUUID;
            });
        });
    });

    it('should re-show the banner', () => {
        cy.loginAs(adminUser);

        cy.wait(2000);

        cy.get('[data-cy=confirmation-dialog-ok-btn]').click();

        cy.get('[title=Notifications]').click();
        cy.get('li').contains('Restore Banner').click();

        cy.wait(2000);

        cy.get('[data-cy=confirmation-dialog-ok-btn]').should('be.visible');
    });


    it('should show tooltips and remove tooltips as localStorage key is present', () => {
        cy.loginAs(adminUser);

        cy.wait(2000);

        cy.get('[data-cy=side-panel-tree]').then(($el) => {
            const el = $el.get(0) //native DOM element
            expect(el._tippy).to.exist;
        });

        cy.wait(2000);

        cy.get('[data-cy=confirmation-dialog-ok-btn]').click();

        cy.get('[title=Notifications]').click();
        cy.get('li').contains('Disable tooltips').click();

        cy.get('[data-cy=side-panel-tree]').then(($el) => {
            const el = $el.get(0) //native DOM element
            expect(el._tippy).to.be.undefined;
        });
    });
});
