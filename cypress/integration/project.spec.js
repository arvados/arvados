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

    it('creates a new project with multiple properties', function() {
        const projName = `Test project (${Math.floor(999999 * Math.random())})`;
        cy.loginAs(activeUser);
        cy.get('[data-cy=side-panel-button]').click();
        cy.get('[data-cy=side-panel-new-project]').click();
        cy.get('[data-cy=form-dialog]')
            .should('contain', 'New Project')
            .within(() => {
                cy.get('[data-cy=name-field]').within(() => {
                    cy.get('input').type(projName);
                });

            });
        // Key: Color (IDTAGCOLORS) - Value: Magenta (IDVALCOLORS3)
        cy.get('[data-cy=form-dialog]').should('not.contain', 'Color: Magenta');
        cy.get('[data-cy=resource-properties-form]').within(() => {
            cy.get('[data-cy=property-field-key]').within(() => {
                cy.get('input').type('Color');
            });
            cy.get('[data-cy=property-field-value]').within(() => {
                cy.get('input').type('Magenta');
            });
            cy.root().submit();
            cy.get('[data-cy=property-field-value]').within(() => {
                cy.get('input').type('Pink');
            });
            cy.root().submit();
            cy.get('[data-cy=property-field-value]').within(() => {
                cy.get('input').type('Yellow');
            });
            cy.root().submit();
        });
        // Confirm proper vocabulary labels are displayed on the UI.
        cy.get('[data-cy=form-dialog]').should('contain', 'Color: Magenta');
        cy.get('[data-cy=form-dialog]').should('contain', 'Color: Pink');
        cy.get('[data-cy=form-dialog]').should('contain', 'Color: Yellow');

        cy.get('[data-cy=resource-properties-form]').within(() => {
            cy.get('[data-cy=property-field-key]').within(() => {
                cy.get('input').focus();
            });
            cy.get('[data-cy=property-field-key]').should('not.contain', 'Color');
        });

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
                // Pink is not in the test vocab
                {IDTAGCOLORS: ['IDVALCOLORS3', 'Pink', 'IDVALCOLORS1']});
        });

        // Open project edit via breadcrumbs
        cy.get('[data-cy=breadcrumbs]').contains(projName).rightclick();
        cy.get('[data-cy=context-menu]').contains('Edit').click();
        cy.get('[data-cy=form-dialog]').within(() => {
            cy.get('[data-cy=resource-properties-list]').within(() => {
                cy.get('div[role=button]').contains('Color: Magenta');
                cy.get('div[role=button]').contains('Color: Pink');
                cy.get('div[role=button]').contains('Color: Yellow');
            });
        });
        // Add another property
        cy.get('[data-cy=resource-properties-form]').within(() => {
            cy.get('[data-cy=property-field-key]').within(() => {
                cy.get('input').type('Animal');
            });
            cy.get('[data-cy=property-field-value]').within(() => {
                cy.get('input').type('Dog');
            });
            cy.root().submit();
        });
        cy.get('[data-cy=form-submit-btn]').click();
        // Reopen edit via breadcrumbs and verify properties
        cy.get('[data-cy=breadcrumbs]').contains(projName).rightclick();
        cy.get('[data-cy=context-menu]').contains('Edit').click();
        cy.get('[data-cy=form-dialog]').within(() => {
            cy.get('[data-cy=resource-properties-list]').within(() => {
                cy.get('div[role=button]').contains('Color: Magenta');
                cy.get('div[role=button]').contains('Color: Pink');
                cy.get('div[role=button]').contains('Color: Yellow');
                cy.get('div[role=button]').contains('Animal: Dog');
            });
        });
    });

    it('creates new project on home project and then a subproject inside it', function() {
        const createProject = function(name, parentName) {
            cy.get('[data-cy=side-panel-button]').click();
            cy.get('[data-cy=side-panel-new-project]').click();
            cy.get('[data-cy=form-dialog]')
                .should('contain', 'New Project')
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
        cy.goToPath(`/projects/${activeUser.user.uuid}`);
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

    it('navigates to the parent project after trashing the one being displayed', function() {
        cy.createGroup(activeUser.token, {
            name: `Test root project ${Math.floor(Math.random() * 999999)}`,
            group_class: 'project',
        }).as('testRootProject').then(function() {
            cy.createGroup(activeUser.token, {
                name : `Test subproject ${Math.floor(Math.random() * 999999)}`,
                group_class: 'project',
                owner_uuid: this.testRootProject.uuid,
            }).as('testSubProject');
        });
        cy.getAll('@testRootProject', '@testSubProject').then(function([testRootProject, testSubProject]) {
            cy.loginAs(activeUser);

            // Go to subproject and trash it.
            cy.goToPath(`/projects/${testSubProject.uuid}`);
            cy.get('[data-cy=side-panel-tree]').should('contain', testSubProject.name);
            cy.get('[data-cy=breadcrumb-last]')
                .should('contain', testSubProject.name)
                .rightclick();
            cy.get('[data-cy=context-menu]').contains('Move to trash').click();

            // Confirm that the parent project should be displayed.
            cy.get('[data-cy=breadcrumb-last]').should('contain', testRootProject.name);
            cy.url().should('contain', `/projects/${testRootProject.uuid}`);
            cy.get('[data-cy=side-panel-tree]').should('not.contain', testSubProject.name);

            // Checks for bugfix #17637.
            cy.get('[data-cy=not-found-content]').should('not.exist');
            cy.get('[data-cy=not-found-page]').should('not.exist');
        });
    });

    it('navigates to the root project after trashing the parent of the one being displayed', function() {
        cy.createGroup(activeUser.token, {
            name: `Test root project ${Math.floor(Math.random() * 999999)}`,
            group_class: 'project',
        }).as('testRootProject').then(function() {
            cy.createGroup(activeUser.token, {
                name : `Test subproject ${Math.floor(Math.random() * 999999)}`,
                group_class: 'project',
                owner_uuid: this.testRootProject.uuid,
            }).as('testSubProject').then(function() {
                cy.createGroup(activeUser.token, {
                    name : `Test sub subproject ${Math.floor(Math.random() * 999999)}`,
                    group_class: 'project',
                    owner_uuid: this.testSubProject.uuid,
                }).as('testSubSubProject');
            });
        });
        cy.getAll('@testRootProject', '@testSubProject', '@testSubSubProject').then(function([testRootProject, testSubProject, testSubSubProject]) {
            cy.loginAs(activeUser);

            // Go to innermost project and trash its parent.
            cy.goToPath(`/projects/${testSubSubProject.uuid}`);
            cy.get('[data-cy=side-panel-tree]').should('contain', testSubSubProject.name);
            cy.get('[data-cy=breadcrumb-last]').should('contain', testSubSubProject.name);
            cy.get('[data-cy=side-panel-tree]')
                .contains(testSubProject.name)
                .rightclick();
            cy.get('[data-cy=context-menu]').contains('Move to trash').click();

            // Confirm that the trashed project's parent should be displayed.
            cy.get('[data-cy=breadcrumb-last]').should('contain', testRootProject.name);
            cy.url().should('contain', `/projects/${testRootProject.uuid}`);
            cy.get('[data-cy=side-panel-tree]').should('not.contain', testSubProject.name);
            cy.get('[data-cy=side-panel-tree]').should('not.contain', testSubSubProject.name);

            // Checks for bugfix #17637.
            cy.get('[data-cy=not-found-content]').should('not.exist');
            cy.get('[data-cy=not-found-page]').should('not.exist');
        });
    });

    it('shows details panel when clicking on the info icon', () => {
        cy.createGroup(activeUser.token, {
            name: `Test root project ${Math.floor(Math.random() * 999999)}`,
            group_class: 'project',
        }).as('testRootProject').then(function(testRootProject) {
            cy.loginAs(activeUser);

            cy.get('[data-cy=side-panel-tree]').contains(testRootProject.name).click();

            cy.get('[data-cy=additional-info-icon]').click();

            cy.contains(testRootProject.uuid).should('exist');
        });
    });

    it('clears search input when changing project', () => {
        cy.createGroup(activeUser.token, {
            name: `Test root project ${Math.floor(Math.random() * 999999)}`,
            group_class: 'project',
        }).as('testProject1').then((testProject1) => {
            cy.shareWith(adminUser.token, activeUser.user.uuid, testProject1.uuid, 'can_write');
        });

        cy.getAll('@testProject1').then(function([testProject1]) {
            cy.loginAs(activeUser);

            cy.get('[data-cy=side-panel-tree]').contains(testProject1.name).click();

            cy.get('[data-cy=search-input] input').type('test123');

            cy.get('[data-cy=side-panel-tree]').contains('Projects').click();

            cy.get('[data-cy=search-input] input').should('not.have.value', 'test123');
        });
    });

    it('opens advanced popup for project with username', () => {
        const projectName = `Test project ${Math.floor(Math.random() * 999999)}`;

        cy.createGroup(adminUser.token, {
            name: projectName,
            group_class: 'project',
        }).as('mainProject')

        cy.getAll('@mainProject')
            .then(function ([mainProject]) {
                cy.loginAs(adminUser);

                cy.get('[data-cy=side-panel-tree]').contains('Groups').click();

                cy.get('[data-cy=uuid]').eq(0).invoke('text').then(uuid => {
                    cy.createLink(adminUser.token, {
                        name: 'can_write',
                        link_class: 'permission',
                        head_uuid: mainProject.uuid,
                        tail_uuid: uuid
                    });

                    cy.createLink(adminUser.token, {
                        name: 'can_write',
                        link_class: 'permission',
                        head_uuid: mainProject.uuid,
                        tail_uuid: activeUser.user.uuid
                    });

                    cy.get('[data-cy=side-panel-tree]').contains('Projects').click();

                    cy.get('main').contains(projectName).rightclick();

                    cy.get('[data-cy=context-menu]').contains('API Details').click();

                    cy.get('[role=tablist]').contains('METADATA').click();

                    cy.get('td').contains(uuid).should('exist');

                    cy.get('td').contains(activeUser.user.uuid).should('exist');
                });
        });
    });

    it('copies project URL to clipboard', () => {
        const projectName = `Test project (${Math.floor(999999 * Math.random())})`;

        cy.loginAs(activeUser);
        cy.get('[data-cy=side-panel-button]').click();
        cy.get('[data-cy=side-panel-new-project]').click();
        cy.get('[data-cy=form-dialog]')
            .should('contain', 'New Project')
            .within(() => {
                cy.get('[data-cy=name-field]').within(() => {
                    cy.get('input').type(projectName);
                });
                cy.get('[data-cy=form-submit-btn]').click();
            });

        cy.get('[data-cy=side-panel-tree]').contains('Projects').click();
        cy.get('[data-cy=project-panel]').contains(projectName).rightclick();
        cy.get('[data-cy=context-menu]').contains('Copy to clipboard').click();
        cy.window().then((win) => (
            win.navigator.clipboard.readText().then((text) => {
                expect(text).to.match(/https\:\/\/localhost\:[0-9]+\/projects\/[a-z0-9]{5}-[a-z0-9]{5}-[a-z0-9]{15}/,);
            })
        ));

    });
});
