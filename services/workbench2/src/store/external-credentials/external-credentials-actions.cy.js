// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { createExternalCredential, updateExternalCredential, removeExternalCredentialPermanently } from './external-credentials-actions';

describe('External Credentials Actions', () => {
    let dispatch;

    const getState = () => ({
        auth: { user: { uuid: 'test-user' } },
        resources: {},
        multiselect: {
            checkedList: ['uuid1', 'uuid2'],
        },
    });

    const generateExternalCredential = (
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

    beforeEach(() => {
        if (dispatch) {
            dispatch.reset();
        }
        dispatch = cy.stub();
    });

    it('creates external credential with correct values', () => {
        const services = {
            externalCredentialsService: {
                create: cy.stub().resolves({}),
            },
        };

        const testCredential = generateExternalCredential('Test Credential', '2024-01-01');

        createExternalCredential(testCredential)(dispatch, getState, services);

        cy.wrap(services.externalCredentialsService.create).should('have.been.calledOnce').and('have.been.calledWith', testCredential);
    });

    it('updates external credential without secret when empty', () => {
        const services = {
            externalCredentialsService: {
                update: cy.stub().resolves({}),
            },
        };

        const testCredential = generateExternalCredential('Test Credential', '2024-01-01');
        testCredential.secret = '';
        testCredential.uuid = 'test-uuid';

        updateExternalCredential(testCredential)(dispatch, getState, services);

        cy.wrap(services.externalCredentialsService.update)
            .should('have.been.calledOnce')
            .and(
                'have.been.calledWith',
                'test-uuid',
                Cypress.sinon.match((obj) => {
                    return !obj.hasOwnProperty('secret');
                }),
                false
            );
    });

    it('updates external credential with secret when non-empty', () => {
        const services = {
            externalCredentialsService: {
                update: cy.stub().resolves({}),
            },
        };

        const testCredential = generateExternalCredential('Test Credential', '2024-01-01');
        testCredential.uuid = 'test-uuid';

        updateExternalCredential(testCredential)(dispatch, getState, services);

        cy.wrap(services.externalCredentialsService.update)
            .should('have.been.calledOnce')
            .and(
                'have.been.calledWith',
                'test-uuid',
                Cypress.sinon.match((obj) => {
                    return obj.secret === testCredential.secret;
                }),
                false
            );
    });

    it('removes multiple external credentials', (done) => {
        const deleteStub = cy.stub().resolves({});
        const services = {
            externalCredentialsService: {
                delete: deleteStub
            }
        };

        removeExternalCredentialPermanently('any-uuid')(dispatch, getState, services);

        // Give the Promise.allSettled time to process
        setTimeout(() => {
            cy.wrap(deleteStub)
                .should('have.been.calledTwice');
            cy.wrap(deleteStub.firstCall)
                .should('have.been.calledWith', 'uuid1');
            cy.wrap(deleteStub.secondCall)
                .should('have.been.calledWith', 'uuid2');
            done();
        }, 0);
    });
});
