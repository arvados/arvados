// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { mount, configure, shallow } from 'enzyme';
import * as Adapter from "enzyme-adapter-react-16";
import { MuiThemeProvider, WithStyles } from '@material-ui/core';
import { CustomTheme } from 'common/custom-theme';
import { WebDavS3InfoDialog, CssRules } from './webdav-s3-dialog';
import { WithDialogProps } from 'store/dialog/with-dialog';
import { WebDavS3InfoDialogData, COLLECTION_WEBDAV_S3_DIALOG_NAME } from 'store/collections/collection-info-actions';
import { Provider } from "react-redux";
import { createStore, combineReducers } from 'redux';
import { configureStore, RootStore } from 'store/store';
import { createBrowserHistory } from "history";
import { createServices } from "services/services";

configure({ adapter: new Adapter() });

describe('WebDavS3InfoDialog', () => {
    let props: WithDialogProps<WebDavS3InfoDialogData> & WithStyles<CssRules>;
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
                    setActiveTab: (event: any, tabNr: number) => { }
                }
            }
        };
        const initialAuthState = {
            localCluster: "zzzzz",
            remoteHostsConfig: {},
            sessions: {},
        };
        store = createStore(combineReducers({
            dialog: (state: any = initialDialogState, action: any) => state,
            auth: (state: any = initialAuthState, action: any) => state,
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
        const wrapper = mount(
            <MuiThemeProvider theme={CustomTheme}>
                <Provider store={store}>
                    <WebDavS3InfoDialog {...props} />
                </Provider>
            </MuiThemeProvider>
        );

        // then
        expect(wrapper.text()).toContain("davs://bobby@download.example.com/by_id/zzzzz-4zz18-b1f8tbldjrm8885");
    });

    it('render win/mac tab', () => {
        store.getState().dialog[COLLECTION_WEBDAV_S3_DIALOG_NAME].data.activeTab = 1;
        // when
        const wrapper = mount(
            <MuiThemeProvider theme={CustomTheme}>
                <Provider store={store}>
                    <WebDavS3InfoDialog {...props} />
                </Provider>
            </MuiThemeProvider>
        );

        // then
        expect(wrapper.text()).toContain("https://download.example.com/by_id/zzzzz-4zz18-b1f8tbldjrm8885");
    });

    it('render s3 tab with federated token', () => {
        store.getState().dialog[COLLECTION_WEBDAV_S3_DIALOG_NAME].data.activeTab = 2;
        // when
        const wrapper = mount(
            <MuiThemeProvider theme={CustomTheme}>
                <Provider store={store}>
                    <WebDavS3InfoDialog {...props} />
                </Provider>
            </MuiThemeProvider>
        );

        // then
        expect(wrapper.text()).toContain("Secret Keyv2_zzzzb-jjjjj-123123_xxxtokenxxx");
    });

    it('render s3 tab with local token', () => {
        store.getState().dialog[COLLECTION_WEBDAV_S3_DIALOG_NAME].data.activeTab = 2;
        store.getState().dialog[COLLECTION_WEBDAV_S3_DIALOG_NAME].data.token = "v2/zzzzz-jjjjj-123123/xxxtokenxxx";
        // when
        const wrapper = mount(
            <MuiThemeProvider theme={CustomTheme}>
                <Provider store={store}>
                    <WebDavS3InfoDialog {...props} />
                </Provider>
            </MuiThemeProvider>
        );

        // then
        expect(wrapper.text()).toContain("Access Keyzzzzz-jjjjj-123123Secret Keyxxxtokenxxx");
    });

    it('render cyberduck tab with wildcard DNS', () => {
        store.getState().dialog[COLLECTION_WEBDAV_S3_DIALOG_NAME].data.activeTab = 0;
        store.getState().dialog[COLLECTION_WEBDAV_S3_DIALOG_NAME].data.collectionsUrl = "https://*.collections.example.com";
        // when
        const wrapper = mount(
            <MuiThemeProvider theme={CustomTheme}>
                <Provider store={store}>
                    <WebDavS3InfoDialog {...props} />
                </Provider>
            </MuiThemeProvider>
        );

        // then
        expect(wrapper.text()).toContain("davs://bobby@zzzzz-4zz18-b1f8tbldjrm8885.collections.example.com");
    });

});
