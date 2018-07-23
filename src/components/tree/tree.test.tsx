// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0
import * as React from 'react';
import { mount } from 'enzyme';
import * as Enzyme from 'enzyme';
import * as Adapter from 'enzyme-adapter-react-16';
import ListItem from "@material-ui/core/ListItem/ListItem";

import { Tree, TreeItem } from './tree';
import { ProjectResource } from '../../models/project';
import { mockProjectResource } from '../../models/test-utils';

Enzyme.configure({ adapter: new Adapter() });

describe("Tree component", () => {

    it("should render ListItem", () => {
        const project: TreeItem<ProjectResource> = {
            data: mockProjectResource(),
            id: "3",
            open: true,
            active: true,
            status: 1,
        };
        const wrapper = mount(<Tree
            render={project => <div />}
            toggleItemOpen={jest.fn()}
            toggleItemActive={jest.fn()}
            onContextMenu={jest.fn()}
            items={[project]} />);
        expect(wrapper.find(ListItem)).toHaveLength(1);
    });

    it("should render arrow", () => {
        const project: TreeItem<ProjectResource> = {
            data: mockProjectResource(),
            id: "3",
            open: true,
            active: true,
            status: 1,
        };
        const wrapper = mount(<Tree
            render={project => <div />}
            toggleItemOpen={jest.fn()}
            toggleItemActive={jest.fn()}
            onContextMenu={jest.fn()}
            items={[project]} />);
        expect(wrapper.find('i')).toHaveLength(1);
    });
});
