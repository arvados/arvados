// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { LAST_REFRESH_TIMESTAMP, RefreshButton } from './refresh-button';

describe('<RefreshButton />', () => {
    let props;
    let replace;
    let urlPath;

    beforeEach(() => {
        props = {
            history: {
                replace: () => { },
            },
            classes: {},
        };

        replace = cy.spy(props.history, 'replace').as('replace');
    });

    it('should render without issues', () => {
        // when
        cy.mount(<RefreshButton {...props} />);

        // then
        cy.get('button').should('exist');
    });

    it('should pass window location to router', () => {
        // setup
        cy.mount(<RefreshButton {...props} />);

        cy.window().then((win) => {
            urlPath = win.location.pathname;
            expect(!!win.localStorage.getItem(LAST_REFRESH_TIMESTAMP)).to.equal(false);
        });

        // when
        cy.get('button').should('exist').click();

        // then
        cy.window().then((win) => {
            cy.get('@replace').should('have.been.calledWith', urlPath);
            expect(!!win.localStorage.getItem(LAST_REFRESH_TIMESTAMP)).not.to.equal(false);
        });
    });
});
