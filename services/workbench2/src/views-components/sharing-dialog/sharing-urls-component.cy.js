// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { SharingURLsComponent } from './sharing-urls-component';
import { ThemeProvider } from "@mui/material";
import { CustomTheme } from 'common/custom-theme';
import { Provider } from "react-redux";
import { configureStore } from "store/store";
import { createBrowserHistory } from "history";

describe("<SharingURLsComponent />", () => {
    let props;
    const store = configureStore(createBrowserHistory());

    beforeEach(() => {
        props = {
            collectionUuid: 'collection-uuid',
            sharingURLsPrefix: 'sharing-urls-prefix',
            sharingTokens: [
                {
                    uuid: 'token-uuid1',
                    apiToken: 'aaaaaaaaaa',
                    expiresAt: '2009-01-03T18:15:00Z',
                },
                {
                    uuid: 'token-uuid2',
                    apiToken: 'bbbbbbbbbb',
                    expiresAt: '2009-01-03T18:15:01Z',
                },
            ],
            onCopy: cy.stub(),
            onDeleteSharingToken: cy.stub().as('onDeleteSharingToken'),
        };
        cy.mount(
            <Provider store={store}>
                <ThemeProvider theme={CustomTheme}>
                    <SharingURLsComponent {...props} />
                </ThemeProvider>
            </Provider>);
    });

    it("renders a list of sharing URLs", () => {
        // Check number of URLs
        cy.get('a').should('have.length', 2);
        // Check 1st URL
        cy.get('a').eq(0).should('contain', `Token aaaaaaaa... expiring at: ${new Date(props.sharingTokens[0].expiresAt).toLocaleString()}`);
        cy.get('a').eq(0).should('have.attr', 'href', `${props.sharingURLsPrefix}/c=${props.collectionUuid}/t=${props.sharingTokens[0].apiToken}/_/`);
        // Check 2nd URL
        cy.get('a').eq(1).should('contain', `Token bbbbbbbb... expiring at: ${new Date(props.sharingTokens[1].expiresAt).toLocaleString()}`);
        cy.get('a').eq(1).should('have.attr', 'href', `${props.sharingURLsPrefix}/c=${props.collectionUuid}/t=${props.sharingTokens[1].apiToken}/_/`);
    });

    it("renders a list URLs with collection UUIDs as subdomains", () => {
        props.sharingURLsPrefix = '*.sharing-urls-prefix';
        const sharingPrefix = '.sharing-urls-prefix';
        cy.mount(
            <Provider store={store}>
                <ThemeProvider theme={CustomTheme}>
                    <SharingURLsComponent {...props} />
                </ThemeProvider>
            </Provider>);

        cy.get('a').eq(0).should('have.attr', 'href', `${props.collectionUuid}${sharingPrefix}/t=${props.sharingTokens[0].apiToken}/_/`);
        cy.get('a').eq(1).should('have.attr', 'href', `${props.collectionUuid}${sharingPrefix}/t=${props.sharingTokens[1].apiToken}/_/`);
    });

    it("renders a list of URLs with no expiration", () => {
        props.sharingTokens[0].expiresAt = null;
        props.sharingTokens[1].expiresAt = null;
        cy.mount(
            <Provider store={store}>
                <ThemeProvider theme={CustomTheme}>
                    <SharingURLsComponent {...props} />
                </ThemeProvider>
            </Provider>);
        cy.get('a').eq(0).should('contain', `Token aaaaaaaa... with no expiration date`);
        cy.get('a').eq(1).should('contain', `Token bbbbbbbb... with no expiration date`);
    });

    it("calls delete token handler when delete button is clicked", () => {
        cy.get('button').eq(0).click();
        cy.get('@onDeleteSharingToken').should('be.calledWith', props.sharingTokens[0].uuid);
    });
});