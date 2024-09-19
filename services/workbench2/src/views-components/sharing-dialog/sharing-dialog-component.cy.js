// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { Provider } from 'react-redux';
import { combineReducers, createStore } from 'redux';
import { SharingDialogComponent } from './sharing-dialog-component';
import {
    extractUuidObjectType,
    ResourceObjectType
} from 'models/resource';
import { ThemeProvider } from "@mui/material";
import { CustomTheme } from 'common/custom-theme';

describe("<SharingDialogComponent />", () => {
    let props;
    let store;

    beforeEach(() => {
        const initialAuthState = {
            config: {
                keepWebServiceUrl: 'http://example.com/',
                keepWebInlineServiceUrl: 'http://*.collections.example.com/',
                clusterConfig: {
                    Users: {
                        AnonymousUserToken: ""
                    }
                }
            }
        }
        store = createStore(combineReducers({
            auth: (state = initialAuthState, action) => state,
        }));

        props = {
            open: true,
            loading: false,
            saveEnabled: false,
            sharedResourceUuid: 'zzzzz-4zz18-zzzzzzzzzzzzzzz',
            privateAccess: true,
            sharingURLsNr: 2,
            sharingURLsDisabled: false,
            onClose: cy.stub(),
            onSave: cy.stub(),
            onCreateSharingToken: cy.stub(),
            refreshPermissions: cy.stub(),
        };
    });

    it("show sharing urls tab on collections when not disabled", () => {
        expect(props.sharingURLsDisabled).to.equal(false);
        expect(props.sharingURLsNr).to.equal(2);
        expect(extractUuidObjectType(props.sharedResourceUuid)).to.equal(ResourceObjectType.COLLECTION)
        cy.mount(
            <Provider store={store}>
                <ThemeProvider theme={CustomTheme}>
                    <SharingDialogComponent {...props} />
                </ThemeProvider>
            </Provider>);
        cy.get('html').should('contain', 'Sharing URLs (2)');

        // disable Sharing URLs UI
        props.sharingURLsDisabled = true;
        cy.mount(
            <Provider store={store}>
                <ThemeProvider theme={CustomTheme}>
                    <SharingDialogComponent {...props} />
                </ThemeProvider>
            </Provider>);
        cy.get('html').should('not.contain', 'Sharing URLs');
    });

    it("does not show sharing urls on non-collection resources", () => {
        props.sharedResourceUuid = 'zzzzz-j7d0g-0123456789abcde';
        expect(extractUuidObjectType(props.sharedResourceUuid)).to.not.equal(ResourceObjectType.COLLECTION);
        expect(props.sharingURLsDisabled).to.equal(false);
        cy.mount(
            <Provider store={store}>
                <ThemeProvider theme={CustomTheme}>
                    <SharingDialogComponent {...props} />
                </ThemeProvider>
            </Provider>);
        cy.get('html').should('not.contain', 'Sharing URLs');
    });
});
