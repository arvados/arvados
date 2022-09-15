// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

describe('Search tests', function() {
    let activeUser;
    let adminUser;

    before(function() {
        // Only set up common users once. These aren't set up as aliases because
        // aliases are cleaned up after every test. Also it doesn't make sense
        // to set the same users on beforeEach() over and over again, so we
        // separate a little from Cypress' 'Best Practices' here.
        cy.getUser('admin', 'Admin', 'User', true, true)
            .as('adminUser').then(function() {
                adminUser = this.adminUser;
            }
        );
        cy.getUser('collectionuser1', 'Collection', 'User', false, true)
            .as('activeUser').then(function() {
                activeUser = this.activeUser;
            }
        );
    })

    beforeEach(function() {
        cy.clearCookies()
        cy.clearLocalStorage()
    })

    it('can search for old collection versions', function() {
        const colName = `Versioned Collection ${Math.floor(Math.random() * Math.floor(999999))}`;
        let colUuid = '';
        let oldVersionUuid = '';
        // Make sure no other collections with this name exist
        cy.doRequest('GET', '/arvados/v1/collections', null, {
            filters: `[["name", "=", "${colName}"]]`,
            include_old_versions: true
        })
        .its('body.items').as('collections')
        .then(function() {
            expect(this.collections).to.be.empty;
        });
        // Creates the collection using the admin token so we can set up
        // a bogus manifest text without block signatures.
        cy.createCollection(adminUser.token, {
            name: colName,
            owner_uuid: activeUser.user.uuid,
            preserve_version: true,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:bar\n"})
        .as('originalVersion').then(function() {
            // Change the file name to create a new version.
            cy.updateCollection(adminUser.token, this.originalVersion.uuid, {
                manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:foo\n"
            })
            colUuid = this.originalVersion.uuid;
        });
        // Confirm that there are 2 versions of the collection
        cy.doRequest('GET', '/arvados/v1/collections', null, {
            filters: `[["name", "=", "${colName}"]]`,
            include_old_versions: true
        })
        .its('body.items').as('collections')
        .then(function() {
            expect(this.collections).to.have.lengthOf(2);
            this.collections.map(function(aCollection) {
                expect(aCollection.current_version_uuid).to.equal(colUuid);
                if (aCollection.uuid !== aCollection.current_version_uuid) {
                    oldVersionUuid = aCollection.uuid;
                }
            });
            cy.loginAs(activeUser);
            const searchQuery = `${colName} type:arvados#collection`;
            // Search for only collection's current version
            cy.doSearch(`${searchQuery}`);
            cy.get('[data-cy=search-results]').should('contain', 'head version');
            cy.get('[data-cy=search-results]').should('not.contain', 'version 1');
            // ...and then, include old versions.
            cy.doSearch(`${searchQuery} is:pastVersion`);
            cy.get('[data-cy=search-results]').should('contain', 'head version');
            cy.get('[data-cy=search-results]').should('contain', 'version 1');
        });
    });

    it('can display path of the selected item', function() {
        const colName = `Collection ${Math.floor(Math.random() * Math.floor(999999))}`;

        // Creates the collection using the admin token so we can set up
        // a bogus manifest text without block signatures.
        cy.createCollection(adminUser.token, {
            name: colName,
            owner_uuid: activeUser.user.uuid,
            preserve_version: true,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:bar\n"
        }).then(function() {
            cy.loginAs(activeUser);

            cy.doSearch(colName);

            cy.get('[data-cy=search-results]').should('contain', colName);

            cy.get('[data-cy=search-results]').contains(colName).closest('tr').click();

            cy.get('[data-cy=element-path]').should('contain', `/ Projects / ${colName}`);
        });
    });

    it('can search items using quotes', function() {
        const random = Math.floor(Math.random() * Math.floor(999999));
        const colName = `Collection ${random}`;
        const colName2 = `Collection test ${random}`;

        // Creates the collection using the admin token so we can set up
        // a bogus manifest text without block signatures.
        cy.createCollection(adminUser.token, {
            name: colName,
            owner_uuid: activeUser.user.uuid,
            preserve_version: true,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:bar\n"
        }).as('collection1');

        cy.createCollection(adminUser.token, {
            name: colName2,
            owner_uuid: activeUser.user.uuid,
            preserve_version: true,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:bar\n"
        }).as('collection2');

        cy.getAll('@collection1', '@collection2')
            .then(function() {
                cy.loginAs(activeUser);

                cy.doSearch(colName);
                cy.get('[data-cy=search-results] table tbody tr').should('have.length', 2);

                cy.doSearch(`"${colName}"`);
                cy.get('[data-cy=search-results] table tbody tr').should('have.length', 1);
            });
    });

    it('can display owner of the item', function() {
        const colName = `Collection ${Math.floor(Math.random() * Math.floor(999999))}`;

        cy.createCollection(adminUser.token, {
            name: colName,
            owner_uuid: activeUser.user.uuid,
            preserve_version: true,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:bar\n"
        }).then(function() {
            cy.loginAs(activeUser);

            cy.doSearch(colName);

            cy.get('[data-cy=search-results]').should('contain', colName);

            cy.get('[data-cy=search-results]').contains(colName).closest('tr')
                .within(() => {
                    cy.get('p').contains(activeUser.user.uuid).should('contain', activeUser.user.full_name);
                });
        });
    });
});