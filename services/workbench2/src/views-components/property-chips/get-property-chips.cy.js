// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { Provider } from 'react-redux';
import { createStore, combineReducers } from 'redux';
import { getPropertyChips } from './get-property-chips';

describe("getPropertyChips", () => {
    let store;

    beforeEach(() => {
        store = createStore(combineReducers({
            properties: (state = {}, action) => state,
        }));
    });

    it("renders property chips", () => {
        const resource = {
            properties: {
                foo: 'bar',
                baz: ['qux', 'quux', [ 'quuz' ]]
            }
        };
        cy.mount(
                    <Provider store={store}>
                        {getPropertyChips(resource, {})}
                    </Provider>);
        cy.get('html').should('contain', 'foo: bar');
        cy.get('html').should('contain', 'baz: qux');
        cy.get('html').should('contain', 'baz: quux');
        cy.get('html').should('contain', 'baz: quuz');
    });

    it("filters out objects", () => {
        const resource = {
            properties: {
                foo: 'bar',
                baz: { qux: 'quux' }
            }
        };
        cy.mount(
                    <Provider store={store}>
                        {getPropertyChips(resource, {})}
                    </Provider>);
        cy.get('html').should('contain', 'foo: bar');
        // should not contain baz at all, because its value is an object
        cy.get('html').should('not.contain', 'baz');
    });
});