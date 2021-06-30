// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { configure, mount } from "enzyme";
import { FileThumbnail } from "./file-thumbnail";
import { CollectionFileType } from '../../models/collection-file';
import Adapter from 'enzyme-adapter-react-16';
import { Provider } from "react-redux";
import { combineReducers, createStore } from "redux";

configure({ adapter: new Adapter() });

jest.mock('is-image', () => ({
    'default': () => true,
}));

let store;

describe("<FileThumbnail />", () => {
    let file;

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

        file = {
            name: 'test-image',
            type: CollectionFileType.FILE,
            url: 'http://example.com/c=zzzzz-4zz18-0123456789abcde/t=v2/zzzzz-gj3su-0123456789abcde/xxxxxxtokenxxxxx/test-image.jpg',
            size: 300
        };
    });

    it("renders file thumbnail with proper src", () => {
        const fileThumbnail = mount(<Provider store={store}><FileThumbnail file={file} /></Provider>);
        expect(fileThumbnail.html()).toBe('<img class="Component-thumbnail-1" alt="test-image" src="http://zzzzz-4zz18-0123456789abcde.collections.example.com/test-image.jpg?api_token=v2/zzzzz-gj3su-0123456789abcde/xxxxxxtokenxxxxx">');
    });
});
