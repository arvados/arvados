// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0
import * as React from 'react';
import { mount } from 'enzyme';
import * as Enzyme from 'enzyme';
import * as Adapter from 'enzyme-adapter-react-16';
import { Collapse } from '@material-ui/core';
import CircularProgress from '@material-ui/core/CircularProgress';
import ListItem from "@material-ui/core/ListItem/ListItem";

import Tree, {TreeItem} from './tree';
import { Project } from '../../models/project';
Enzyme.configure({ adapter: new Adapter() });

describe("ProjectTree component", () => {

	it("should render ListItem", () => {
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
			loading: true,
        };
		const wrapper = mount(<Tree render={project => <div/>} toggleItem={() => { }} items={[project]}/>)
		expect(wrapper.find(ListItem).length).toEqual(1);
	});
    
    it("should render arrow", () => {
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
			loading: true,
        };
		const wrapper = mount(<Tree render={project => <div/>} toggleItem={() => { }} items={[project]}/>)
		expect(wrapper.find('i').length).toEqual(1);
	});
});
