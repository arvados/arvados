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
    })

    beforeEach(function() {
        cy.clearCookies()
        cy.clearLocalStorage()
    })

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
                        .and('contain', this.testCollection.uuid);
                    // Check for the read-only icon
                    cy.get('[data-cy=read-only-icon]').should(`${isWritable ? 'not.' : ''}exist`);
                    // Check that both read and write operations are available on
                    // the 'More options' menu.
                    cy.get('[data-cy=collection-panel-options-btn]')
                        .click()
                    cy.get('[data-cy=context-menu]')
                        .should('contain', 'Add to favorites')
                        .and(`${isWritable ? '' : 'not.'}contain`, 'Edit collection')
                        .type('{esc}'); // Collapse the options menu
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
                        .type('{esc}'); // Collapse the options menu
                    // File item 'more options' button
                    cy.get('[data-cy=file-item-options-btn')
                        .click()
                    cy.get('[data-cy=context-menu]')
                        .should('contain', 'Download')
                        .and(`${isWritable ? '' : 'not.'}contain`, 'Remove')
                        .type('{esc}'); // Collapse
                })
            })
        })
    })
})
