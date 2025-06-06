// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

describe('User profile tests', function() {
    let activeUser;
    let testProjectName = `mainProject ${Math.floor(Math.random() * 999999)}`;

    before(function() {
        cy.getUser('user', 'Active', 'User', false, true)
            .as('activeUser').then(function() {
                activeUser = this.activeUser;
            }
        );
    });

    it('respects default project tab user preference', function() {
        // Create test project
        cy.createProject({
            owningUser: activeUser,
            projectName: testProjectName,
        });

        cy.loginAs(activeUser);

        // Verify default tab on load
        cy.get('[data-cy=process-data]').should('exist');
        cy.get('[data-cy=process-run]').should('not.exist');

        // Navigate to project and switch to runs tab
        cy.assertDataExplorerContains(testProjectName, true).click();
        cy.get('[data-cy=mpv-tabs]').contains("Workflow Runs").click();

        // Verify tab state
        cy.get('[data-cy=process-data]').should('not.exist');
        cy.get('[data-cy=process-run]').should('exist');

        // Navigate back to home
        cy.doBreadcrumbsNavigation("Home Projects");

        // Verify tabs switched back to data and project visible
        cy.assertDataExplorerContains(testProjectName, true);
        cy.get('[data-cy=process-data]').should('exist');
        cy.get('[data-cy=process-run]').should('not.exist');

        // Change default tab preferecne
        cy.doAccountMenuAction("Preferences");
        cy.get('input[type=radio][name="prefs.wb.default_project_tab"][value="Workflow Runs"]').click();
        cy.get('[data-cy=preferences-form] button[type=submit]').click();

        // Verify new default tab
        cy.doSidePanelNavigation("Home Projects");
        cy.get('[data-cy=process-data]').should('not.exist');
        cy.get('[data-cy=process-run]').should('exist');
        cy.assertDataExplorerContains(testProjectName, false);

        // Switch to data tab and navigate to project
        cy.get('[data-cy=mpv-tabs]').contains("Data").click();
        cy.get('[data-cy=process-data]').should('exist');
        cy.get('[data-cy=process-run]').should('not.exist');
        cy.assertDataExplorerContains(testProjectName, true).click();

        // Verify switched back to runs and project absent
        cy.get('[data-cy=process-data]').should('not.exist');
        cy.get('[data-cy=process-run]').should('exist');
        cy.assertDataExplorerContains(testProjectName, false);

        // Change default tab preferecne back
        cy.doAccountMenuAction("Preferences");
        cy.get('input[type=radio][name="prefs.wb.default_project_tab"][value="Data"]').click();
        cy.get('[data-cy=preferences-form] button[type=submit]').click();
    });

});
