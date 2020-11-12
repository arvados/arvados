// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

describe('Collection panel tests', function() {
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
    });

    beforeEach(function() {
        cy.clearCookies();
        cy.clearLocalStorage();
    });

    it('shows collection by URL', function() {
        cy.loginAs(activeUser);
        [true, false].map(function(isWritable) {
            cy.createGroup(adminUser.token, {
                name: 'Shared project',
                group_class: 'project',
            }).as('sharedGroup').then(function() {
                // Creates the collection using the admin token so we can set up
                // a bogus manifest text without block signatures.
                cy.createCollection(adminUser.token, {
                    name: 'Test collection',
                    owner_uuid: this.sharedGroup.uuid,
                    properties: {someKey: 'someValue'},
                    manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:bar\n"})
                .as('testCollection').then(function() {
                    // Share the group with active user.
                    cy.createLink(adminUser.token, {
                        name: isWritable ? 'can_write' : 'can_read',
                        link_class: 'permission',
                        head_uuid: this.sharedGroup.uuid,
                        tail_uuid: activeUser.user.uuid
                    })
                    cy.visit(`/collections/${this.testCollection.uuid}`);
                    // Check that name & uuid are correct.
                    cy.get('[data-cy=collection-info-panel]')
                        .should('contain', this.testCollection.name)
                        .and('contain', this.testCollection.uuid)
                        .and('not.contain', 'This is an old version');
                    // Check for the read-only icon
                    cy.get('[data-cy=read-only-icon]').should(`${isWritable ? 'not.' : ''}exist`);
                    // Check that both read and write operations are available on
                    // the 'More options' menu.
                    cy.get('[data-cy=collection-panel-options-btn]')
                        .click()
                    cy.get('[data-cy=context-menu]')
                        .should('contain', 'Add to favorites')
                        .and(`${isWritable ? '' : 'not.'}contain`, 'Edit collection');
                    cy.get('body').click(); // Collapse the menu avoiding details panel expansion
                    cy.get('[data-cy=collection-properties-panel]')
                        .should('contain', 'someKey')
                        .and('contain', 'someValue')
                        .and('not.contain', 'anotherKey')
                        .and('not.contain', 'anotherValue')
                    if (isWritable === true) {
                        // Check that properties can be added.
                        cy.get('[data-cy=collection-properties-form]').within(() => {
                            cy.get('[data-cy=property-field-key]').within(() => {
                                cy.get('input').type('anotherKey');
                            });
                            cy.get('[data-cy=property-field-value]').within(() => {
                                cy.get('input').type('anotherValue');
                            });
                            cy.root().submit();
                        })
                        cy.get('[data-cy=collection-properties-panel]')
                            .should('contain', 'anotherKey')
                            .and('contain', 'anotherValue')
                    } else {
                        // Properties form shouldn't be displayed.
                        cy.get('[data-cy=collection-properties-form]').should('not.exist');
                    }
                    // Check that the file listing show both read & write operations
                    cy.get('[data-cy=collection-files-panel]').within(() => {
                        cy.root().should('contain', 'bar');
                        cy.get('[data-cy=upload-button]')
                            .should(`${isWritable ? '' : 'not.'}contain`, 'Upload data');
                    });
                    cy.get('[data-cy=collection-files-panel]')
                        .contains('bar').rightclick();
                    cy.get('[data-cy=context-menu]')
                        .should('contain', 'Download')
                        .and('contain', 'Open in new tab')
                        .and('contain', 'Copy to clipboard')
                        .and(`${isWritable ? '' : 'not.'}contain`, 'Rename')
                        .and(`${isWritable ? '' : 'not.'}contain`, 'Remove');
                    cy.get('body').click(); // Collapse the menu
                    // Hamburger 'more options' menu button
                    cy.get('[data-cy=collection-files-panel-options-btn]')
                        .click()
                    cy.get('[data-cy=context-menu]')
                        .should('contain', 'Select all')
                        .click()
                    cy.get('[data-cy=collection-files-panel-options-btn]')
                        .click()
                    cy.get('[data-cy=context-menu]')
                        // .should('contain', 'Download selected')
                        .should(`${isWritable ? '' : 'not.'}contain`, 'Remove selected')
                    cy.get('body').click(); // Collapse the menu
                    // File item 'more options' button
                    cy.get('[data-cy=file-item-options-btn')
                        .click()
                    cy.get('[data-cy=context-menu]')
                        .should('contain', 'Download')
                        .and(`${isWritable ? '' : 'not.'}contain`, 'Remove');
                    cy.get('body').click(); // Collapse the menu
                })
            })
        })
    })

    it('renames a file using valid names', function() {
        // Creates the collection using the admin token so we can set up
        // a bogus manifest text without block signatures.
        cy.createCollection(adminUser.token, {
            name: `Test collection ${Math.floor(Math.random() * 999999)}`,
            owner_uuid: activeUser.user.uuid,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:bar\n"})
        .as('testCollection').then(function() {
            cy.loginAs(activeUser);
            cy.visit(`/collections/${this.testCollection.uuid}`);
            const nameTransitions = [
                ['bar', '&'],
                ['&', 'foo'],
                ['foo', '&amp;'],
                ['&amp;', 'I ❤️ ⛵️'],
                ['I ❤️ ⛵️', ' baz'],
                [' baz', ' '] // Must be the last case as cy.contains(' ') doesn't work
            ];
            nameTransitions.forEach(([from, to]) => {
                cy.get('[data-cy=collection-files-panel]')
                    .contains(`${from}`).rightclick();
                cy.get('[data-cy=context-menu]')
                    .contains('Rename')
                    .click();
                cy.get('[data-cy=form-dialog]')
                    .should('contain', 'Rename')
                    .within(() => {
                        cy.get('input').type(`{selectall}{backspace}${to}`);
                    });
                cy.get('[data-cy=form-submit-btn]').click();
                cy.get('[data-cy=collection-files-panel]')
                    .should('not.contain', `${from}`)
                    .and('contain', `${to}`);
            })
        });
    });

    it('renames a file to a different directory', function() {
        // Creates the collection using the admin token so we can set up
        // a bogus manifest text without block signatures.
        cy.createCollection(adminUser.token, {
            name: `Test collection ${Math.floor(Math.random() * 999999)}`,
            owner_uuid: activeUser.user.uuid,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:bar\n"})
        .as('testCollection').then(function() {
            cy.loginAs(activeUser);
            cy.visit(`/collections/${this.testCollection.uuid}`);
            // Rename 'bar' to 'subdir/foo'
            cy.get('[data-cy=collection-files-panel]')
                .contains('bar').rightclick();
            cy.get('[data-cy=context-menu]')
                .contains('Rename')
                .click();
            cy.get('[data-cy=form-dialog]')
                .should('contain', 'Rename')
                .within(() => {
                    cy.get('input').type(`{selectall}{backspace}subdir/foo`);
                });
            cy.get('[data-cy=form-submit-btn]').click();
            cy.get('[data-cy=collection-files-panel]')
                .should('not.contain', 'bar')
                .and('contain', 'subdir');
            // Look for the "arrow icon" and expand the "subdir" directory.
            cy.get('[data-cy=virtual-file-tree] > div > i').click();
            // Rename 'subdir/foo' to 'baz'
            cy.get('[data-cy=collection-files-panel]')
                .contains('foo').rightclick();
            cy.get('[data-cy=context-menu]')
                .contains('Rename')
                .click();
            cy.get('[data-cy=form-dialog]')
                .should('contain', 'Rename')
                .within(() => {
                    cy.get('input')
                        .should('have.value', 'subdir/foo')
                        .type(`{selectall}{backspace}baz`);
                });
            cy.get('[data-cy=form-submit-btn]').click();
            cy.get('[data-cy=collection-files-panel]')
                .should('contain', 'subdir') // empty dir kept
                .and('contain', 'baz');
        });
    });

    it('tries to rename a file with an illegal names', function() {
        // Creates the collection using the admin token so we can set up
        // a bogus manifest text without block signatures.
        cy.createCollection(adminUser.token, {
            name: `Test collection ${Math.floor(Math.random() * 999999)}`,
            owner_uuid: activeUser.user.uuid,
            manifest_text: ". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:bar\n"})
        .as('testCollection').then(function() {
            cy.loginAs(activeUser);
            cy.visit(`/collections/${this.testCollection.uuid}`);
            const illegalNamesFromBackend = ['.', '..'];
            illegalNamesFromBackend.forEach((name) => {
                cy.get('[data-cy=collection-files-panel]')
                    .contains('bar').rightclick();
                cy.get('[data-cy=context-menu]')
                    .contains('Rename')
                    .click();
                cy.get('[data-cy=form-dialog]')
                    .should('contain', 'Rename')
                    .within(() => {
                        cy.get('input').type(`{selectall}{backspace}${name}`);
                    });
                cy.get('[data-cy=form-submit-btn]').click();
                cy.get('[data-cy=form-dialog]')
                    .should('contain', 'Rename')
                    .within(() => {
                        cy.contains('Could not rename');
                    });
                cy.get('[data-cy=form-cancel-btn]').click();
            });
            const illegalNamesFromUI = [
                ['', 'This field is required'],
                [' ', 'Leading/trailing whitespaces not allowed'],
                [' foo', 'Leading/trailing whitespaces not allowed'],
                ['foo ', 'Leading/trailing whitespaces not allowed'],
                ['//foo', 'Empty dir name not allowed']
            ]
            illegalNamesFromUI.forEach(([name, errMsg]) => {
                cy.get('[data-cy=collection-files-panel]')
                    .contains('bar').rightclick();
                cy.get('[data-cy=context-menu]')
                    .contains('Rename')
                    .click();
                cy.get('[data-cy=form-dialog]')
                    .should('contain', 'Rename')
                    .within(() => {
                        cy.get('input').type(`{selectall}{backspace}${name}`);
                    });
                cy.get('[data-cy=form-cancel-btn]').focus();
                cy.get('[data-cy=form-dialog]')
                    .should('contain', 'Rename')
                    .within(() => {
                        cy.contains(`${errMsg}`);
                    });
                cy.get('[data-cy=form-cancel-btn]').click();
            })
        });
    });

    it('can correctly display old versions', function() {
        const colName = `Versioned Collection ${Math.floor(Math.random() * 999999)}`;
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
            // Check the old version displays as what it is.
            cy.loginAs(activeUser)
            cy.visit(`/collections/${oldVersionUuid}`);
            cy.get('[data-cy=collection-info-panel]').should('contain', 'This is an old version');
            cy.get('[data-cy=read-only-icon]').should('exist');
            cy.get('[data-cy=collection-info-panel]').should('contain', colName);
            cy.get('[data-cy=collection-files-panel]').should('contain', 'bar');
        });
    });
})
