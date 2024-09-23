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
        dispatch: cy.stub().as('dispatch'),
      };
    });

    describe('Logout Menu Item', () => {
        beforeEach(() => {
            cy.mount(<AccountMenuComponent {...props} />);
        });

        it('should dispatch a logout action when clicked', () => {
            cy.get('button').should('exist').click();
            cy.get('[data-cy="logout-menuitem"]').click();
            cy.get('@dispatch').should('have.been.calledWith', {
                payload: {deleteLinkData: true, preservePath: false},
                type: 'LOGOUT',
            });
        });
    });
});
