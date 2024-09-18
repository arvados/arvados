// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { AutoLogoutComponent, LAST_ACTIVE_TIMESTAMP } from './auto-logout';

describe('<AutoLogoutComponent />', () => {
    let props;
    const sessionIdleTimeout = 300;
    const lastWarningDuration = 60;
    const eventListeners = {};

    beforeEach(() => {
        cy.clock();
        window.addEventListener = cy.stub((event, cb) => {
            eventListeners[event] = cb;
        });
        props = {
            sessionIdleTimeout: sessionIdleTimeout,
            lastWarningDuration: lastWarningDuration,
            doLogout: cy.spy().as('doLogout'),
            doWarn: cy.stub().as('doWarn'),
            doCloseWarn: cy.stub(),
        };
        cy.mount(<div><AutoLogoutComponent {...props} /></div>);
    });

    afterEach(() => {
        cy.clock().invoke('restore');
    });

    it('should logout after idle timeout', () => {
        cy.tick((sessionIdleTimeout-1)*1000);
        cy.get('@doLogout').should('not.have.been.called');
        cy.tick(1000);
        cy.get('@doLogout').should('have.been.called');
    });

    it('should warn the user previous to close the session', () => {
        cy.tick((sessionIdleTimeout-lastWarningDuration-1)*1000);
        cy.get('@doWarn').should('not.have.been.called');
        cy.tick(1000);
        cy.get('@doWarn').should('have.been.called');
    });

    it('should reset the idle timer when activity event is received', () => {
        cy.tick((sessionIdleTimeout-lastWarningDuration-1)*1000);
        cy.get('@doWarn').should('not.have.been.called');
        // Simulate activity from other window/tab
        eventListeners.storage({
            key: LAST_ACTIVE_TIMESTAMP,
            newValue: '42' // value currently doesn't matter
        })
        cy.tick(1000);
        // Warning should not appear because idle timer was reset
        cy.get('@doWarn').should('not.have.been.called');
    });
});