// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

describe('Login tests', function() {
    before(function() {
        this.email = `account_${Math.random()}@example.com`
        this.password = Math.random()
        this.firstName = 'Test'
        this.lastName = 'User'
    })

    beforeEach(function() {
        cy.visit('/')
        cy.contains('Please log in')
        cy.get('button').contains('Log in').click()
        cy.url().should('contain', "/users/sign_in")
    })

    it('register a new user', function() {
        cy.get('a[role=button]').contains('Sign up for a new account').click()
        cy.url().should('contain', '/users/sign_up')
        cy.get('input[name="user[first_name]"]').type(this.firstName)
        cy.get('input[name="user[last_name]"]').type(this.lastName)
        cy.get('input[name="user[email]"]').type(this.email)
        cy.get('input[name="user[password]"]').type(this.password)
        cy.get('input[name="user[password_confirmation]"]').type(this.password)
        cy.get('input[type=submit]').contains('Sign up').click()
        cy.url().should('contain', '/projects/')
        cy.get('button[title="Account Management"]').click()
        cy.get('ul[role=menu] > li[role=menuitem]').contains(`${this.firstName} ${this.lastName}`)
    })

    it('logs in successfully', function() {
        cy.get('input[type=email]').type(this.email)
        cy.get('input[type=password]').type(this.password)
        cy.get('input[type=submit]').contains('Sign in').click()
        cy.url().should('contain', '/projects/')
        cy.get('button[title="Account Management"]').click()
        cy.get('ul[role=menu] > li[role=menuitem]').contains(`${this.firstName} ${this.lastName}`)
    })

    it('fails to log in with incorrect password', function() {
        cy.get('input[type=email]').type(this.email)
        cy.get('input[type=password]').type('incorrect')
        cy.get('input[type=submit]').contains('Sign in').click()
        cy.url().should('contain', "/users/sign_in")
        cy.get('div.alert').contains('Invalid email or password')
    })
})