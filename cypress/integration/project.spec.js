// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

describe('Project tests', function() {
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
    });

    beforeEach(function() {
        cy.clearCookies();
        cy.clearLocalStorage();
    });

    it('adds creates a new project with properties', function() {
        const projName = `Test project (${Math.floor(999999 * Math.random())})`;
        cy.loginAs(activeUser);
        cy.get('[data-cy=side-panel-button]').click();
        cy.get('[data-cy=side-panel-new-project]').click();
        cy.get('[data-cy=form-dialog]')
            .should('contain', 'New project')
            .within(() => {
                cy.get('[data-cy=name-field]').within(() => {
                    cy.get('input').type(projName);
                });

            });
        // Key: Color (IDTAGCOLORS) - Value: Magenta (IDVALCOLORS3)
        cy.get('[data-cy=resource-properties-form]').within(() => {
            cy.get('[data-cy=property-field-key]').within(() => {
                cy.get('input').type('Color');
            });
            cy.get('[data-cy=property-field-value]').within(() => {
                cy.get('input').type('Magenta');
            });
            cy.root().submit();
        });
        // Confirm proper vocabulary labels are displayed on the UI.
        cy.get('[data-cy=form-dialog]').should('contain', 'Color: Magenta');

        // Create project and confirm the properties' real values.
        cy.get('[data-cy=form-submit-btn]').click();
        cy.get('[data-cy=breadcrumb-last]').should('contain', projName);
        cy.doRequest('GET', '/arvados/v1/groups', null, {
            filters: `[["name", "=", "${projName}"], ["group_class", "=", "project"]]`,
        })
        .its('body.items').as('projects')
        .then(function() {
            expect(this.projects).to.have.lengthOf(1);
            expect(this.projects[0].properties).to.deep.equal(
                {IDTAGCOLORS: 'IDVALCOLORS3'});
        });
    });

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
                        });
                    });
                    cy.get('[data-cy=name-field]').within(() => {
                        cy.get('input').type(name);
                    });
                });
            cy.get('[data-cy=form-submit-btn]').click();
        }

        cy.loginAs(activeUser);
        cy.doSearch(`${activeUser.user.uuid}`);
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
    });
})