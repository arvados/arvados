// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

describe('Multi-file deletion tests', function () {
    let activeUser;
    let adminUser;

    before(function () {
        cy.getUser('admin', 'Admin', 'User', true, true)
            .as('adminUser').then(function () {
                adminUser = this.adminUser;
            }
            );
        cy.getUser('collectionuser1', 'Collection', 'User', false, true)
            .as('activeUser').then(function () {
                activeUser = this.activeUser;
            }
            );
    });

    beforeEach(function () {
        cy.clearCookies();
        cy.clearLocalStorage();
    });

    it('deletes all files from root dir', function () {
        cy.createCollection(adminUser.token, {
            name: `Test collection ${Math.floor(Math.random() * 999999)}`,
            owner_uuid: activeUser.user.uuid,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:bar\n. 37b51d194a7513e45b56f6524f2d51f2+3 0:3:baz\n"
        })
            .as('testCollection').then(function () {
                cy.loginAs(activeUser);
                cy.goToPath(`/collections/${this.testCollection.uuid}`);

                cy.get('[data-cy=collection-files-panel]').within(() => {
                    cy.get('[type="checkbox"]').first().check();
                    cy.get('[type="checkbox"]').last().check();
                });
                cy.get('[data-cy=collection-files-panel-options-btn]').click();
                cy.get('[data-cy=context-menu] div').contains('Remove selected').click();
                cy.get('[data-cy=confirmation-dialog-ok-btn]').click();
                cy.wait(1000);
                cy.get('[data-cy=collection-files-panel]')
                    .should('not.contain', 'baz')
                    .and('not.contain', 'bar');
            });
    });

    it.skip('deletes all files from non root dir', function () {
        cy.createCollection(adminUser.token, {
            name: `Test collection ${Math.floor(Math.random() * 999999)}`,
            owner_uuid: activeUser.user.uuid,
            manifest_text: "./subdir 37b51d194a7513e45b56f6524f2d51f2+3 0:3:foo\n. 37b51d194a7513e45b56f6524f2d51f2+3 0:3:baz\n"
        })
            .as('testCollection').then(function () {
                cy.loginAs(activeUser);
                cy.goToPath(`/collections/${this.testCollection.uuid}`);

                cy.get('[data-cy=virtual-file-tree] > div > i').first().click();
                cy.get('[data-cy=collection-files-panel]')
                    .should('contain', 'foo');

                cy.get('[data-cy=collection-files-panel]')
                    .contains('foo').closest('[data-cy=virtual-file-tree]').find('[type="checkbox"]').click();

                cy.get('[data-cy=collection-files-panel-options-btn]').click();
                cy.get('[data-cy=context-menu] div').contains('Remove selected').click();
                cy.get('[data-cy=confirmation-dialog-ok-btn]').click();

                cy.get('[data-cy=collection-files-panel]')
                    .should('not.contain', 'subdir')
                    .and('contain', 'baz');
            });
    });

    it('deletes all files from non root dir', function () {
        cy.createCollection(adminUser.token, {
            name: `Test collection ${Math.floor(Math.random() * 999999)}`,
            owner_uuid: activeUser.user.uuid,
            manifest_text: "./subdir 37b51d194a7513e45b56f6524f2d51f2+3 0:3:foo\n. 37b51d194a7513e45b56f6524f2d51f2+3 0:3:baz\n"
        })
            .as('testCollection').then(function () {
                cy.loginAs(activeUser);
                cy.goToPath(`/collections/${this.testCollection.uuid}`);

                cy.get('[data-cy=collection-files-panel]').contains('subdir').click();
                cy.wait(1000);
                cy.get('[data-cy=collection-files-panel]')
                    .should('contain', 'foo');

                cy.get('[data-cy=collection-files-panel]')
                    .contains('foo').parent().find('[type="checkbox"]').click();

                cy.get('[data-cy=collection-files-panel-options-btn]').click();
                cy.get('[data-cy=context-menu] div').contains('Remove selected').click();
                cy.get('[data-cy=confirmation-dialog-ok-btn]').click();

                cy.get('[data-cy=collection-files-panel]')
                    .should('not.contain', 'foo')
                    .and('contain', 'subdir');
            });
    });
})
