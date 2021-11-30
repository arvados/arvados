// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

describe('Group manage tests', function() {
    let activeUser;
    let adminUser;
    let otherUser;

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
        cy.getUser('user', 'Active', 'User', false, true)
            .as('activeUser').then(function() {
                activeUser = this.activeUser;
            }
        );
        cy.getUser('otheruser', 'Other', 'User', false, true)
            .as('otherUser').then(function() {
                otherUser = this.otherUser;
            }
        );
    });

    beforeEach(function() {
        cy.clearCookies();
        cy.clearLocalStorage();
    });

    it('creates a new group', function() {
        const groupName = `Test group (${Math.floor(999999 * Math.random())})`;
        cy.loginAs(activeUser);

        // Navigate to Groups
        cy.get('[data-cy=side-panel-tree]').contains('Groups').click();

        // Create new group
        cy.get('[data-cy=groups-panel-new-group]').click();
        cy.get('[data-cy=form-dialog]')
            .should('contain', 'Create a group')
            .within(() => {
                cy.get('input[name=name]').type(groupName);
                cy.get('button[type=submit]').click();
            });
        
        // Check that the group was created
        cy.get('[data-cy=groups-panel-data-explorer]').contains(groupName).click();
        cy.get('[data-cy=group-members-data-explorer]').contains('Active User');
    });

    it('adds users to the group', function() {
        // Add other user to the group
        cy.get('[data-cy=group-member-add]').click();
        cy.get('[data-cy=form-dialog]')
            .should('contain', 'Add users')
            .within(() => {
                cy.get('input').type("other");
            });
        cy.contains('Other User').click();
        cy.get('[data-cy=form-dialog] button[type=submit]').click();

        // Check that both users are present with appropriate permissions
        cy.get('[data-cy=group-members-data-explorer]')
            .contains('Other User')
            .parents('tr')
            .within(() => {
                cy.contains('Read');
            });
        cy.get('[data-cy=group-members-data-explorer] tr')
            .contains('Active User')
            .parents('tr')
            .within(() => {
                cy.contains('Manage');
            });
    });

    it('changes permission level of a member', function() {
        // Test change permission level
        cy.get('[data-cy=group-members-data-explorer]')
            .contains('Other User')
            .parents('tr')
            .within(() => {
                cy.contains('Read')
                    .parents('td')
                    .within(() => {
                        cy.get('button').click();
                    });
            });
        cy.get('[data-cy=context-menu]')
            .contains('Write')
            .click();
        cy.get('[data-cy=group-members-data-explorer]')
            .contains('Other User')
            .parents('tr')
            .within(() => {
                cy.contains('Write');
            });
    });

    it('removes users from the group', function() {
        // Remove other user
        cy.get('[data-cy=group-members-data-explorer]')
            .contains('Other User')
            .parents('tr')
            .within(() => {
                cy.get('[data-cy=resource-delete-button]').click();
            });
        cy.get('[data-cy=confirmation-dialog-ok-btn]').click();
        cy.get('[data-cy=group-members-data-explorer]')
            .should('not.contain', 'Other User');

    });
});
