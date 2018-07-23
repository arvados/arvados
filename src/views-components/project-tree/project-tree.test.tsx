// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import * as Enzyme from 'enzyme';
import { mount } from 'enzyme';
import * as Adapter from 'enzyme-adapter-react-16';
import ListItemIcon from '@material-ui/core/ListItemIcon';
import { Collapse } from '@material-ui/core';
import CircularProgress from '@material-ui/core/CircularProgress';

import { ProjectTree } from './project-tree';
import { TreeItem } from '../../components/tree/tree';
import { ProjectResource } from '../../models/project';
import { mockProjectResource } from '../../models/test-utils';

Enzyme.configure({ adapter: new Adapter() });

describe("ProjectTree component", () => {

    it("should render ListItemIcon", () => {
        const project: TreeItem<ProjectResource> = {
            data: mockProjectResource(),
            id: "3",
            open: true,
            active: true,
            status: 1
        };
        const wrapper = mount(<ProjectTree
            projects={[project]}
            toggleOpen={jest.fn()}
            toggleActive={jest.fn()}
            onContextMenu={jest.fn()} />);

        expect(wrapper.find(ListItemIcon)).toHaveLength(1);
    });

    it("should render 2 ListItemIcons", () => {
        const project: Array<TreeItem<ProjectResource>> = [
            {
                data: mockProjectResource(),
                id: "3",
                open: false,
                active: true,
                status: 1
            },
            {
                data: mockProjectResource(),
                id: "3",
                open: false,
                active: true,
                status: 1
            }
        ];
        const wrapper = mount(<ProjectTree
            projects={project}
            toggleOpen={jest.fn()}
            toggleActive={jest.fn()}
            onContextMenu={jest.fn()} />);

        expect(wrapper.find(ListItemIcon)).toHaveLength(2);
    });

    it("should render Collapse", () => {
        const project: Array<TreeItem<ProjectResource>> = [
            {
                data: mockProjectResource(),
                id: "3",
                open: true,
                active: true,
                status: 2,
                items: [
                    {
                        data: mockProjectResource(),
                        id: "3",
                        open: true,
                        active: true,
                        status: 1
                    }
                ]
            }
        ];
        const wrapper = mount(<ProjectTree
            projects={project}
            toggleOpen={jest.fn()}
            toggleActive={jest.fn()}
            onContextMenu={jest.fn()} />);

        expect(wrapper.find(Collapse)).toHaveLength(1);
    });

    it("should render CircularProgress", () => {
        const project: TreeItem<ProjectResource> = {
            data: mockProjectResource(),
            id: "3",
            open: false,
            active: true,
            status: 1
        };
        const wrapper = mount(<ProjectTree
            projects={[project]}
            toggleOpen={jest.fn()}
            toggleActive={jest.fn()}
            onContextMenu={jest.fn()} />);

        expect(wrapper.find(CircularProgress)).toHaveLength(1);
    });
});
