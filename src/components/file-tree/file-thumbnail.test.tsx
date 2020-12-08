// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { shallow, configure } from "enzyme";
import { FileThumbnail } from "./file-thumbnail";
import { CollectionFileType } from '../../models/collection-file';
import * as Adapter from 'enzyme-adapter-react-16';

configure({ adapter: new Adapter() });

jest.mock('is-image', () => ({
    'default': () => true,
}));

describe("<FileThumbnail />", () => {
    let file;

    beforeEach(() => {
        file = {
            name: 'test-image',
            type: CollectionFileType.FILE,
            url: 'http://example.com/c=zzzzz-4zz18-0123456789abcde/t=v2/zzzzz-gj3su-0123456789abcde/xxxxxxtokenxxxxx/test-image.jpg',
            size: 300
        };
    });

    it("renders file thumbnail with proper src", () => {
        const fileThumbnail = shallow(<FileThumbnail file={file} />);
        expect(fileThumbnail.html()).toBe('<img class="Component-thumbnail-1" alt="test-image" src="http://example.com/c=zzzzz-4zz18-0123456789abcde/test-image.jpg?api_token=v2/zzzzz-gj3su-0123456789abcde/xxxxxxtokenxxxxx"/>');
    });
});
