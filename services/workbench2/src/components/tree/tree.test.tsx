// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0
import React from 'react';
import { mount } from 'enzyme';
import * as Enzyme from 'enzyme';
import Adapter from 'enzyme-adapter-react-16';
import ListItem from "@material-ui/core/ListItem/ListItem";

import { Tree, TreeItem, TreeItemStatus } from './tree';
import { ProjectResource } from '../../models/project';
import { mockProjectResource } from '../../models/test-utils';
import { Checkbox } from '@material-ui/core';

Enzyme.configure({ adapter: new Adapter() });

describe("Tree component", () => {

    it("should render ListItem", () => {
        const project: TreeItem<ProjectResource> = {
            data: mockProjectResource(),
            id: "3",
            open: true,
            active: true,
            status: TreeItemStatus.LOADED
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
            status: TreeItemStatus.LOADED,
        };
        const wrapper = mount(<Tree
            render={project => <div />}
            toggleItemOpen={jest.fn()}
            toggleItemActive={jest.fn()}
            onContextMenu={jest.fn()}
            items={[project]} />);
        expect(wrapper.find('i')).toHaveLength(1);
    });

    it("should render checkbox", () => {
        const project: TreeItem<ProjectResource> = {
            data: mockProjectResource(),
            id: "3",
            open: true,
            active: true,
            status: TreeItemStatus.LOADED
        };
        const wrapper = mount(<Tree
            showSelection={true}
            render={() => <div />}
            toggleItemOpen={jest.fn()}
            toggleItemActive={jest.fn()}
            onContextMenu={jest.fn()}
            items={[project]} />);
        expect(wrapper.find(Checkbox)).toHaveLength(1);
    });

    it("call onSelectionChanged with associated item", () => {
        const project: TreeItem<ProjectResource> = {
            data: mockProjectResource(),
            id: "3",
            open: true,
            active: true,
            status: TreeItemStatus.LOADED,
        };
        const spy = jest.fn();
        const onSelectionChanged = (event: any, item: TreeItem<any>) => spy(item);
        const wrapper = mount(<Tree
            showSelection={true}
            render={() => <div />}
            toggleItemOpen={jest.fn()}
            toggleItemActive={jest.fn()}
            onContextMenu={jest.fn()}
            toggleItemSelection={onSelectionChanged}
            items={[project]} />);
        wrapper.find(Checkbox).simulate('click');
        expect(spy).toHaveBeenLastCalledWith({
            data: mockProjectResource(),
            id: "3",
            open: true,
            active: true,
            status: TreeItemStatus.LOADED,
        });
    });

});
