// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { configure, shallow } from "enzyme";

import Adapter from "enzyme-adapter-react-16";
import { Breadcrumbs } from "./breadcrumbs";
import { Button } from "@material-ui/core";
import ChevronRightIcon from '@material-ui/icons/ChevronRight';

configure({ adapter: new Adapter() });

describe("<Breadcrumbs />", () => {

    let onClick: () => void;
    let resources = {};

    beforeEach(() => {
        onClick = jest.fn();
    });

    it("renders one item", () => {
        const items = [
            { label: 'breadcrumb 1' }
        ];
        const breadcrumbs = shallow(<Breadcrumbs items={items} resources={resources} onClick={onClick} onContextMenu={jest.fn()} />).dive();
        expect(breadcrumbs.find(Button)).toHaveLength(1);
        expect(breadcrumbs.find(ChevronRightIcon)).toHaveLength(0);
    });

    it("renders multiple items", () => {
        const items = [
            { label: 'breadcrumb 1' },
            { label: 'breadcrumb 2' }
        ];
        const breadcrumbs = shallow(<Breadcrumbs items={items} resources={resources} onClick={onClick} onContextMenu={jest.fn()} />).dive();
        expect(breadcrumbs.find(Button)).toHaveLength(2);
        expect(breadcrumbs.find(ChevronRightIcon)).toHaveLength(1);
    });

    it("calls onClick with clicked item", () => {
        const items = [
            { label: 'breadcrumb 1' },
            { label: 'breadcrumb 2' }
        ];
        const breadcrumbs = shallow(<Breadcrumbs items={items} resources={resources} onClick={onClick} onContextMenu={jest.fn()} />).dive();
        breadcrumbs.find(Button).at(1).simulate('click');
        expect(onClick).toBeCalledWith(items[1]);
    });

});
