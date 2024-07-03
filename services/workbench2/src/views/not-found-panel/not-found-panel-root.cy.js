// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { ThemeProvider, StyledEngineProvider } from '@mui/material';
import { CustomTheme } from 'common/custom-theme';
import { NotFoundPanelRoot } from './not-found-panel-root';

describe('NotFoundPanelRoot', () => {
    let props;

    beforeEach(() => {
        props = {
            classes: {
                root: 'root',
                title: 'title',
                active: 'active',
            },
            clusterConfig: {
                Mail: {
                    SupportEmailAddress: 'support@example.com'
                }
            },
            location: null,
        };
    });

    it('should render component', () => {
        // given
        const expectedMessage = "The page you requested was not found";

        // when
        cy.mount(
            <StyledEngineProvider injectFirst>
                <ThemeProvider theme={CustomTheme}>
                    <NotFoundPanelRoot {...props} />
                </ThemeProvider>
            </StyledEngineProvider>
            );

        // then
        cy.get('p').contains(expectedMessage);
    });

    it('should render component without email url when no email', () => {
        // setup
        props.clusterConfig.Mail.SupportEmailAddress = '';

        // when
        cy.mount(
            <StyledEngineProvider injectFirst>
                <ThemeProvider theme={CustomTheme}>
                    <NotFoundPanelRoot {...props} />
                </ThemeProvider>
            </StyledEngineProvider>
            );

        // then
        cy.get('a').should('not.exist');
    });

    it('should render component with additional message and email url', () => {
        // given
        const hash = '123hash123';
        const pathname = `/collections/${hash}`;

        // setup
        props.location = {
            pathname,
        };

        // when
        cy.mount(
            <StyledEngineProvider injectFirst>
                <ThemeProvider theme={CustomTheme}>
                    <NotFoundPanelRoot {...props} />
                </ThemeProvider>
            </StyledEngineProvider>
            );

        // then
        cy.get('p').eq(0).contains(hash);

        // and
        cy.get('a').should('have.length', 1);
    });
});