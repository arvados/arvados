// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { mount } from 'enzyme';
import * as Enzyme from 'enzyme';
import * as Adapter from 'enzyme-adapter-react-16';
import ListItemIcon from '@material-ui/core/ListItemIcon';
import { Collapse } from '@material-ui/core';

import ProjectTree from './project-tree';
import { TreeItem } from '../tree/tree';
import { Project } from '../../models/project';
Enzyme.configure({ adapter: new Adapter() });

describe("ProjectTree component", () => {

    it("checks is there ListItemIcon in the ProjectTree component", () => {
        const project: TreeItem<Project> = {
            data: {
                name: "sample name",
                createdAt: "2018-06-12",
                modifiedAt: "2018-06-13",
                uuid: "uuid",
                ownerUuid: "ownerUuid",
                href: "href",
            },
            id: "3",
            open: true,
            active: true
        };
        const wrapper = mount(<ProjectTree projects={[project]} toggleProjectTreeItem={() => { }} />);

        expect(wrapper.find(ListItemIcon).length).toEqual(1);
    });

    it("checks are there two ListItemIcon's in the ProjectTree component", () => {
        const project: Array<TreeItem<Project>> = [
            {
                data: {
                    name: "sample name",
                    createdAt: "2018-06-12",
                    modifiedAt: "2018-06-13",
                    uuid: "uuid",
                    ownerUuid: "ownerUuid",
                    href: "href",
                },
                id: "3",
                open: false,
                active: true
            },
            {
                data: {
                    name: "sample name",
                    createdAt: "2018-06-12",
                    modifiedAt: "2018-06-13",
                    uuid: "uuid",
                    ownerUuid: "ownerUuid",
                    href: "href",
                },
                id: "3",
                open: false,
                active: true
            }
        ];
        const wrapper = mount(<ProjectTree projects={project} toggleProjectTreeItem={() => { }} />);

        expect(wrapper.find(ListItemIcon).length).toEqual(2);
    });

    it("check ProjectTree, when open is changed", () => {
        const project: TreeItem<Project> = {
            data: {
                name: "sample name",
                createdAt: "2018-06-12",
                modifiedAt: "2018-06-13",
                uuid: "uuid",
                ownerUuid: "ownerUuid",
                href: "href",
            },
            id: "3",
            open: true,
            active: true,
            items: [
                {
                    data: {
                        name: "sample name",
                        createdAt: "2018-06-12",
                        modifiedAt: "2018-06-13",
                        uuid: "uuid",
                        ownerUuid: "ownerUuid",
                        href: "href",
                    },
                    id: "4",
                    open: false,
                    active: true
                }
            ]
        };
        const wrapper = mount(<ProjectTree projects={[project]} toggleProjectTreeItem={() => { }} />);
        wrapper.setState({open: true });

        expect(wrapper.find(Collapse).length).toEqual(1);
    });
});
