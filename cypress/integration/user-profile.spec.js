// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

describe('User profile tests', function() {
    let activeUser;
    let adminUser;
    const roleGroupName = `Test role group (${Math.floor(999999 * Math.random())})`;
    const projectGroupName = `Test project group (${Math.floor(999999 * Math.random())})`;

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

    function assertProfileValues({
        firstName,
        lastName,
        email,
        username,
        org,
        org_email,
        role,
        website,
    }) {
        cy.get('[data-cy=profile-form] [data-cy=firstName] [data-cy=value]').contains(firstName);
        cy.get('[data-cy=profile-form] [data-cy=lastName] [data-cy=value]').contains(lastName);
        cy.get('[data-cy=profile-form] [data-cy=email] [data-cy=value]').contains(email);
        cy.get('[data-cy=profile-form] [data-cy=username] [data-cy=value]').contains(username);

        cy.get('[data-cy=profile-form] input[name="prefs.profile.organization"]').invoke('val').should('equal', org);
        cy.get('[data-cy=profile-form] input[name="prefs.profile.organization_email"]').invoke('val').should('equal', org_email);
        cy.get('[data-cy=profile-form] select[name="prefs.profile.role"]').invoke('val').should('equal', role);
        cy.get('[data-cy=profile-form] input[name="prefs.profile.website_url"]').invoke('val').should('equal', website);
    }

    function enterProfileValues({
        org,
        org_email,
        role,
        website,
    }) {
        cy.get('[data-cy=profile-form] input[name="prefs.profile.organization"]').clear();
        if (org) {
            cy.get('[data-cy=profile-form] input[name="prefs.profile.organization"]').type(org);
        }
        cy.get('[data-cy=profile-form] input[name="prefs.profile.organization_email"]').clear();
        if (org_email) {
            cy.get('[data-cy=profile-form] input[name="prefs.profile.organization_email"]').type(org_email);
        }
        cy.get('[data-cy=profile-form] select[name="prefs.profile.role"]').select(role);
        cy.get('[data-cy=profile-form] input[name="prefs.profile.website_url"]').clear();
        if (website) {
            cy.get('[data-cy=profile-form] input[name="prefs.profile.website_url"]').type(website);
        }
    }

    beforeEach(function() {
        cy.loginAs(adminUser);
        cy.goToPath('/my-account');
        enterProfileValues({
            org: '',
            org_email: '',
            role: '',
            website: '',
        });
        cy.get('[data-cy=profile-form] button[type="submit"]').click({force: true});

        cy.goToPath('/user/' + activeUser.user.uuid);
        enterProfileValues({
            org: '',
            org_email: '',
            role: '',
            website: '',
        });
        cy.get('[data-cy=profile-form] button[type="submit"]').click({force: true});
    });

    it('non-admin can edit own profile', function() {
        cy.loginAs(activeUser);

        cy.get('header button[title="Account Management"]').click();
        cy.get('#account-menu').contains('My account').click();

        // Admin tab should be hidden
        cy.get('div [role="tab"]').should('not.contain', 'ADMIN');

        // Check initial values
        assertProfileValues({
            firstName: 'Active',
            lastName: 'User',
            email: 'user@example.local',
            username: 'user',
            org: '',
            org_email: '',
            role: '',
            website: '',
        });

        // Change values
        enterProfileValues({
            org: 'Org name',
            org_email: 'email@example.com',
            role: 'Data Scientist',
            website: 'example.com',
        });

        // Submit
        cy.get('[data-cy=profile-form] button[type="submit"]').click();

        // Check new values
        assertProfileValues({
            firstName: 'Active',
            lastName: 'User',
            email: 'user@example.local',
            username: 'user',
            org: 'Org name',
            org_email: 'email@example.com',
            role: 'Data Scientist',
            website: 'example.com',
        });
    });

    it('non-admin cannot edit other profile', function() {
        cy.loginAs(activeUser);
        cy.goToPath('/user/' + adminUser.user.uuid);

        assertProfileValues({
            firstName: 'Admin',
            lastName: 'User',
            email: 'admin@example.local',
            username: 'admin',
            org: '',
            org_email: '',
            role: '',
            website: '',
        });

        // Inputs should be disabled
        cy.get('[data-cy=profile-form] input[name="prefs.profile.organization"]').should('be.disabled');
        cy.get('[data-cy=profile-form] input[name="prefs.profile.organization_email"]').should('be.disabled');
        cy.get('[data-cy=profile-form] select[name="prefs.profile.role"]').should('be.disabled');
        cy.get('[data-cy=profile-form] input[name="prefs.profile.website_url"]').should('be.disabled');

        // Submit should be disabled
        cy.get('[data-cy=profile-form] button[type="submit"]').should('be.disabled');

        // Admin tab should be hidden
        cy.get('div [role="tab"]').should('not.contain', 'ADMIN');
    });

    it('admin can edit own profile', function() {
        cy.loginAs(adminUser);

        cy.get('header button[title="Account Management"]').click();
        cy.get('#account-menu').contains('My account').click();

        // Admin tab should be visible
        cy.get('div [role="tab"]').should('contain', 'ADMIN');

        // Check initial values
        assertProfileValues({
            firstName: 'Admin',
            lastName: 'User',
            email: 'admin@example.local',
            username: 'admin',
            org: '',
            org_email: '',
            role: '',
            website: '',
        });

        // Change values
        enterProfileValues({
            org: 'Admin org name',
            org_email: 'admin@example.com',
            role: 'Researcher',
            website: 'admin.local',
        });
        cy.get('[data-cy=profile-form] button[type="submit"]').click();

        // Check new values
        assertProfileValues({
            firstName: 'Admin',
            lastName: 'User',
            email: 'admin@example.local',
            username: 'admin',
            org: 'Admin org name',
            org_email: 'admin@example.com',
            role: 'Researcher',
            website: 'admin.local',
        });
    });

    it('admin can edit other profile', function() {
        cy.loginAs(adminUser);
        cy.goToPath('/user/' + activeUser.user.uuid);

        // Check new values
        assertProfileValues({
            firstName: 'Active',
            lastName: 'User',
            email: 'user@example.local',
            username: 'user',
            org: '',
            org_email: '',
            role: '',
            website: '',
        });

        enterProfileValues({
            org: 'Changed org name',
            org_email: 'changed@example.com',
            role: 'Researcher',
            website: 'changed.local',
        });
        cy.get('[data-cy=profile-form] button[type="submit"]').click();

        // Check new values
        assertProfileValues({
            firstName: 'Active',
            lastName: 'User',
            email: 'user@example.local',
            username: 'user',
            org: 'Changed org name',
            org_email: 'changed@example.com',
            role: 'Researcher',
            website: 'changed.local',
        });
    });

    it('displays role groups on user profile', function() {
        cy.loginAs(adminUser);

        cy.createGroup(adminUser.token, {
            name: roleGroupName,
            group_class: 'role',
        }).as('roleGroup').then(function() {
            cy.createLink(adminUser.token, {
                name: 'can_write',
                link_class: 'permission',
                head_uuid: this.roleGroup.uuid,
                tail_uuid: adminUser.user.uuid
            });
            cy.createLink(adminUser.token, {
                name: 'can_write',
                link_class: 'permission',
                head_uuid: this.roleGroup.uuid,
                tail_uuid: activeUser.user.uuid
            });
        });

        cy.createGroup(adminUser.token, {
            name: projectGroupName,
            group_class: 'project',
        }).as('projectGroup').then(function() {
            cy.createLink(adminUser.token, {
                name: 'can_write',
                link_class: 'permission',
                head_uuid: this.projectGroup.uuid,
                tail_uuid: adminUser.user.uuid
            });
            cy.createLink(adminUser.token, {
                name: 'can_write',
                link_class: 'permission',
                head_uuid: this.projectGroup.uuid,
                tail_uuid: activeUser.user.uuid
            });
        });

        cy.goToPath('/user/' + activeUser.user.uuid);
        cy.get('div [role="tab"]').contains('GROUPS').click();
        cy.get('[data-cy=user-profile-groups-data-explorer]').contains(roleGroupName);
        cy.get('[data-cy=user-profile-groups-data-explorer]').should('not.contain', projectGroupName);

        cy.goToPath('/user/' + adminUser.user.uuid);
        cy.get('div [role="tab"]').contains('GROUPS').click();
        cy.get('[data-cy=user-profile-groups-data-explorer]').contains(roleGroupName);
        cy.get('[data-cy=user-profile-groups-data-explorer]').should('not.contain', projectGroupName);
    });

});
