// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import axios from 'axios';
import { DownloadAction } from './download-action';
import { ThemeProvider } from '@mui/material';
import { CustomTheme } from 'common/custom-theme';

describe('<DownloadAction />', () => {
    let props;
    let zip;

    beforeEach(() => {
        props = {};
    });

    it('should return null if missing href or kind of file in props', () => {
        // when
        cy.mount(
                <ThemeProvider theme={CustomTheme}>
                    <DownloadAction {...props} />
                </ThemeProvider>);

        // then
        cy.get('[data-cy-root]').children().should('have.length', 0);
    });

    it('should return a element', () => {
        // setup
        props.href = '#';

        // when
        cy.mount(
            <ThemeProvider theme={CustomTheme}>
                <DownloadAction {...props} />
            </ThemeProvider>);

        // then
        cy.get('[data-cy-root]').children().should('have.length.greaterThan', 0);
    });

    it('should handle download', () => {
        // setup
        props = {
            href: ['file1'],
            kind: 'files',
            download: [],
            currentCollectionUuid: '123412-123123'
        };

        Cypress.on('uncaught:exception', (err, runnable) => {
            // Returning false here prevents Cypress from failing the test when axios returns 404
            if (err.message.includes('Request failed with status code 404')) {
                return false;
              }
              // Otherwise, let the error fail the test
              return true;
          });
        
        cy.intercept('GET', '*', (req) => {
            req.reply({
              statusCode: 200,
              body: { message: 'Mocked response' },
            });
          }).as('getData');
        
        cy.spy(axios, 'get').as('get');
        
        cy.mount(
            <ThemeProvider theme={CustomTheme}>
                <DownloadAction {...props} />
            </ThemeProvider>);

        // when
        cy.get('span').contains('Download selected').click();

        // then
        cy.get('@get').should('be.calledWith', props.href[0]);
    });
});