// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { mount, configure } from 'enzyme';
import Adapter from 'enzyme-adapter-react-16';

import {
    SharingURLsComponent,
    SharingURLsComponentProps
} from './sharing-urls-component';

configure({ adapter: new Adapter() });

describe("<SharingURLsComponent />", () => {
    let props: SharingURLsComponentProps;
    let wrapper;

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
            onCopy: jest.fn(),
            onDeleteSharingToken: jest.fn(),
        };
        wrapper = mount(<SharingURLsComponent {...props} />);
    });

    it("renders a list of sharing URLs", () => {
        expect(wrapper.find('a').length).toBe(2);
        // Check 1st URL
        expect(wrapper.find('a').at(0).text()).toContain(`Token aaaaaaaa... expiring at: ${new Date(props.sharingTokens[0].expiresAt).toLocaleString()}`);
        expect(wrapper.find('a').at(0).props().href).toBe(`${props.sharingURLsPrefix}/c=${props.collectionUuid}/t=${props.sharingTokens[0].apiToken}/_/`);
        // Check 2nd URL
        expect(wrapper.find('a').at(1).text()).toContain(`Token bbbbbbbb... expiring at: ${new Date(props.sharingTokens[1].expiresAt).toLocaleString()}`);
        expect(wrapper.find('a').at(1).props().href).toBe(`${props.sharingURLsPrefix}/c=${props.collectionUuid}/t=${props.sharingTokens[1].apiToken}/_/`);
    });

    it("renders a list URLs with collection UUIDs as subdomains", () => {
        props.sharingURLsPrefix = '*.sharing-urls-prefix';
        const sharingPrefix = '.sharing-urls-prefix';
        wrapper = mount(<SharingURLsComponent {...props} />);
        expect(wrapper.find('a').at(0).props().href).toBe(`${props.collectionUuid}${sharingPrefix}/t=${props.sharingTokens[0].apiToken}/_/`);
        expect(wrapper.find('a').at(1).props().href).toBe(`${props.collectionUuid}${sharingPrefix}/t=${props.sharingTokens[1].apiToken}/_/`);
    });

    it("calls delete token handler when delete button is clicked", () => {
        wrapper.find('button').at(0).simulate('click');
        expect(props.onDeleteSharingToken).toHaveBeenCalledWith(props.sharingTokens[0].uuid);
    });
});