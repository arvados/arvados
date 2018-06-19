// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { mount, configure, shallow } from "enzyme";
import * as Adapter from "enzyme-adapter-react-16";
import ContextMenu from "./context-menu";
import { ListItem } from "@material-ui/core";

configure({ adapter: new Adapter() });

describe("<ContextMenu />", () => {

    const item = {
        name: "",
        owner: "",
        lastModified: "",
        type: ""
    };

    const actions = [[{
        icon: "",
        name: "Action 1.1",
        onClick: jest.fn()
    },
    {
        icon: "",
        name: "Action 1.2",
        onClick: jest.fn()
    },], [{
        icon: "",
        name: "Action 2.1",
        onClick: jest.fn()
    }]];

    it("calls provided actions with provided item", () => {
        const contextMenu = mount(<ContextMenu
            anchorEl={document.createElement("div")}
            onClose={jest.fn()}
            {...{ actions, item }} />);

        contextMenu.find(ListItem).at(0).simulate("click");
        contextMenu.find(ListItem).at(1).simulate("click");
        contextMenu.find(ListItem).at(2).simulate("click");

        expect(actions[0][0].onClick).toHaveBeenCalledWith(item);
        expect(actions[0][1].onClick).toHaveBeenCalledWith(item);
        expect(actions[1][0].onClick).toHaveBeenCalledWith(item);
    });
});