// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { CustomTheme } from 'common/custom-theme';
import { InactivePanelRoot } from './inactive-panel';
import { ThemeProvider, StyledEngineProvider } from '@mui/material';

describe('InactivePanel', () => {
    let props;

    beforeEach(() => {
        props = {
            classes: {
                root: 'root',
                title: 'title',
                ontop: 'ontop',
            },
            isLoginClusterFederation: false,
            inactivePageText: 'Inactive page content',
        };
    });

    it('should render content and link account option', () => {
        // given
        const expectedMessage = "Inactive page content";
        const expectedLinkAccountText = 'If you would like to use this login to access another account click "Link Account"';

        // when
        cy.mount(
            <StyledEngineProvider injectFirst>
                <ThemeProvider theme={CustomTheme}>
                    <InactivePanelRoot {...props} />
                </ThemeProvider>
            </StyledEngineProvider>
            );

        // then
        cy.get('p').eq(0).contains(expectedMessage);
        cy.get('p').eq(1).contains(expectedLinkAccountText);
    })

    it('should render content and link account warning on LoginCluster federations', () => {
        // given
        props.isLoginClusterFederation = true;
        const expectedMessage = "Inactive page content";
        const expectedLinkAccountText = 'If you would like to use this login to access another account, please contact your administrator';

        // when
        cy.mount(
            <StyledEngineProvider injectFirst>
                <ThemeProvider theme={CustomTheme}>
                    <InactivePanelRoot {...props} />
                </ThemeProvider>
            </StyledEngineProvider>
            );

        // then
        cy.get('p').eq(0).contains(expectedMessage);
        cy.get('p').eq(1).contains(expectedLinkAccountText);
    })
});