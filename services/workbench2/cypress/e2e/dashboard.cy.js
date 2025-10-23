// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

describe('Main Dashboard', () => {
    let activeUser;
    let adminUser;

    const sectionsTitles = [
        'Favorites',
        'Recently Visited',
        'Recent Workflow Runs',
    ];

    before(function () {
        // Only set up common users once. These aren't set up as aliases because
        // aliases are cleaned up after every test. Also it doesn't make sense
        // to set the same users on beforeEach() over and over again, so we
        // separate a little from Cypress' 'Best Practices' here.
        cy.getUser('admin', 'Admin', 'User', true, true)
            .as('adminUser')
            .then(function () {
                adminUser = this.adminUser;
            });
        cy.getUser('user', 'Active', 'User', false, true)
            .as('activeUser')
            .then(function () {
                activeUser = this.activeUser;
            });
    });

    it('displays the appropriate sections', () => {
        cy.loginAs(activeUser);
        cy.get('[data-cy=tree-top-level-item]').contains('Dashboard').click();
        cy.get('[data-cy=dashboard-root]').should('exist');
        cy.get('[data-cy=breadcrumbs]').contains('Dashboard');
        cy.get('[data-cy=dashboard-root] [data-cy=dashboard-section]').should('have.length', sectionsTitles.length);
        sectionsTitles.forEach(title => {
            cy.get('[data-cy=dashboard-section]').contains(title).should('exist');
        });
    });
});

describe('Favorites section', () => {
    let activeUser;
    let adminUser;

    before(function () {
        // Only set up common users once. These aren't set up as aliases because
        // aliases are cleaned up after every test. Also it doesn't make sense
        // to set the same users on beforeEach() over and over again, so we
        // separate a little from Cypress' 'Best Practices' here.
        cy.getUser('admin', 'Admin', 'User', true, true)
            .as('adminUser')
            .then(function () {
                adminUser = this.adminUser;
            });
        cy.getUser('user', 'Active', 'User', false, true)
            .as('activeUser')
            .then(function () {
                activeUser = this.activeUser;
            });
    });

    it('displays the favorites section', () => {
        cy.loginAs(activeUser);
        cy.get('[data-cy=tree-top-level-item]').contains('Dashboard').click();
        cy.get('[data-cy=dashboard-section]').contains('Favorites').should('exist');
    });

    it('handles favorite pins operations', () => {
        cy.createProject({
            owningUser: adminUser,
            projectName: 'TestProject1',
            addToFavorites: true,
        }).as('testProject1');
        cy.createProject({
            owningUser: adminUser,
            projectName: 'TestProject2',
            addToFavorites: true,
        }).as('testProject2');
        cy.createProject({
            owningUser: adminUser,
            projectName: 'TestProject3',
            addToFavorites: true,
        }).as('testProject3');
        cy.getAll('@testProject1', '@testProject2', '@testProject3').then(
            ([testProject1, testProject2, testProject3]) => {
                cy.loginAs(adminUser);
                cy.get('[data-cy=tree-top-level-item]').contains('Dashboard').click();
                cy.get('[data-cy=dashboard-section]').contains('Favorites').should('exist');

                //verify favorite pins
                cy.get('[data-cy=favorite-pin]').should('have.length', 3);
                cy.get('[data-cy=favorite-pin]').eq(0).contains('TestProject3')
                cy.get('[data-cy=favorite-pin]').eq(1).contains('TestProject2')
                cy.get('[data-cy=favorite-pin]').eq(2).contains('TestProject1')

                //remove favorite pin
                cy.get(`[data-cy=${testProject1.head_uuid}-star]`).click();
                cy.get('[data-cy=favorite-pin]').should('have.length', 2);
                cy.get('[data-cy=favorite-pin]').contains('TestProject1').should('not.exist');

                //add favorite pin
                cy.doSidePanelNavigation('Home Projects');
                cy.doMPVTabSelect("Data");
                cy.get('[data-cy=data-table-row]').contains('TestProject1').rightclick();
                cy.get('[data-cy=context-menu]').contains('Add to favorites').click();
                cy.doSidePanelNavigation('Dashboard');
                cy.get('[data-cy=favorite-pin]').should('have.length', 3);
                cy.get('[data-cy=favorite-pin]').contains('TestProject1');

                //verify ordered by last favorited
                cy.get('[data-cy=favorite-pin]').eq(0).contains('TestProject1');
                cy.get('[data-cy=favorite-pin]').eq(1).contains('TestProject3');
                cy.get('[data-cy=favorite-pin]').eq(2).contains('TestProject2');

                //opens context menu
                cy.get('[data-cy=favorite-pin]').contains('TestProject1').rightclick();
                cy.get('[data-cy=context-menu]').contains('TestProject1');
                cy.get('body').click();

                //navs to item
                cy.get('[data-cy=favorite-pin]').contains('TestProject1').click();
                cy.get('[data-cy=project-details-card]').contains('TestProject1').should('exist');
            });
        });
});

describe('Recently Visited section', () => {
    let activeUser;
    let adminUser;

    before(function () {
        // Only set up common users once. These aren't set up as aliases because
        // aliases are cleaned up after every test. Also it doesn't make sense
        // to set the same users on beforeEach() over and over again, so we
        // separate a little from Cypress' 'Best Practices' here.
        cy.getUser('admin', 'Admin', 'User', true, true)
            .as('adminUser')
            .then(function () {
                adminUser = this.adminUser;
            });
        cy.getUser('user', 'Active', 'User', false, true)
            .as('activeUser')
            .then(function () {
                activeUser = this.activeUser;
            });
    });

    it('displays the recently visited section', () => {
        cy.loginAs(activeUser);
        cy.get('[data-cy=tree-top-level-item]').contains('Dashboard').click();
        cy.get('[data-cy=dashboard-section]').contains('Recently Visited').should('exist');
    });

    it('handles recently visited operations', () => {
        cy.createProject({
            owningUser: adminUser,
            projectName: 'TestProject1',
        }).as('testProject1');
        cy.createProject({
            owningUser: adminUser,
            projectName: 'TestProject2',
        }).as('testProject2');
        cy.createProject({
            owningUser: adminUser,
            projectName: 'TestProject3',
        }).as('testProject3');
        cy.getAll('@testProject1', '@testProject2', '@testProject3').then(
            ([testProject1, testProject2, testProject3]) => {
                cy.loginAs(adminUser);

                // visit some projects
                cy.doSidePanelNavigation('Home Projects');
                cy.get('[data-cy=side-panel-tree]').contains(testProject1.name).click();
                cy.get('[data-cy=project-details-card]').contains(testProject1.name).should('exist');
                cy.get('[data-cy=side-panel-tree]').contains(testProject2.name).click();
                cy.get('[data-cy=project-details-card]').contains(testProject2.name).should('exist');
                cy.get('[data-cy=side-panel-tree]').contains(testProject3.name).click();
                cy.get('[data-cy=project-details-card]').contains(testProject3.name).should('exist');

                // verify recently visited
                cy.get('[data-cy=tree-top-level-item]').contains('Dashboard').click();
                cy.get('[data-cy=dashboard-section]').contains('Recently Visited').should('exist');
                cy.get('[data-cy=dashboard-item-row]').should('have.length', 3);
                cy.get('[data-cy=dashboard-item-row]').eq(0).contains(testProject3.name);
                cy.get('[data-cy=dashboard-item-row]').eq(1).contains(testProject2.name);
                cy.get('[data-cy=dashboard-item-row]').eq(2).contains(testProject1.name);

                // opens context menu
                cy.get('[data-cy=dashboard-item-row]').contains(testProject1.name).rightclick();
                cy.get('[data-cy=context-menu]').contains(testProject1.name);
                cy.get('body').click();

                // navs to item
                cy.get('[data-cy=dashboard-item-row]').contains(testProject1.name).click();
                cy.get('[data-cy=project-details-card]').contains(testProject1.name).should('exist');

                // verify recently visited order has changed
                cy.get('[data-cy=tree-top-level-item]').contains('Dashboard').click();
                cy.get('[data-cy=dashboard-section]').contains('Recently Visited').should('exist');
                cy.get('[data-cy=dashboard-item-row]').should('have.length', 3);
                cy.get('[data-cy=dashboard-item-row]').eq(0).contains(testProject1.name);
                cy.get('[data-cy=dashboard-item-row]').eq(1).contains(testProject3.name);
                cy.get('[data-cy=dashboard-item-row]').eq(2).contains(testProject2.name);
            });
        });
});

describe('Recent Workflow Runs section', () => {
    let activeUser;
    let adminUser;

    before(function () {
        // Only set up common users once. These aren't set up as aliases because
        // aliases are cleaned up after every test. Also it doesn't make sense
        // to set the same users on beforeEach() over and over again, so we
        // separate a little from Cypress' 'Best Practices' here.
        cy.getUser('admin', 'Admin', 'User', true, true)
            .as('adminUser')
            .then(function () {
                adminUser = this.adminUser;
            });
        cy.getUser('user', 'Active', 'User', false, true)
            .as('activeUser')
            .then(function () {
                activeUser = this.activeUser;
            });
    });

    it('displays the recent workflow runs section', () => {
        cy.loginAs(activeUser);
        cy.get('[data-cy=tree-top-level-item]').contains('Dashboard').click();
        cy.get('[data-cy=dashboard-section]').contains('Recent Workflow Runs').should('exist');
    });

    it('handles recent workflow runs operations', () => {
        cy.setupDockerImage('arvados/jobs')
            .then((dockerImage) => {
                cy.createDefaultContainerRequest(
                    adminUser.token,
                    dockerImage,
                    { name: "test_container_request_1", state: "Committed" },
                ).as("containerRequest1");
                cy.createDefaultContainerRequest(
                    adminUser.token,
                    dockerImage,
                    { name: "test_container_request_2", state: "Committed" },
                ).as("containerRequest2");
                cy.createDefaultContainerRequest(
                    adminUser.token,
                    dockerImage,
                    { name: "test_container_request_3", state: "Committed" },
                ).as("containerRequest3");
            });
        cy.getAll("@containerRequest1", "@containerRequest2", "@containerRequest3")
            .then(function ([containerRequest1, containerRequest2, containerRequest3]) {
                cy.loginAs(adminUser);

                // verify recent workflow runs
                cy.get('[data-cy=tree-top-level-item]').contains('Dashboard').click();
                cy.get('[data-cy=dashboard-section]').contains('Recent Workflow Runs').should('exist');
                cy.get('[data-cy=dashboard-item-row]').should('have.length', 3);
                cy.get('[data-cy=dashboard-item-row]').eq(0).contains(containerRequest3.name);
                cy.get('[data-cy=dashboard-item-row]').eq(1).contains(containerRequest2.name);
                cy.get('[data-cy=dashboard-item-row]').eq(2).contains(containerRequest1.name);

                // open context menu
                cy.get('[data-cy=dashboard-item-row]').contains(containerRequest1.name).rightclick();
                cy.get('[data-cy=context-menu]').contains(containerRequest1.name);
                cy.get('body').click();

                // navs to item
                cy.get('[data-cy=dashboard-item-row]').contains(containerRequest1.name).click();
                cy.get('[data-cy=process-details-card]').contains(containerRequest1.name).should('exist');
            });
    });
});
