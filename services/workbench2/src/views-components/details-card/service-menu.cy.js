// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Provider } from "react-redux";
import { combineReducers, createStore } from "redux";
import { ServiceMenu } from './service-menu';

const apiToken = "v2/xxxxx-gj3su-000000000000000/00000000000000000000000000000000000000000000000000";

describe('ServiceMenu', () => {
    let store;

    beforeEach(() => {
        const initialAuthState = { apiToken: apiToken };

        store = createStore(combineReducers({
            auth: (state = initialAuthState, action) => state,
        }));

        // Stub the global window.open
        cy.window().then((win) => {
            cy.stub(win, 'open').as('open');
        });
    });

    it("displays single service", () => {
        const service = {
            access: "public",
            label: "My Service",
            initial_url: "http://example.com/",
        };

        cy.mount(
            <Provider store={store}>
                <ServiceMenu
                    buttonClass="serviceButton"
                    services={[service]}
                />
            </Provider>
        );

        // Verify button has correct text
        cy.get('.serviceButton').should('have.text', `Connect to ${service.label}`);

        // Click button
        cy.get('.serviceButton').click();
        // Verify correct URL opened
        cy.get('@open').should("have.been.calledWith", service.initial_url);
    });

    it("displays multiple services", () => {
        const services = [{
            access: "public",
            label: "Foo Service",
            initial_url: "http://example.com/foo",
            expected_url: "http://example.com/foo",
        }, {
            access: "private",
            label: "Bar Service",
            initial_url: "http://example.com/bar",
            expected_url: `http://example.com/bar?arvados_api_token=${apiToken}`,
        }, {
            access: "private",
            label: "A Secret Third Service",
            initial_url: "http://example.com/bar?existing=something",
            expected_url: `http://example.com/bar?arvados_api_token=${apiToken}&existing=something`,
        }];

        cy.mount(
            <Provider store={store}>
                <ServiceMenu
                    buttonClass="serviceButton"
                    services={services}
                />
            </Provider>
        );

        // Verify button has correct text
        cy.get('.serviceButton').should('have.text', "Connect to service");

        // Open menu and verify service list contains the correct items
        cy.get('.serviceButton').click();
        cy.get('#service-menu ul li')
            .should('have.length', services.length)
            .each((el, i) => {
                // Click on each service and verify opened url
                cy.wrap(el)
                    .should('have.text', services[i].label)
                    .click();
                cy.get('@open').should("have.been.calledWith", services[i].expected_url);
            });
    });
});
