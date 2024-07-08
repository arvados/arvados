// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { ThemeProvider, StyledEngineProvider } from '@mui/material';
import { CustomTheme } from 'common/custom-theme';
import { WebDavS3InfoDialog } from './webdav-s3-dialog';
import { COLLECTION_WEBDAV_S3_DIALOG_NAME } from 'store/collections/collection-info-actions';
import { Provider } from "react-redux";
import { createStore, combineReducers } from 'redux';

describe('WebDavS3InfoDialog', () => {
    let props;
    let store;

    beforeEach(() => {
        const initialDialogState = {
            [COLLECTION_WEBDAV_S3_DIALOG_NAME]: {
                open: true,
                data: {
                    uuid: "zzzzz-4zz18-b1f8tbldjrm8885",
                    token: "v2/zzzzb-jjjjj-123123/xxxtokenxxx",
                    downloadUrl: "https://download.example.com",
                    collectionsUrl: "https://collections.example.com",
                    localCluster: "zzzzz",
                    username: "bobby",
                    activeTab: 0,
                    setActiveTab: (event, tabNr) => { }
                }
            }
        };
        const initialAuthState = {
            localCluster: "zzzzz",
            remoteHostsConfig: {},
            sessions: {},
        };
        store = createStore(combineReducers({
            dialog: (state = initialDialogState, action) => state,
            auth: (state = initialAuthState, action) => state,
        }));

        props = {
            classes: {
                details: 'details',
            }
        };
    });

    it('render cyberduck tab', () => {
        store.getState().dialog[COLLECTION_WEBDAV_S3_DIALOG_NAME].data.activeTab = 0;
        // when
        cy.mount(
            <StyledEngineProvider injectFirst>
                <ThemeProvider theme={CustomTheme}>
                    <Provider store={store}>
                        <WebDavS3InfoDialog {...props} />
                    </Provider>
                </ThemeProvider>
            </StyledEngineProvider>
        );

        // then
        cy.contains("davs://bobby@download.example.com/c=zzzzz-4zz18-b1f8tbldjrm8885").should('exist');
    });

    it('render win/mac tab', () => {
        store.getState().dialog[COLLECTION_WEBDAV_S3_DIALOG_NAME].data.activeTab = 1;
        // when
        cy.mount(
            <StyledEngineProvider injectFirst>
                <ThemeProvider theme={CustomTheme}>
                    <Provider store={store}>
                        <WebDavS3InfoDialog {...props} />
                    </Provider>
                </ThemeProvider>
            </StyledEngineProvider>
        );

        // then
        cy.contains("https://download.example.com/c=zzzzz-4zz18-b1f8tbldjrm8885").should('exist');
    });

    it('render s3 tab with federated token', () => {
        store.getState().dialog[COLLECTION_WEBDAV_S3_DIALOG_NAME].data.activeTab = 2;
        // when
        cy.mount(
            <StyledEngineProvider injectFirst>
                <ThemeProvider theme={CustomTheme}>
                    <Provider store={store}>
                        <WebDavS3InfoDialog {...props} />
                    </Provider>
                </ThemeProvider>
            </StyledEngineProvider>
        );

        // then
        cy.contains("Secret Keyv2_zzzzb-jjjjj-123123_xxxtokenxxx").should('exist');
    });

    it('render s3 tab with local token', () => {
        store.getState().dialog[COLLECTION_WEBDAV_S3_DIALOG_NAME].data.activeTab = 2;
        store.getState().dialog[COLLECTION_WEBDAV_S3_DIALOG_NAME].data.token = "v2/zzzzz-jjjjj-123123/xxxtokenxxx";
        // when
        cy.mount(
            <StyledEngineProvider injectFirst>
                <ThemeProvider theme={CustomTheme}>
                    <Provider store={store}>
                        <WebDavS3InfoDialog {...props} />
                    </Provider>
                </ThemeProvider>
            </StyledEngineProvider>
        );

        // then
        cy.contains("Access Keyzzzzz-jjjjj-123123Secret Keyxxxtokenxxx").should('exist');
    });

    it('render cyberduck tab with wildcard DNS', () => {
        store.getState().dialog[COLLECTION_WEBDAV_S3_DIALOG_NAME].data.activeTab = 0;
        store.getState().dialog[COLLECTION_WEBDAV_S3_DIALOG_NAME].data.collectionsUrl = "https://*.collections.example.com";
        // when
        cy.mount(
            <StyledEngineProvider injectFirst>
                <ThemeProvider theme={CustomTheme}>
                    <Provider store={store}>
                        <WebDavS3InfoDialog {...props} />
                    </Provider>
                </ThemeProvider>
            </StyledEngineProvider>
        );

        // then
        cy.contains("davs://bobby@zzzzz-4zz18-b1f8tbldjrm8885.collections.example.com").should('exist');
    });

});
