// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

describe('Login tests', function() {
    before(function() {
        cy.resetDB();
    })

    beforeEach(function() {
        cy.arvadosFixture('users').as('users')
        cy.arvadosFixture('api_client_authorizations').as('client_auth')
        cy.clearCookies()
        cy.clearLocalStorage()
    })

    it('logs in successfully with correct token', function() {
        const active_user = this.users['active']
        const active_token = this.client_auth['active']['api_token']
        cy.visit('/token/?api_token='+active_token)
        cy.url().should('contain', '/projects/')
        cy.get('button[title="Account Management"]').click()
        cy.get('ul[role=menu] > li[role=menuitem]').contains(`${active_user['first_name']} ${active_user['last_name']}`)
    })

    it('fails to log in with expired token', function() {
        const expired_token = this.client_auth['expired']['api_token']
        cy.visit('/token/?api_token='+expired_token)
        cy.contains('Please log in')
        cy.url().should('not.contain', '/projects/')
    })

    it('fails to log in with no token', function() {
        cy.visit('/token/?api_token=')
        cy.contains('Please log in')
        cy.url().should('not.contain', '/projects/')
    })

    it('shows login page on first visit', function() {
        cy.visit('/')
        cy.contains('Please log in')
        cy.url().should('not.contain', '/projects/')
    })
})