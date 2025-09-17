// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Provider } from "react-redux";
import { combineReducers, createStore } from "redux";
import { ServiceMenu, injectTokenParam } from './service-menu';

const apiToken = "v2/xxxxx-gj3su-000000000000000/00000000000000000000000000000000000000000000000000";

describe('ServiceMenu', () => {
    let store;

    beforeEach(() => {
        const initialAuthState = { apiToken: apiToken };

        store = createStore(combineReducers({
            auth: (state = initialAuthState, action) => state,
        }));

        // Stub the global WebSocket
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

        // Open menu and verify service list contains 2 items
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

describe('injectTokenParam', () => {
    it('injects tokens into valid URLs', () => {
        const testCases = [{
            // Test normal case
            url: "http://example.com/",
            token: apiToken,
            result: `http://example.com/?arvados_api_token=${apiToken}`,
        },{
            // Test no trailing slash - URL constructor will add trailing slash
            url: "https://example.com",
            token: "foobar",
            result: "https://example.com/?arvados_api_token=foobar",
        },{
            // Test with basic auth
            url: "https://user:pass@example.com/",
            token: "baz",
            result: "https://user:pass@example.com/?arvados_api_token=baz",
        },{
            // Test with existing params
            url: "https://example.com/?foo=bar",
            token: "foo123",
            result: "https://example.com/?arvados_api_token=foo123&foo=bar",
        },{
            // Test with existing params and no slash - URL constructor will add slash
            url: "https://example.com?foo=bar",
            token: "foo123",
            result: "https://example.com/?arvados_api_token=foo123&foo=bar",
        },{
            // Test with no params but with question mark
            url: "http://example.com/?",
            token: "foobar",
            result: "http://example.com/?arvados_api_token=foobar",
        }];

        return Promise.all(testCases.map(async testCase => {
            const result = await injectTokenParam(testCase.url, testCase.token);
            expect(result).to.equal(testCase.result);
        }));
    });

    it('raises exceptions for invalid situations', () => {
        const invalidCases = [{
            url: "http://example.com",
            token: "",
            msg: "User token required",
        },{
            url: "",
            token: "foo",
            msg: "URL cannot be empty",
        }];

        return Promise.all(invalidCases.map(testCase => {
            const promise = injectTokenParam(testCase.url, testCase.token);

            return promise.then(() => {
                    throw new Error('Expected injectTokenParam() to return error but it did not. '
                        + `Expected error: "${testCase.msg}" given url "${testCase.url}" and token "${testCase.token}"`);
                }, (err) => {
                    // Verify the promise rejection reason
                    expect(err).to.equal(testCase.msg);
                }
            );
        }));

    });
});
