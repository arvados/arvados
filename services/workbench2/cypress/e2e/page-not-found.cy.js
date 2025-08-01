// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

describe('Page not found tests', function() {
    let adminUser;

    before(function() {
        cy.getUser('admin', 'Admin', 'User', true, true)
            .as('adminUser').then(function() {
                adminUser = this.adminUser;
            }
        );
    });

    it('shows not found page', function() {
        // when
        cy.loginAs(adminUser);
        cy.goToPath(`/this/is/an/invalid/route`);

        // then
        cy.get('[data-cy=not-found-page]').should('exist');
        cy.get('[data-cy=not-found-content]').should('exist');
    });


    it('shows not found popup', function() {
        // given
        [
            '/projects/zzzzz-j7d0g-nonexistingproj',
            '/projects/zzzzz-tpzed-nonexistinguser',
        ].forEach(function(path) {
            // Using de slower loginAs() method to avoid bumping into dialog
            // dismissal issues that are not related to this test.
            cy.loginAs(adminUser);

            // when
            cy.goToPath(path);
            cy.get('button').contains('Data').click();
            cy.waitForDom();

            // then
            cy.get('[data-cy=default-view]').should('exist');
        });

        [
            '/processes/zzzzz-xvhdp-nonexistingproc',
            '/collections/zzzzz-4zz18-nonexistingcoll'
        ].forEach(function(path) {
            cy.loginAs(adminUser);

            cy.goToPath(path);

            cy.get('[data-cy=not-found-view]').should('exist');
        });
    });

    it('shows not found popup in workflow tab', function() {
        cy.createProject({
            owningUser: adminUser,
            projectName: 'test-project',
        })
        cy.loginAs(adminUser);
        cy.waitForDom();

        cy.get('button').contains('Data').click();
        cy.get('[data-cy=project-panel]').contains("test-project").click();

        cy.get('[data-cy=mpv-tabs]').contains("Workflow Runs").click();
        cy.contains('No workflow runs found').should('exist');
    });
});

