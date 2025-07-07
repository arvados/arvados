// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { MPVContainer } from './multi-panel-view';
import { ThemeProvider } from "@mui/material";
import { CustomTheme } from "common/custom-theme";
import { Provider } from "react-redux";
import { combineReducers, createStore } from "redux";

const PanelMock = ({panelName, panelRef, children, ...rest}) =>
    <div {...rest}>{children}</div>;

describe('<MPVContainer />', () => {
    let props;
    let store;

    beforeEach(() => {
        props = {
            classes: {},
        };
        const initialRouterState = { location: null };
        store = createStore(combineReducers({
            router: (state = initialRouterState, action) => state,
        }));
    });

    it('should show default panel buttons for every child', () => {
        const childs = [
            <PanelMock key={1}>This is one panel</PanelMock>,
            <PanelMock key={2}>This is another panel</PanelMock>,
        ];
        cy.mount(
            <Provider store={store}>
                <ThemeProvider theme={CustomTheme}>
                    <MPVContainer {...props}>{[...childs]}</MPVContainer>
                </ThemeProvider>
            </Provider>
        );
        //check if the buttons are rendered
        cy.get('button').should('have.length', 2);
        cy.get('button').eq(0).should('contain', 'Panel 1');
        cy.get('button').eq(1).should('contain', 'Panel 2');
        //check if the panels are rendered
        cy.contains('This is one panel');
        cy.contains('This is another panel').should('not.exist');
    });

    it('should show panel when clicking on its button', () => {
        const childs = [
            <PanelMock key={1}>This is one panel</PanelMock>,
            <PanelMock key={2}>This is another panel</PanelMock>,
        ];

        cy.mount(
            <Provider store={store}>
                <ThemeProvider theme={CustomTheme}>
                    <MPVContainer {...props}>{[...childs]}</MPVContainer>
                </ThemeProvider>
            </Provider>
        );

        // Initial state: panel 2 not visible
        cy.contains('This is one panel');
        cy.contains('This is another panel').should('not.exist');

        // Panel visible when clicking on its button
        cy.get('button').contains('Panel 2').click();
        cy.contains('This is one panel').should('not.exist');
        cy.contains('This is another panel');
    });

    it('should show custom panel buttons when config provided', () => {
        const childs = [
            <PanelMock key={1}>This is one panel</PanelMock>,
            <PanelMock key={2}>This is another panel</PanelMock>,
        ];
        props.panelStates = [
            {name: 'First Panel'},
        ]
        cy.mount(
            <Provider store={store}>
                <ThemeProvider theme={CustomTheme}>
                    <MPVContainer {...props}>{[...childs]}</MPVContainer>
                </ThemeProvider>
            </Provider>
        );
        // First panel received the custom button naming
        cy.get('button').eq(0).should('contain', 'First Panel');
        cy.contains('This is one panel');

        // Second panel received the default button naming and hidden status by default
        cy.get('button').eq(1).should('contain', 'Panel 2');
        cy.contains('This is another panel').should('not.exist');
        cy.get('button').eq(1).click();
        cy.contains('This is another panel');
    });

    it('should set initial panel visibility according to panelStates prop', () => {
        const childs = [
            <PanelMock key={1}>This is one panel</PanelMock>,
            <PanelMock key={2}>This is another panel</PanelMock>,
        ];
        props.panelStates = [
            {name: 'First Panel'},
            {name: 'Second Panel', visible: true},
        ]
        cy.mount(
            <Provider store={store}>
                <ThemeProvider theme={CustomTheme}>
                    <MPVContainer {...props}>{[...childs]}</MPVContainer>
                </ThemeProvider>
            </Provider>
        );
        // Initial state: panel 2 not visible
        cy.contains('This is one panel').should('not.exist');
        cy.contains('This is another panel');
    });
});
