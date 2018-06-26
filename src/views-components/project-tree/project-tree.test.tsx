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

import ProjectTree from './project-tree';
import { TreeItem } from '../../components/tree/tree';
import { Project } from '../../models/project';
import { ResourceKind } from "../../models/resource";

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
                kind: ResourceKind.PROJECT
            },
            id: "3",
            open: true,
            active: true,
            status: 1
        };
        const wrapper = mount(<ProjectTree projects={[project]} toggleOpen={jest.fn()} toggleActive={jest.fn()} />);

        expect(wrapper.find(ListItemIcon)).toHaveLength(1);
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
                    kind: ResourceKind.PROJECT
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
                    kind: ResourceKind.PROJECT
                },
                id: "3",
                open: false,
                active: true,
                status: 1
            }
        ];
        const wrapper = mount(<ProjectTree projects={project} toggleOpen={jest.fn()} toggleActive={jest.fn()} />);

        expect(wrapper.find(ListItemIcon)).toHaveLength(2);
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
                    kind: ResourceKind.PROJECT
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
                            kind: ResourceKind.PROJECT
                        },
                        id: "3",
                        open: true,
                        active: true,
                        status: 1
                    }
                ]
            }
        ];
        const wrapper = mount(<ProjectTree projects={project} toggleOpen={jest.fn()} toggleActive={jest.fn()}/>);

        expect(wrapper.find(Collapse)).toHaveLength(1);
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
                kind: ResourceKind.PROJECT
            },
            id: "3",
            open: false,
            active: true,
            status: 1
        };
        const wrapper = mount(<ProjectTree projects={[project]} toggleOpen={jest.fn()} toggleActive={jest.fn()} />);

        expect(wrapper.find(CircularProgress)).toHaveLength(1);
    });
});
