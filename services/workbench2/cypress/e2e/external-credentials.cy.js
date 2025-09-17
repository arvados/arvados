// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import moment from 'moment';

describe('External Credentials panel tests', function () {
    let activeUser;
    let adminUser;

    before(function () {
        // Set up common users
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

    beforeEach(() => {
        cy.loginAs(adminUser);
        cy.visit('/external_credentials');
    });

    it('displays empty state correctly', () => {
        cy.get('[data-cy=new-credential-button]').should('be.visible');
        cy.contains('External credentials list empty.').should('be.visible');
    });

    it('shows all expected columns', () => {
        const expectedColumns = ['Name', 'Description', 'Credential class', 'External ID', 'Expires at', 'Scopes'];

        expectedColumns.forEach((column) => {
            cy.get('thead').contains(column).should('be.visible');
        });
    });

    it('displays credential details correctly', () => {
        const expirationDate = moment().add(1, 'year');
        cy.createExternalCredential(adminUser.token, generateExternalCredential('Test Credential', expirationDate)).then((credential) => {
            cy.reload();

            cy.contains(credential.name).should('be.visible');
            cy.contains(credential.description).should('be.visible');
            cy.contains(credential.credential_class).should('be.visible');
            cy.contains(credential.external_id).should('be.visible');
            cy.contains(expirationDate.format('M/D/YYYY')).should('be.visible');
            cy.get('[data-cy=expired-badge]').should('not.exist');
            cy.get('[data-cy=expiring-badge]').should('not.exist');
            cy.contains(credential.scopes[0]).should('be.visible');
            cy.contains(credential.scopes[1]).should('be.visible');
        });
    });

    it('opens context menu on right click', () => {
        cy.createExternalCredential(adminUser.token, generateExternalCredential('Context Menu Test', moment().add(1, 'year'))).then((credential) => {
            cy.reload();

            cy.contains(credential.name).rightclick();
            cy.get('[data-cy=context-menu]').should('be.visible');
        });
    });

    it('displays expired and expiring badges correctly', () => {
        const expiringDate = moment().add(1, 'month');
        const expiredDate = moment().subtract(1, 'month');

        cy.createExternalCredential(adminUser.token, generateExternalCredential('Expiring Test Credential', expiringDate)).then((expiringCredential) => {
            cy.createExternalCredential(adminUser.token, generateExternalCredential('Expired Test Credential', expiredDate)).then((expiredCredential) => {
                cy.reload();

                cy.contains(expiringCredential.name).should('be.visible');
                cy.get('[data-cy=expiring-badge]').should('be.visible');

                cy.contains(expiredCredential.name).should('be.visible');
                cy.get('[data-cy=expired-badge]').should('be.visible');
            });
        });
    });

    it('creates new credential with add button', () => {
        const newCredentialName = `Test Credential ${Math.floor(Math.random() * 999999)}`;
        cy.get('[data-cy=new-credential-button]').click();
        cy.get('[data-cy=form-dialog]').should('be.visible').and('contain', 'New External Credential');

        // verify default values
        cy.get('input[name=credentialClass]').should('have.value', 'aws_access_key');
        cy.get('[data-cy=date-picker-input]').should('have.value', moment().add(1, 'year').format('MM/D/YYYY'));

        cy.get('input[name=name]').type(newCredentialName);
        cy.get('div[role=textbox]').type('Test Description');
        cy.get('input[name=externalId]').type('Test External ID');
        cy.get('input[name=string-array-input]').type('scope1{enter}');
        cy.get('input[name=string-array-input]').type('scope2{enter}');
        cy.get('input[name=secret]').type('test-secret');

        // modify default values
        cy.get('input[name=credentialClass]').type('_foo');
        cy.get('[data-cy=date-picker-input]').type('12/25/2099');

        cy.get('[data-cy=form-submit-btn]').click();

        cy.contains(newCredentialName).should('be.visible');
        cy.contains('Test Description').should('be.visible');
        cy.contains('aws_access_key_foo').should('be.visible');
        cy.contains('Test External ID').should('be.visible');
        cy.contains('scope1').should('be.visible');
        cy.contains('scope2').should('be.visible');
        cy.get('[data-cy=date-picker-input]').should('have.value', moment('12/25/2099', 'MM/D/YYYY').format('MM/D/YYYY'));

        // remove credential
        cy.contains(newCredentialName).rightclick();
        cy.get('[data-cy=context-menu]').contains('Remove').click();
        cy.get('[data-cy=confirmation-dialog-ok-btn]').click();
        cy.get('[data-cy=form-dialog]').should('not.exist');
    });

    it('edits an existing credential', () => {
        const newCredentialName = `Test Credential ${Math.floor(Math.random() * 999999)}`;
        const editCredentialName = `Edited Test Credential ${Math.floor(Math.random() * 999999)}`;
        cy.createExternalCredential(adminUser.token, generateExternalCredential(newCredentialName, moment().add(1, 'year'))).then((credential) => {
            cy.reload();
            cy.contains(credential.name).rightclick();
            cy.get('[data-cy=context-menu]').contains('Edit').click();
            cy.get('[data-cy=form-dialog]').should('be.visible').and('contain', 'Edit External Credential');

            cy.get('input[name=name]').clear().type(editCredentialName);
            cy.get('div[role=textbox]').clear().type('Edited Description');
            cy.get('input[name=credentialClass]').clear().type('Edited Credential Class');
            cy.get('input[name=externalId]').clear().type('Edited External ID');
            cy.get('[data-cy=date-picker-input]').type('01/01/2100');
            cy.get('input[name=secret]').should('have.value', '');
            cy.get('input[name=string-array-input]').type('new scope{enter}');
            //remove the first scope
            cy.get('svg[data-testid="CancelIcon"]').eq(0).click();

            cy.get('[data-cy=form-submit-btn]').click();

            cy.get('[data-cy=data-table]').contains(editCredentialName).should('be.visible');
            cy.contains('Edited Description').should('be.visible');
            cy.contains('Edited Credential Class').should('be.visible');
            cy.contains('Edited External ID').should('be.visible');
            cy.contains('new scope').should('be.visible');
            cy.get('td[data-cy=6]').contains(`${credential.scopes[1]}, new scope`).should('be.visible');
            cy.contains('1/1/2100').should('be.visible');

            // remove credential
            cy.contains(editCredentialName).rightclick();
            cy.get('[data-cy=context-menu]').contains('Remove').click();
            cy.get('[data-cy=confirmation-dialog-ok-btn]').click();
            cy.get('[data-cy=form-dialog]').should('not.exist');
        });
    });
});

export const generateExternalCredential = (
    name,
    expiresAt,
    description = `Test Description ${Math.floor(Math.random() * 999999)}`,
    credentialClass = `Test Credential Class ${Math.floor(Math.random() * 999999)}`,
    externalId = `Test External ID ${Math.floor(Math.random() * 999999)}`,
    scopes = [`scope1 ${Math.floor(Math.random() * 999999)}`, `scope2 ${Math.floor(Math.random() * 999999)}`],
    secret = 'test-secret'
) => {
    return {
        name: `${name} ${Math.floor(Math.random() * 999999)}`,
        description,
        credential_class: credentialClass,
        external_id: externalId,
        expires_at: expiresAt,
        scopes,
        secret,
    };
};
