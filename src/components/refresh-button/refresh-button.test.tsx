// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { Button } from "@material-ui/core";
import { shallow, configure } from "enzyme";
import Adapter from "enzyme-adapter-react-16";
import { RefreshButton } from './refresh-button';

configure({ adapter: new Adapter() });

describe('<RefreshButton />', () => {
    let props;

    beforeEach(() => {
        props = {
            history: {
                replace: jest.fn(),
            },
            classes: {},
        };
    });

    it('should render without issues', () => {
        // when
        const wrapper = shallow(<RefreshButton {...props} />);

        // then
        expect(wrapper.html()).toContain('button');
    });

    it('should pass window location to router', () => {
        // setup
        const wrapper = shallow(<RefreshButton {...props} />);

        // when
        wrapper.find(Button).simulate('click');

        // then
        expect(props.history.replace).toHaveBeenCalledWith('/');
    });
});
