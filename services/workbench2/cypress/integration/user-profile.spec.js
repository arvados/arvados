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
        cy.get('[data-cy=profile-form] input[name="firstName"]').invoke('val').should('equal', firstName);
        cy.get('[data-cy=profile-form] input[name="lastName"]').invoke('val').should('equal', lastName);
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

    function assertContextMenuItems({
        account,
        activate,
        deactivate,
        login,
        setup
    }) {
        cy.get('[data-cy=user-profile-panel-options-btn]').click();
        cy.get('[data-cy=context-menu]').within(() => {
            cy.get('[role=button]').contains('API Details');

            cy.get('[role=button]').should(account ? 'contain' : 'not.contain', 'Account Settings');
            cy.get('[role=button]').should(activate ? 'contain' : 'not.contain', 'Activate User');
            cy.get('[role=button]').should(deactivate ? 'contain' : 'not.contain', 'Deactivate User');
            cy.get('[role=button]').should(login ? 'contain' : 'not.contain', 'Login As User');
            cy.get('[role=button]').should(setup ? 'contain' : 'not.contain', 'Setup User');
        });
        cy.get('div[role=presentation]').click();
    }

    beforeEach(function() {
        cy.updateResource(adminUser.token, 'users', adminUser.user.uuid, {
            prefs: {
                profile: {
                    organization: '',
                    organization_email: '',
                    role: '',
                    website_url: '',
                },
            },
        });
        cy.updateResource(adminUser.token, 'users', activeUser.user.uuid, {
            prefs: {
                profile: {
                    organization: '',
                    organization_email: '',
                    role: '',
                    website_url: '',
                },
            },
        });
    });

    it('non-admin can edit own profile', function() {
        cy.loginAs(activeUser);

        cy.get('header button[title="Account Management"]').click();
        cy.get('#account-menu').contains('My account').click();

        // Admin actions should be hidden, no account menu
        assertContextMenuItems({
            account: false,
            activate: false,
            deactivate: false,
            login: false,
            setup: false,
        });

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

        // Admin actions should be hidden, no account menu
        assertContextMenuItems({
            account: false,
            activate: false,
            deactivate: false,
            login: false,
            setup: false,
        });
    });

    it('admin can edit own profile', function() {
        cy.loginAs(adminUser);

        cy.get('header button[title="Account Management"]').click();
        cy.get('#account-menu').contains('My account').click();

        // Admin actions should be visible, no account menu
        assertContextMenuItems({
            account: false,
            activate: false,
            deactivate: true,
            login: false,
            setup: false,
        });

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

        // Admin actions should be visible, no account menu
        assertContextMenuItems({
            account: false,
            activate: false,
            deactivate: true,
            login: true,
            setup: false,
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

    it('allows performing admin functions', function() {
        cy.loginAs(adminUser);
        cy.goToPath('/user/' + activeUser.user.uuid);

        // Check that user is active
        cy.get('[data-cy=account-status]').contains('Active');
        cy.get('div [role="tab"]').contains('GROUPS').click();
        cy.get('[data-cy=user-profile-groups-data-explorer]').should('contain', 'All users');
        cy.get('div [role="tab"]').contains('PROFILE').click();
        assertContextMenuItems({
            account: false,
            activate: false,
            deactivate: true,
            login: true,
            setup: false,
        });

        // Deactivate user
        cy.get('[data-cy=user-profile-panel-options-btn]').click();
        cy.get('[data-cy=context-menu]').contains('Deactivate User').click();
        cy.get('[data-cy=confirmation-dialog-ok-btn]').click();

        // Check that user is deactivated
        cy.get('[data-cy=account-status]').contains('Inactive');
        cy.get('div [role="tab"]').contains('GROUPS').click();
        cy.get('[data-cy=user-profile-groups-data-explorer]').should('not.contain', 'All users');
        cy.get('div [role="tab"]').contains('PROFILE').click();
        assertContextMenuItems({
            account: false,
            activate: true,
            deactivate: false,
            login: true,
            setup: true,
        });

        // Setup user
        cy.get('[data-cy=user-profile-panel-options-btn]').click();
        cy.get('[data-cy=context-menu]').contains('Setup User').click();
        cy.get('[data-cy=confirmation-dialog-ok-btn]').click();

        // Check that user is setup
        cy.get('[data-cy=account-status]').contains('Setup');
        cy.get('div [role="tab"]').contains('GROUPS').click();
        cy.get('[data-cy=user-profile-groups-data-explorer]').should('contain', 'All users');
        cy.get('div [role="tab"]').contains('PROFILE').click();
        assertContextMenuItems({
            account: false,
            activate: true,
            deactivate: true,
            login: true,
            setup: false,
        });

        // Activate user
        cy.get('[data-cy=user-profile-panel-options-btn]').click();
        cy.get('[data-cy=context-menu]').contains('Activate User').click();
        cy.get('[data-cy=confirmation-dialog-ok-btn]').click();

        // Check that user is active
        cy.get('[data-cy=account-status]').contains('Active');
        cy.get('div [role="tab"]').contains('GROUPS').click();
        cy.get('[data-cy=user-profile-groups-data-explorer]').should('contain', 'All users');
        cy.get('div [role="tab"]').contains('PROFILE').click();
        assertContextMenuItems({
            account: false,
            activate: false,
            deactivate: true,
            login: true,
            setup: false,
        });

        // Deactivate and activate user skipping setup
        cy.get('[data-cy=user-profile-panel-options-btn]').click();
        cy.get('[data-cy=context-menu]').contains('Deactivate User').click();
        cy.get('[data-cy=confirmation-dialog-ok-btn]').click();
        // Check
        cy.get('[data-cy=account-status]').contains('Inactive');
        cy.get('div [role="tab"]').contains('GROUPS').click();
        cy.get('[data-cy=user-profile-groups-data-explorer]').should('not.contain', 'All users');
        cy.get('div [role="tab"]').contains('PROFILE').click();
        assertContextMenuItems({
            account: false,
            activate: true,
            deactivate: false,
            login: true,
            setup: true,
        });
        // reactivate
        cy.get('[data-cy=user-profile-panel-options-btn]').click();
        cy.get('[data-cy=context-menu]').contains('Activate User').click();
        cy.get('[data-cy=confirmation-dialog-ok-btn]').click();

        // Check that user is active
        cy.get('[data-cy=account-status]').contains('Active');
        cy.get('div [role="tab"]').contains('GROUPS').click();
        cy.get('[data-cy=user-profile-groups-data-explorer]').should('contain', 'All users');
        cy.get('div [role="tab"]').contains('PROFILE').click();
        assertContextMenuItems({
            account: false,
            activate: false,
            deactivate: true,
            login: true,
            setup: false,
        });
    });

});
