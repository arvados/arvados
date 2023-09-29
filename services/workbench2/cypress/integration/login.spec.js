// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

describe('Login tests', function() {
    let activeUser;
    let inactiveUser;
    let adminUser;
    let randomUser = {};

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
        cy.getUser('active', 'Active', 'User', false, true)
            .as('activeUser').then(function() {
                activeUser = this.activeUser;
            }
        );
        cy.getUser('inactive', 'Inactive', 'User', false, false)
            .as('inactiveUser').then(function() {
                inactiveUser = this.inactiveUser;
            }
        );
        randomUser.username = `randomuser${Math.floor(Math.random() * 999999)}`;
        randomUser.password = {
            crypt: 'zpAReoZzPnwmQ',
            clear: 'topsecret',
        };
        cy.exec(`useradd ${randomUser.username} -p ${randomUser.password.crypt}`);
    })

    after(function() {
        cy.exec(`userdel ${randomUser.username}`);
    })

    beforeEach(function() {
        cy.clearCookies()
        cy.clearLocalStorage()
    })

    it('shows login page on first visit', function() {
        cy.visit('/')
        cy.get('div#root').should('contain', 'Please log in')
        cy.url().should('not.contain', '/projects/')
    })

    it('shows login page with no token', function() {
        cy.visit('/token/?api_token=')
        cy.get('div#root').should('contain', 'Please log in')
        cy.url().should('not.contain', '/projects/')
    })

    it('shows inactive page to inactive user', function() {
        cy.visit(`/token/?api_token=${inactiveUser.token}`)
        cy.get('div#root').should('contain', 'Your account is inactive');
    })

    it('shows login page with invalid token', function() {
        cy.visit('/token/?api_token=nope')
        cy.get('div#root').should('contain', 'Please log in')
        cy.url().should('not.contain', '/projects/')
    })

    it('logs in successfully with valid user token', function() {
        cy.visit(`/token/?api_token=${activeUser.token}`);
        cy.url().should('contain', '/projects/');
        cy.get('div#root').should('contain', 'Arvados Workbench (zzzzz)');
        cy.get('div#root').should('not.contain', 'Your account is inactive');
        cy.get('button[title="Account Management"]').click();
        cy.get('ul[role=menu] > li[role=menuitem]').contains(
            `${activeUser.user.first_name} ${activeUser.user.last_name}`);
    })

    it('logs out when token no longer valid', function() {
        cy.createProject({
            owningUser: activeUser,
            projectName: `Test Project ${Math.floor(Math.random() * 999999)}`,
            addToFavorites: false
        }).as('testProject1');
        // Log in
        cy.visit(`/token/?api_token=${activeUser.token}`);
        cy.url().should('contain', '/projects/');
        cy.get('div#root').should('contain', 'Arvados Workbench (zzzzz)');
        cy.get('div#root').should('not.contain', 'Your account is inactive');
        cy.waitForDom();

        // Invalidate own token.
        const tokenUuid = activeUser.token.split('/')[1];
        cy.doRequest('PUT', `/arvados/v1/api_client_authorizations/${tokenUuid}`, {
            id: tokenUuid,
            api_client_authorization: JSON.stringify({
                api_token: `randomToken${Math.floor(Math.random() * 999999)}`
            })
        }, null, activeUser.token, true);
        // Should log the user out.

        cy.getAll('@testProject1').then(([testProject1]) => {
            cy.get('main').contains(testProject1.name).click();
            cy.get('div#root').should('contain', 'Please log in');
            // Should retain last visited url when auth is invalidated
            cy.url().should('contain', `/projects/${testProject1.uuid}`);
        })
    })

    it('logs in successfully with valid admin token', function() {
        cy.visit(`/token/?api_token=${adminUser.token}`);
        cy.url().should('contain', '/projects/');
        cy.get('div#root').should('contain', 'Arvados Workbench (zzzzz)');
        cy.get('div#root').should('not.contain', 'Your account is inactive');
        cy.get('button[title="Admin Panel"]').click();
        cy.get('ul[role=menu] > li[role=menuitem]')
            .contains('Repositories')
            .type('{esc}');
        cy.get('button[title="Account Management"]').click();
        cy.get('ul[role=menu] > li[role=menuitem]').contains(
            `${adminUser.user.first_name} ${adminUser.user.last_name}`);
    })

    it('fails to authenticate using the login form with wrong password', function() {
        cy.visit('/');
        cy.get('#username').type(randomUser.username);
        cy.get('#password').type('wrong password');
        cy.get("button span:contains('Log in')").click();
        cy.get('p#password-helper-text').should('contain', 'PAM: Authentication failure');
        cy.url().should('not.contain', '/projects/');
    })

    it('successfully authenticates using the login form', function() {
        cy.visit('/');
        cy.get('#username').type(randomUser.username);
        cy.get('#password').type(randomUser.password.clear);
        cy.get("button span:contains('Log in')").click();
        cy.url().should('contain', '/projects/');
        cy.get('div#root').should('contain', 'Arvados Workbench (zzzzz)');
        cy.get('div#root').should('contain', 'Your account is inactive');
        cy.get('button[title="Account Management"]').click();
        cy.get('ul[role=menu] > li[role=menuitem]').contains(randomUser.username);
    })
})
