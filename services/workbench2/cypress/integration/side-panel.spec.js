// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

describe('Side panel tests', function() {
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
        cy.getUser('user', 'Active', 'User', false, true)
            .as('activeUser').then(function() {
                activeUser = this.activeUser;
            }
        );
    })

    beforeEach(function() {
        cy.clearCookies()
        cy.clearLocalStorage()
    })

    it('enables the +NEW side panel button on users home project', function() {
        cy.loginAs(activeUser);
        cy.get('[data-cy=side-panel-button]')
            .should('exist')
            .and('not.be.disabled');
    })

    it('disables or enables the +NEW side panel button depending on project permissions', function() {
        cy.loginAs(activeUser);
        [true, false].map(function(isWritable) {
            cy.createGroup(adminUser.token, {
                name: `Test ${isWritable ? 'writable' : 'read-only'} project`,
                group_class: 'project',
            }).as('sharedGroup').then(function() {
                cy.createLink(adminUser.token, {
                    name: isWritable ? 'can_write' : 'can_read',
                    link_class: 'permission',
                    head_uuid: this.sharedGroup.uuid,
                    tail_uuid: activeUser.user.uuid
                })
                cy.goToPath(`/projects/${this.sharedGroup.uuid}`);
                cy.get('[data-cy=side-panel-button]')
                    .should('exist')
                    .and(`${isWritable ? 'not.' : ''}be.disabled`);
            })
        })
    })

    it('disables the +NEW side panel button on appropriate sections', function() {
        cy.loginAs(activeUser);
        [
            {url: '/shared-with-me', label: 'Shared with me'},
            {url: '/public-favorites', label: 'Public Favorites'},
            {url: '/favorites', label: 'My Favorites'},
            {url: '/all_processes', label: 'All Processes'},
            {url: '/trash', label: 'Trash'},
        ].map(function(section) {
            cy.goToPath(section.url);
            cy.get('[data-cy=breadcrumb-first]')
                .should('contain', section.label);
            cy.get('[data-cy=side-panel-button]')
                .should('exist')
                .and('be.disabled');
        })
    })

    it('disables the +NEW side panel button when viewing filter group', function() {
        cy.loginAs(adminUser);
        cy.createGroup(adminUser.token, {
            name: `my-favorite-filter-group`,
            group_class: 'filter',
            properties: {filters: []},
        }).as('myFavoriteFilterGroup').then(function (myFavoriteFilterGroup) {
            cy.goToPath(`/projects/${myFavoriteFilterGroup.uuid}`);
            cy.get('[data-cy=breadcrumb-last]').should('contain', 'my-favorite-filter-group');

            cy.get('[data-cy=side-panel-button]')
                    .should('exist')
                    .and(`be.disabled`);
        })
    })

    it('can edit project in side panel', () => {
        cy.createProject({
            owningUser: activeUser,
            targetUser: activeUser,
            projectName: 'mySharedWritableProject',
            canWrite: true,
            addToFavorites: false
        });

        cy.getAll('@mySharedWritableProject')
            .then(function ([mySharedWritableProject]) {
                cy.loginAs(activeUser);

                cy.get('[data-cy=side-panel-tree]').contains('Projects').click();

                const newProjectName = `New project name ${mySharedWritableProject.name}`;
                const newProjectDescription = `New project description ${mySharedWritableProject.name}`;

                cy.testEditProjectOrCollection('[data-cy=side-panel-tree]', mySharedWritableProject.name, newProjectName, newProjectDescription);
            });
    });

    it('side panel react to refresh when project data changes', () => {
        const project = 'writableProject';

        cy.createProject({
            owningUser: activeUser,
            targetUser: activeUser,
            projectName: project,
            canWrite: true,
            addToFavorites: false
        });

        cy.getAll('@writableProject').then(function ([writableProject]) {
            cy.loginAs(activeUser);
            cy.get('[data-cy=side-panel-tree]')
                .contains('Projects').click();
            cy.get('[data-cy=side-panel-tree]')
                .contains(writableProject.name).should('exist');
            cy.trashGroup(activeUser.token, writableProject.uuid).then(() => {
                cy.contains('Refresh').click();
                cy.contains(writableProject.name).should('not.exist');
            });
        });
    });
})
