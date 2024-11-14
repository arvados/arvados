// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { AccountMenuComponent } from './account-menu';

describe('<AccountMenu />', () => {
    let props;
    beforeEach(() => {
        props = {
            classes: {},
            user: {
                email: 'email@example.com',
                firstName: 'User',
                lastName: 'Test',
                uuid: 'zzzzz-tpzed-testuseruuid',
                ownerUuid: '',
                username: 'testuser',
                prefs: {},
                isAdmin: false,
                isActive: true
            },
            currentRoute: '',
            workbenchURL: '',
            localCluser: 'zzzzz',
            apiToken: 'zzzzz',
            dispatch: cy.stub().as('dispatch'),
            onLogout: cy.stub().as('onLogout'),
            getNewExtraToken: cy.stub().as('getNewExtraToken'),
            openTokenDialog: cy.stub().as('openTokenDialog'),
        };
    });

    describe('Logout Menu Item', () => {
        it('should dispatch a logout action when clicked', () => {
            // response can be anything not 404
            cy.intercept('*', { foo: 'bar' });

            try {
                cy.mount(<AccountMenuComponent {...props} />);
                
                cy.get('button').should('exist').click({ force: true });
                cy.get('[data-cy="logout-menuitem"]').should('exist').click({ force: true });
                cy.get('@onLogout').should('have.been.called');

            } catch (error) {
                console.error(error)
            }
        });
    });
});
