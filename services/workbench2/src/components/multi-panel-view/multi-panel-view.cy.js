// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { MPVContainer } from './multi-panel-view';
import { ThemeProvider } from "@mui/material";
import { CustomTheme } from "common/custom-theme";

const PanelMock = ({panelName, panelMaximized, doHidePanel, doMaximizePanel, doUnMaximizePanel, panelIlluminated, panelRef, children, ...rest}) =>
    <div {...rest}>{children}</div>;

describe('<MPVContainer />', () => {
    let props;

    beforeEach(() => {
        props = {
            classes: {},
        };
    });

    it('should show default panel buttons for every child', () => {
        const childs = [
            <PanelMock key={1}>This is one panel</PanelMock>,
            <PanelMock key={2}>This is another panel</PanelMock>,
        ];
        cy.mount(
            <ThemeProvider theme={CustomTheme}>
                <MPVContainer {...props}>{[...childs]}</MPVContainer>
            </ThemeProvider>
        );
        //check if the buttons are rendered
        cy.get('button').should('have.length', 2);
        cy.get('button').eq(0).should('contain', 'Panel 1');
        cy.get('button').eq(1).should('contain', 'Panel 2');
        //check if the panels are rendered
        cy.contains('This is one panel');
        cy.contains('This is another panel');
    });

    it('should show panel when clicking on its button', () => {
        const childs = [
            <PanelMock key={1}>This is one panel</PanelMock>,
        ];
        props.panelStates = [
            {name: 'Initially invisible Panel', visible: false},
        ]

        cy.mount(
            <ThemeProvider theme={CustomTheme}>
                <MPVContainer {...props}>{[...childs]}</MPVContainer>
            </ThemeProvider>
        );

        // Initial state: panel not visible
        cy.contains('This is one panel').should('not.exist');
        cy.contains('All panels are hidden');
        
        // Panel visible when clicking on its button
        cy.get('button').click();
        cy.contains('This is one panel');
        cy.contains('All panels are hidden').should('not.exist');
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
            <ThemeProvider theme={CustomTheme}>
                <MPVContainer {...props}>{[...childs]}</MPVContainer>
            </ThemeProvider>
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

    it('should set panel hidden when requested', () => {
        const childs = [
            <PanelMock key={1}>This is one panel</PanelMock>,
        ];
        props.panelStates = [
            {name: 'First Panel', visible: false},
        ]
        cy.mount(
            <ThemeProvider theme={CustomTheme}>
                <MPVContainer {...props}>{[...childs]}</MPVContainer>
            </ThemeProvider>
        );
        cy.get('button').contains('First Panel');
        cy.contains('This is one panel').should('not.exist');
        cy.contains('All panels are hidden');
    });
});