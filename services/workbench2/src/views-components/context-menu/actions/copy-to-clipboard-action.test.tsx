// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { shallow, configure } from 'enzyme';
import { ListItem } from "@material-ui/core";
import Adapter from 'enzyme-adapter-react-16';
import { CopyToClipboardAction } from './copy-to-clipboard-action';

configure({ adapter: new Adapter() });

jest.mock('copy-to-clipboard', () => jest.fn());

describe('CopyToClipboardAction', () => {
    let props;

    beforeEach(() => {
        props = {
            onClick: jest.fn(),
            href: 'https://collections.example.com/c=zzzzz-4zz18-k0hamvtwyit6q56/t=xxxxxxxx/LIMS/1.html',
        };
    });

    it('should render properly and handle click', () => {
        // when
        const wrapper = shallow(<CopyToClipboardAction {...props} />);
        wrapper.find(ListItem).simulate('click');

        // then
        expect(wrapper).not.toBeUndefined();

        // and
        expect(props.onClick).toHaveBeenCalled();
    });
});