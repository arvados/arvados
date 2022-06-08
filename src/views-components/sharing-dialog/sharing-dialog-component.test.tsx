// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { mount, configure } from 'enzyme';
import Adapter from 'enzyme-adapter-react-16';
import { Provider } from 'react-redux';
import { combineReducers, createStore } from 'redux';

import SharingDialogComponent, {
    SharingDialogComponentProps,
} from './sharing-dialog-component';
import {
    extractUuidObjectType,
    ResourceObjectType
} from 'models/resource';

configure({ adapter: new Adapter() });

describe("<SharingDialogComponent />", () => {
    let props: SharingDialogComponentProps;
    let store;

    beforeEach(() => {
        const initialAuthState = {
            config: {
                keepWebServiceUrl: 'http://example.com/',
                keepWebInlineServiceUrl: 'http://*.collections.example.com/',
            }
        }
        store = createStore(combineReducers({
            auth: (state: any = initialAuthState, action: any) => state,
        }));

        props = {
            open: true,
            loading: false,
            saveEnabled: false,
            sharedResourceUuid: 'zzzzz-4zz18-zzzzzzzzzzzzzzz',
            privateAccess: true,
            sharingURLsNr: 2,
            sharingURLsDisabled: false,
            onClose: jest.fn(),
            onSave: jest.fn(),
            onCreateSharingToken: jest.fn(),
            refreshPermissions: jest.fn(),
        };
    });

    it("show sharing urls tab on collections when not disabled", () => {
        expect(props.sharingURLsDisabled).toBe(false);
        expect(props.sharingURLsNr).toBe(2);
        expect(extractUuidObjectType(props.sharedResourceUuid) === ResourceObjectType.COLLECTION).toBe(true);
        let wrapper = mount(<Provider store={store}><SharingDialogComponent {...props} /></Provider>);
        expect(wrapper.html()).toContain('Sharing URLs (2)');

        // disable Sharing URLs UI
        props.sharingURLsDisabled = true;
        wrapper = mount(<Provider store={store}><SharingDialogComponent {...props} /></Provider>);
        expect(wrapper.html()).not.toContain('Sharing URLs');
    });

    it("does not show sharing urls on non-collection resources", () => {
        props.sharedResourceUuid = 'zzzzz-j7d0g-0123456789abcde';
        expect(extractUuidObjectType(props.sharedResourceUuid) === ResourceObjectType.COLLECTION).toBe(false);
        expect(props.sharingURLsDisabled).toBe(false);
        let wrapper = mount(<Provider store={store}><SharingDialogComponent {...props} /></Provider>);
        expect(wrapper.html()).not.toContain('Sharing URLs');
    });
});