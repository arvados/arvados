// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { configure, mount } from "enzyme";
import Adapter from "enzyme-adapter-react-16";
import { MPVContainer } from './multi-panel-view';
import { Button } from "@material-ui/core";

configure({ adapter: new Adapter() });

const PanelMock = ({panelName, panelMaximized, doHidePanel, doMaximizePanel, children, ...rest}) =>
    <div {...rest}>{children}</div>;

describe('<MPVContainer />', () => {
    let props;

    beforeEach(() => {
        props = {
            classes: {},
        };
    });

    it('should show default panel buttons for every child', () => {
        const childs = [
            <PanelMock key={1}>This is one panel</PanelMock>,
            <PanelMock key={2}>This is another panel</PanelMock>,
        ];
        const wrapper = mount(<MPVContainer {...props}>{[...childs]}</MPVContainer>);
        expect(wrapper.find(Button).first().html()).toContain('Panel 1');
        expect(wrapper.html()).toContain('This is one panel');
        expect(wrapper.find(Button).last().html()).toContain('Panel 2');
        expect(wrapper.html()).toContain('This is another panel');
    });

    it('should show panel when clicking on its button', () => {
        const childs = [
            <PanelMock key={1}>This is one panel</PanelMock>,
        ];
        props.panelStates = [
            {name: 'Initially invisible Panel', visible: false},
        ]

        const wrapper = mount(<MPVContainer {...props}>{[...childs]}</MPVContainer>);

        // Initial state: panel not visible
        expect(wrapper.html()).not.toContain('This is one panel');
        expect(wrapper.html()).toContain('All panels are hidden');

        // Panel visible when clicking on its button
        wrapper.find(Button).simulate('click');
        expect(wrapper.html()).toContain('This is one panel');
        expect(wrapper.html()).not.toContain('All panels are hidden');
    });

    it('should show custom panel buttons when config provided', () => {
        const childs = [
            <PanelMock key={1}>This is one panel</PanelMock>,
            <PanelMock key={2}>This is another panel</PanelMock>,
        ];
        props.panelStates = [
            {name: 'First Panel'},
        ]
        const wrapper = mount(<MPVContainer {...props}>{[...childs]}</MPVContainer>);
        expect(wrapper.find(Button).first().html()).toContain('First Panel');
        expect(wrapper.html()).toContain('This is one panel');
        // Second panel received the default button naming and hidden status by default
        expect(wrapper.find(Button).last().html()).toContain('Panel 2');
        expect(wrapper.html()).not.toContain('This is another panel');
        wrapper.find(Button).last().simulate('click');
        expect(wrapper.html()).toContain('This is another panel');
    });

    it('should set panel hidden when requested', () => {
        const childs = [
            <PanelMock key={1}>This is one panel</PanelMock>,
        ];
        props.panelStates = [
            {name: 'First Panel', visible: false},
        ]
        const wrapper = mount(<MPVContainer {...props}>{[...childs]}</MPVContainer>);
        expect(wrapper.find(Button).html()).toContain('First Panel');
        expect(wrapper.html()).not.toContain('This is one panel');
        expect(wrapper.html()).toContain('All panels are hidden');
    });
});