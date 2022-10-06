// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

describe('Virtual machine login manage tests', function() {
    let activeUser;
    let adminUser;

    const vmHost = `vm-${Math.floor(999999 * Math.random())}.host`;

    before(function() {
        // Only set up common users once. These aren't set up as aliases because
        // aliases are cleaned up after every test. Also it doesn't make sense
        // to set the same users on beforeEach() over and over again, so we
        // separate a little from Cypress' 'Best Practices' here.
        cy.getUser('admin', 'VMAdmin', 'User', true, true)
            .as('adminUser').then(function() {
                adminUser = this.adminUser;
            }
        );
        cy.getUser('user', 'VMActive', 'User', false, true)
            .as('activeUser').then(function() {
                activeUser = this.activeUser;
            }
        );
    });

    it('adds and removes vm logins', function() {
        cy.loginAs(adminUser);
        cy.createVirtualMachine(adminUser.token, {hostname: vmHost});

        // Navigate to VM admin
        cy.get('header button[title="Admin Panel"]').click();
        cy.get('#admin-menu').contains('Virtual Machines').click();

        // Add login permission to admin
        cy.get('[data-cy=vm-admin-table]')
            .contains(vmHost)
            .parents('tr')
            .within(() => {
                cy.get('button[title="Add Login Permission"]').click();
            });
        cy.get('[data-cy=form-dialog]')
            .should('contain', 'Add login permission')
            .within(() => {
                cy.get('label')
                  .contains('Search for user')
                  .parent()
                  .within(() => {
                    cy.get('input').type('VMAdmin');
                  })
            });
        cy.get('[role=tooltip]').click();
        cy.get('[data-cy=form-dialog]')
            .should('contain', 'Add login permission')
            .within(() => {
                cy.get('label')
                  .contains('Add groups')
                  .parent()
                  .within(() => {
                    cy.get('input').type('docker sudo{enter}');
                  })
            });
        cy.get('[data-cy=form-dialog]').within(() => {
            cy.get('[data-cy=form-submit-btn]').click();
        });

        cy.get('[data-cy=vm-admin-table]')
            .contains(vmHost)
            .parents('tr')
            .within(() => {
                cy.get('td').contains('admin');
        });

        // Add login permission to activeUser
        cy.get('[data-cy=vm-admin-table]')
            .contains(vmHost)
            .parents('tr')
            .within(() => {
                cy.get('button[title="Add Login Permission"]').click();
            });
        cy.get('[data-cy=form-dialog]')
            .should('contain', 'Add login permission')
            .within(() => {
                cy.get('label')
                  .contains('Search for user')
                  .parent()
                  .within(() => {
                    cy.get('input').type('VMActive user');
                  })
            });
        cy.get('[role=tooltip]').click();
        cy.get('[data-cy=form-dialog]').within(() => {
            cy.get('[data-cy=form-submit-btn]').click();
        });

        cy.get('[data-cy=vm-admin-table]')
            .contains(vmHost)
            .parents('tr')
            .within(() => {
                cy.get('td').contains('user');
        });

        // Check admin's vm page for login
        cy.get('header button[title="Account Management"]').click();
        cy.get('#account-menu').contains('Virtual Machines').click();

        cy.get('[data-cy=vm-user-table]')
            .contains(vmHost)
            .parents('tr')
            .within(() => {
                cy.get('td').contains('admin');
                cy.get('td').contains('docker');
                cy.get('td').contains('sudo');
                cy.get('td').contains('ssh admin@' + vmHost);
        });

        // Check activeUser's vm page for login
        cy.loginAs(activeUser);
        cy.get('header button[title="Account Management"]').click();
        cy.get('#account-menu').contains('Virtual Machines').click();

        cy.get('[data-cy=vm-user-table]')
            .contains(vmHost)
            .parents('tr')
            .within(() => {
                cy.get('td').contains('user');
                cy.get('td').should('not.contain', 'docker');
                cy.get('td').should('not.contain', 'sudo');
                cy.get('td').contains('ssh user@' + vmHost);
        });

        // Edit login permissions
        cy.loginAs(adminUser);
        cy.get('header button[title="Admin Panel"]').click();
        cy.get('#admin-menu').contains('Virtual Machines').click();

        cy.get('[data-cy=vm-admin-table]')
            .contains('admin'); // Wait for page to finish

        cy.get('[data-cy=vm-admin-table]')
            .contains(vmHost)
            .parents('tr')
            .contains('admin')
            .click();

        cy.get('[data-cy=form-dialog]')
            .should('contain', 'Update login permission')
            .within(() => {
                cy.get('label')
                    .contains('Add groups')
                    .parent()
                    .as('groupInput');
            });

        cy.get('@groupInput').within(() => {
            cy.get('div[role=button]').contains('sudo').parent().find('svg').click();
            cy.get('div[role=button]').contains('docker').parent().find('svg').click();
        });

        cy.get('[data-cy=form-dialog]').within(() => {
            cy.get('[data-cy=form-submit-btn]').click();
        });

        // Wait for page to finish loading
        cy.get('[data-cy=vm-admin-table]')
            .contains(vmHost)
            .parents('tr')
            .within(() => {
                cy.get('div[role=button]')
                    .parent()
                    .first()
                    .contains('admin')
            });

        cy.get('[data-cy=vm-admin-table]')
            .contains(vmHost)
            .parents('tr')
            .contains('user')
            .click();

        cy.get('[data-cy=form-dialog]')
            .should('contain', 'Update login permission')
            .within(() => {
                cy.get('label')
                    .contains('Add groups')
                    .parent()
                    .within(() => {
                        cy.get('input').type('docker{enter}');
                    })
            });

        cy.get('[data-cy=form-dialog]').within(() => {
            cy.get('[data-cy=form-submit-btn]').click();
        });

        // Verify new login permissions
        // Check admin's vm page for login
        cy.get('header button[title="Account Management"]').click();
        cy.get('#account-menu').contains('Virtual Machines').click();

        cy.get('[data-cy=vm-user-table]')
            .contains(vmHost)
            .parents('tr')
            .within(() => {
                cy.get('td').contains('admin');
                cy.get('td').should('not.contain', 'docker');
                cy.get('td').should('not.contain', 'sudo');
                cy.get('td').contains('ssh admin@' + vmHost);
        });

        // Verify new login permissions
        // Check activeUser's vm page for login
        cy.loginAs(activeUser);
        cy.get('header button[title="Account Management"]').click();
        cy.get('#account-menu').contains('Virtual Machines').click();

        cy.get('[data-cy=vm-user-table]')
            .contains(vmHost)
            .parents('tr')
            .within(() => {
                cy.get('td').contains('user');
                cy.get('td').contains('docker');
                cy.get('td').should('not.contain', 'sudo');
                cy.get('td').contains('ssh user@' + vmHost);
        });

        // Remove login permissions
        cy.loginAs(adminUser);
        cy.get('header button[title="Admin Panel"]').click();
        cy.get('#admin-menu').contains('Virtual Machines').click();

        cy.get('[data-cy=vm-admin-table]')
            .contains('user'); // Wait for page to finish

        cy.get('[data-cy=vm-admin-table]')
            .contains(vmHost)
            .parents('tr')
            .as('vmRow')
            .contains('user')
            .parents('[role=button]')
            .find('svg')
            .as('removeButton');
        cy.get('@removeButton').click();
        cy.get('[data-cy=confirmation-dialog-ok-btn]').click();

        cy.get('@vmRow')
            .within(() => {
                cy.get('div[role=button]').should('not.contain', 'user');
                cy.get('div[role=button]').should('have.length', 1)
            });

        cy.get('@vmRow')
            .find('div[role=button]')
            .contains('admin')
            .parents('[role=button]')
            .find('svg')
            .as('removeButton');
        cy.get('@removeButton').click();
        cy.get('[data-cy=confirmation-dialog-ok-btn]').click();

        cy.get('[data-cy=vm-admin-table]')
            .contains(vmHost)
            .parents('tr')
            .within(() => {
                cy.get('div[role=button]').should('not.contain', 'admin');
            });

        // Check admin's vm page for login
        cy.get('header button[title="Account Management"]').click();
        cy.get('#account-menu').contains('Virtual Machines').click();

        cy.get('[data-cy=vm-user-panel]')
            .should('not.contain', vmHost);

        // Check activeUser's vm page for login
        cy.loginAs(activeUser);
        cy.get('header button[title="Account Management"]').click();
        cy.get('#account-menu').contains('Virtual Machines').click();

        cy.get('[data-cy=vm-user-panel]')
            .should('not.contain', vmHost);
    });
});
