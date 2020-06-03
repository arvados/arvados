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
        cy.visit(`/projects/${activeUser.user.uuid}`);
        cy.get('[data-cy=side-panel-button]')
            .should('exist')
            .and('not.be.disabled');
    })

    it('disables or enables the +NEW side panel button on depending on project permissions', function() {
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
                cy.visit(`/projects/${this.sharedGroup.uuid}`);
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
            {url: '/workflows', label: 'Workflows'},
            {url: '/all_processes', label: 'All Processes'},
            {url: '/trash', label: 'Trash'},
        ].map(function(section) {
            cy.visit(section.url);
            cy.get('[data-cy=breadcrumb-first]')
                .should('contain', section.label);
            cy.get('[data-cy=side-panel-button]')
                .should('exist')
                .and('be.disabled');
        })
    })

    it('creates new collection on home project', function() {
        cy.loginAs(activeUser);
        cy.visit(`/projects/${activeUser.user.uuid}`);
        cy.get('[data-cy=breadcrumb-first]').should('contain', 'Projects');
        cy.get('[data-cy=breadcrumb-last]').should('not.exist');
        // Create new collection
        cy.get('[data-cy=side-panel-button]').click();
        cy.get('[data-cy=side-panel-new-collection]').click();
        const collName = `Test collection (${Math.floor(999999 * Math.random())})`;
        cy.get('[data-cy=form-dialog]')
            .should('contain', 'New collection')
            .within(() => {
                cy.get('[data-cy=parent-field]').within(() => {
                    cy.get('input').should('have.value', 'Home project');
                })
                cy.get('[data-cy=name-field]').within(() => {
                    cy.get('input').type(collName);
                })
            })
        cy.get('[data-cy=form-submit-btn]').click();
        // Confirm that the user was taken to the newly created thing
        cy.get('[data-cy=form-dialog]').should('not.exist');
        cy.get('[data-cy=breadcrumb-first]').should('contain', 'Projects');
        cy.get('[data-cy=breadcrumb-last]').should('contain', collName);
    })

    it('creates new project on home project and then a subproject inside it', function() {
        const createProject = function(name, parentName) {
            cy.get('[data-cy=side-panel-button]').click();
            cy.get('[data-cy=side-panel-new-project]').click();
            cy.get('[data-cy=form-dialog]')
                .should('contain', 'New project')
                .within(() => {
                    cy.get('[data-cy=parent-field]').within(() => {
                        cy.get('input').invoke('val').then((val) => {
                            expect(val).to.include(parentName);
                        })
                    })
                    cy.get('[data-cy=name-field]').within(() => {
                        cy.get('input').type(name);
                    })
                })
            cy.get('[data-cy=form-submit-btn]').click();
        }

        cy.loginAs(activeUser);
        cy.visit(`/projects/${activeUser.user.uuid}`);
        cy.get('[data-cy=breadcrumb-first]').should('contain', 'Projects');
        cy.get('[data-cy=breadcrumb-last]').should('not.exist');
        // Create new project
        const projName = `Test project (${Math.floor(999999 * Math.random())})`;
        createProject(projName, 'Home project');
        // Confirm that the user was taken to the newly created thing
        cy.get('[data-cy=form-dialog]').should('not.exist');
        cy.get('[data-cy=breadcrumb-first]').should('contain', 'Projects');
        cy.get('[data-cy=breadcrumb-last]').should('contain', projName);
        // Create a subproject
        const subProjName = `Test project (${Math.floor(999999 * Math.random())})`;
        createProject(subProjName, projName);
        cy.get('[data-cy=form-dialog]').should('not.exist');
        cy.get('[data-cy=breadcrumb-first]').should('contain', 'Projects');
        cy.get('[data-cy=breadcrumb-last]').should('contain', subProjName);
    })
})