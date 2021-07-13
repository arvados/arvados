// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { shallow, configure } from 'enzyme';
import Adapter from 'enzyme-adapter-react-16';
import { FileViewerAction } from './file-viewer-action';

configure({ adapter: new Adapter() });

describe('FileViewerAction', () => {
    let props;

    beforeEach(() => {
        props = {
            onClick: jest.fn(),
            href: 'https://collections.example.com/c=zzzzz-4zz18-k0hamvtwyit6q56/t=xxxxxxx/LIMS/1.html',
        };
    });

    it('should render properly and handle click', () => {
        // when
        const wrapper = shallow(<FileViewerAction {...props} />);
        wrapper.find('a').simulate('click');

        // then
        expect(wrapper).not.toBeUndefined();

        // and
        expect(props.onClick).toHaveBeenCalled();
    });
});