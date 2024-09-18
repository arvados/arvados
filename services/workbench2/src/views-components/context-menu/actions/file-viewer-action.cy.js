// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { FileViewerAction } from './file-viewer-action';
import { ThemeProvider } from '@mui/material';
import { CustomTheme } from 'common/custom-theme';

describe('FileViewerAction', () => {
    let props;

    beforeEach(() => {
        props = {
            onClick: cy.stub().as('onClick'),
            href: 'https://example.com',
        };
    });

    it('should render properly and handle click', () => {
        // when
        cy.mount(
            <ThemeProvider theme={CustomTheme}>
                <FileViewerAction {...props} />
            </ThemeProvider>);
        
        // then
        cy.get('[data-cy=open-in-new-tab]').should('exist');
        cy.get('[data-cy=open-in-new-tab]').click();

        // and
        cy.get('@onClick').should('have.been.called');
    });
});