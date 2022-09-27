// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

describe('Group manage tests', function() {
    let activeUser;
    let adminUser;
    let otherUser;
    let userThree;
    const groupName = `Test group (${Math.floor(999999 * Math.random())})`;

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
        cy.getUser('userThree', 'User', 'Three', false, true)
            .as('userThree').then(function() {
                userThree = this.userThree;
            }
        );
    });

    it('creates a new group', function() {
        cy.loginAs(activeUser);

        // Navigate to Groups
        cy.get('[data-cy=side-panel-tree]').contains('Groups').click();

        // Create new group
        cy.get('[data-cy=groups-panel-new-group]').click();
        cy.get('[data-cy=form-dialog]')
            .should('contain', 'New Group')
            .within(() => {
                cy.get('input[name=name]').type(groupName);
                cy.get('[data-cy=users-field] input').type("three");
            });
        cy.get('[role=tooltip]').click();
        cy.get('[data-cy=form-dialog]').within(() => {
            cy.get('[data-cy=form-submit-btn]').click();
        })

        // Check that the group was created
        cy.get('[data-cy=groups-panel-data-explorer]').contains(groupName).click();
        cy.get('[data-cy=group-members-data-explorer]').contains(activeUser.user.full_name);
        cy.get('[data-cy=group-members-data-explorer]').contains(userThree.user.full_name);
    });

    it('adds users to the group', function() {
        // Add other user to the group
        cy.get('[data-cy=group-member-add]').click();
        cy.get('.sharing-dialog')
            .should('contain', 'Sharing settings')
            .within(() => {
                cy.get('[data-cy=invite-people-field] input').type("other");
            });
        cy.get('[role=tooltip]').click();
        cy.get('.sharing-dialog').contains('Save').click();
        cy.get('.sharing-dialog').contains('Close').click();

        // Check that both users are present with appropriate permissions
        cy.get('[data-cy=group-members-data-explorer]')
            .contains(otherUser.user.full_name)
            .parents('tr')
            .within(() => {
                cy.contains('Read');
            });
        cy.get('[data-cy=group-members-data-explorer] tr')
            .contains(activeUser.user.full_name)
            .parents('tr')
            .within(() => {
                cy.contains('Manage');
            });
    });

    it('changes permission level of a member', function() {
        // Test change permission level
        cy.get('[data-cy=group-members-data-explorer]')
            .contains(otherUser.user.full_name)
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
            .contains(otherUser.user.full_name)
            .parents('tr')
            .within(() => {
                cy.contains('Write');
            });
    });

    it('can unhide and re-hide users', function() {
        // Must use admin user to have manage permission on user
        cy.loginAs(adminUser);
        cy.get('[data-cy=side-panel-tree]').contains('Groups').click();
        cy.get('[data-cy=groups-panel-data-explorer]').contains(groupName).click();

        // Check that other user is hidden
        cy.get('[data-cy=group-details-permissions-tab]').click();
        cy.get('[data-cy=group-permissions-data-explorer]')
            .should('not.contain', otherUser.user.full_name)
        cy.get('[data-cy=group-details-members-tab]').click();

        // Test unhide
        cy.get('[data-cy=group-members-data-explorer]')
            .contains(otherUser.user.full_name)
            .parents('tr')
            .within(() => {
                cy.get('[data-cy=user-visible-checkbox]').click();
            });
        // Check that other user is visible
        cy.get('[data-cy=group-details-permissions-tab]').click();
        cy.get('[data-cy=group-permissions-data-explorer]')
            .contains(otherUser.user.full_name)
            .parents('tr')
            .within(() => {
                cy.contains('Read');
            });
        // Test re-hide
        cy.get('[data-cy=group-details-members-tab]').click();
        cy.get('[data-cy=group-members-data-explorer]')
            .contains(otherUser.user.full_name)
            .parents('tr')
            .within(() => {
                cy.get('[data-cy=user-visible-checkbox]').click();
            });
        // Check that other user is hidden
        cy.get('[data-cy=group-details-permissions-tab]').click();
        cy.get('[data-cy=group-permissions-data-explorer]')
            .should('not.contain', otherUser.user.full_name)
    });

    it('displays resources shared with the group', function() {
        // Switch to activeUser
        cy.loginAs(activeUser);
        cy.get('[data-cy=side-panel-tree]').contains('Groups').click();

        // Get groupUuid and create shared project
        cy.get('[data-cy=groups-panel-data-explorer]')
            .contains(groupName)
            .parents('tr')
            .find('[data-cy=uuid]')
            .invoke('text')
            .as('groupUuid')
            .then((groupUuid) => {
                cy.createProject({
                    owningUser: activeUser,
                    projectName: 'test-project',
                }).as('testProject').then((testProject) => {
                    cy.shareWith(activeUser.token, groupUuid, testProject.uuid, 'can_read');
                });
            });

        // Check that the project is listed in permissions
        cy.get('[data-cy=groups-panel-data-explorer]').contains(groupName).click();
        cy.get('[data-cy=group-details-permissions-tab]').click();
        cy.get('[data-cy=group-permissions-data-explorer]')
            .contains('test-project')
            .parents('tr')
            .within(() => {
                cy.contains('Read');
            });
    });

    it('removes users from the group', function() {
        cy.get('[data-cy=side-panel-tree]').contains('Groups').click();
        cy.get('[data-cy=groups-panel-data-explorer]').contains(groupName).click();

        // Remove other user
        cy.get('[data-cy=group-members-data-explorer]')
            .contains(otherUser.user.full_name)
            .parents('tr')
            .within(() => {
                cy.get('[data-cy=resource-delete-button]').click();
            });
        cy.get('[data-cy=confirmation-dialog-ok-btn]').click();
        cy.get('[data-cy=group-members-data-explorer]')
            .should('not.contain', otherUser.user.full_name);

        // Remove user three
        cy.get('[data-cy=group-members-data-explorer]')
            .contains(userThree.user.full_name)
            .parents('tr')
            .within(() => {
                cy.get('[data-cy=resource-delete-button]').click();
            });
        cy.get('[data-cy=confirmation-dialog-ok-btn]').click();
        cy.get('[data-cy=group-members-data-explorer]')
            .should('not.contain', userThree.user.full_name);
    });

    it('renames the group', function() {
        // Navigate to Groups
        cy.get('[data-cy=side-panel-tree]').contains('Groups').click();

        // Open rename dialog
        cy.get('[data-cy=groups-panel-data-explorer]')
            .contains(groupName)
            .rightclick();
        cy.get('[data-cy=context-menu]')
            .contains('Rename')
            .click();

        // Rename the group
        cy.get('[data-cy=form-dialog]')
            .should('contain', 'Edit Group')
            .within(() => {
                cy.get('input[name=name]').clear().type(groupName + ' (renamed)');
                cy.get('button').contains('Save').click();
            });

        // Check that the group was renamed
        cy.get('[data-cy=groups-panel-data-explorer]')
            .contains(groupName + ' (renamed)');
    });

    it('deletes the group', function() {
        // Navigate to Groups
        cy.get('[data-cy=side-panel-tree]').contains('Groups').click();

        // Delete the group
        cy.get('[data-cy=groups-panel-data-explorer]')
            .contains(groupName + ' (renamed)')
            .rightclick();
        cy.get('[data-cy=context-menu]')
            .contains('Remove')
            .click();
        cy.get('[data-cy=confirmation-dialog-ok-btn]').click();

        // Check that the group was deleted
        cy.get('[data-cy=groups-panel-data-explorer]')
            .should('not.contain', groupName + ' (renamed)');
    });

    it.only('disables group-related controls for built-in groups', function() {
        cy.loginAs(adminUser);

        ['All users', 'Anonymous users', 'System group'].forEach((builtInGroup) => {
            cy.get('[data-cy=side-panel-tree]').contains('Groups').click();
            cy.get('[data-cy=groups-panel-data-explorer]').contains(builtInGroup).click();

            // Check group member actions
            // cy.get('[data-cy=group-members-data-explorer]')
            //     .within(() => {
                    cy.get('[data-cy=group-member-add]').should('not.exist');
                    cy.get('[data-cy=user-visible-checkbox] input').should('be.disabled');
                    cy.get('[data-cy=resource-delete-button]').should('be.disabled');
                    // cy.get('[data-cy=edit-permission-button]').should('not.exist');
                });

            // Check permissions actions
            cy.get('[data-cy=group-details-permissions-tab]').click();
            // cy.get('[data-cy=group-permissions-data-explorer]').within(() => {
                cy.get('[data-cy=resource-delete-button]').should('be.disabled');
                cy.get('[data-cy=edit-permission-button]').should('not.exist');
            // });
        });
    });

});
