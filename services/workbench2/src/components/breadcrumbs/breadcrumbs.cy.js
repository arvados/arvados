// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { Breadcrumbs } from "./breadcrumbs";
import { ThemeProvider, StyledEngineProvider } from "@mui/material";
import { CustomTheme } from 'common/custom-theme';
import { Provider } from "react-redux";
import { combineReducers, createStore } from "redux";

describe("<Breadcrumbs />", () => {

    let onClick;
    let resources = {};
    let store;
    beforeEach(() => {
        onClick = cy.spy().as('onClick');
        const initialAuthState = {
            config: {
                clusterConfig: {
                    Collections: {
                        ForwardSlashNameSubstitution: "/"
                    }
                }
            }
        }
        store = createStore(combineReducers({
            auth: (state = initialAuthState, action) => state,
        }));
    });

    it("renders one item", () => {
        const items = [
            { label: 'breadcrumb 1', uuid: '1' }
        ];
        cy.mount(
            <Provider store={store}>
                <StyledEngineProvider injectFirst>
                    <ThemeProvider theme={CustomTheme}>
                        <Breadcrumbs items={items} resources={resources} onClick={onClick} onContextMenu={cy.stub()} />
                    </ThemeProvider>
                </StyledEngineProvider>
            </Provider>);
        cy.get('button').should('have.length', 1);
        cy.get('button').should('have.text', 'breadcrumb 1');
        cy.get('[data-testid=ChevronRightIcon]').should('have.length', 0);
    });

    it("renders multiple items", () => {
        const items = [
            { label: 'breadcrumb 1', uuid: '1' },
            { label: 'breadcrumb 2', uuid: '2' }
        ];
        cy.mount(
            <Provider store={store}>
                <StyledEngineProvider injectFirst>
                    <ThemeProvider theme={CustomTheme}>
                        <Breadcrumbs items={items} resources={resources} onClick={onClick} onContextMenu={cy.stub()} />
                    </ThemeProvider>
                </StyledEngineProvider>
            </Provider>);
        cy.get('button').should('have.length', 2);
        cy.get('[data-testid=ChevronRightIcon]').should('have.length', 1);
    });

    it("calls onClick with clicked item", () => {
        const items = [
            { label: 'breadcrumb 1', uuid: '1' },
            { label: 'breadcrumb 2', uuid: '2' }
        ];
        cy.mount(
            <Provider store={store}>
                <StyledEngineProvider injectFirst>
                    <ThemeProvider theme={CustomTheme}>
                        <Breadcrumbs items={items} resources={resources} onClick={onClick} onContextMenu={cy.stub()} />
                    </ThemeProvider>
                </StyledEngineProvider>
            </Provider>);
        cy.get('button').eq(1).click();
        cy.get('@onClick').should('have.been.calledWith', Cypress.sinon.match.func, items[1]);
    });

});
