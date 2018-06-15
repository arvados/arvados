// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { mount } from 'enzyme';
import * as Enzyme from 'enzyme';
import * as Adapter from 'enzyme-adapter-react-16';
import ListItemIcon from '@material-ui/core/ListItemIcon';
import { Collapse } from '@material-ui/core';
import CircularProgress from '@material-ui/core/CircularProgress';

import ProjectTree from './project-tree';
import { TreeItem } from '../tree/tree';
import { Project } from '../../models/project';
Enzyme.configure({ adapter: new Adapter() });

describe("ProjectTree component", () => {

    it("should render ListItemIcon", () => {
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
            status: 1
        };
        const wrapper = mount(<ProjectTree projects={[project]} toggleProjectTreeItem={() => { }} />);

        expect(wrapper.find(ListItemIcon).length).toEqual(1);
    });

    it("should render 2 ListItemIcons", () => {
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
                active: true,
                status: 1
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
                active: true,
                status: 1
            }
        ];
        const wrapper = mount(<ProjectTree projects={project} toggleProjectTreeItem={() => { }} />);

        expect(wrapper.find(ListItemIcon).length).toEqual(2);
    });

    it("should render Collapse", () => {
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
                open: true,
                active: true,
                status: 2,
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
                        id: "3",
                        open: true,
                        active: true,
                        status: 1
                    }
                ]
            }
        ];
        const wrapper = mount(<ProjectTree projects={project} toggleProjectTreeItem={() => { }} />);

        expect(wrapper.find(Collapse).length).toEqual(1);
    });

    it("should render CircularProgress", () => {
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
            open: false,
            active: true,
            status: 1
        };
        const wrapper = mount(<ProjectTree projects={[project]} toggleProjectTreeItem={() => { }} />);

        expect(wrapper.find(CircularProgress).length).toEqual(1);
    });
});
