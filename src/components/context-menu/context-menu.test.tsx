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
    const actions = [[{
        icon: "",
        name: "Action 1.1"
    }, {
        icon: "",
        name: "Action 1.2"
    },], [{
        icon: "",
        name: "Action 2.1"
    }]];

    it("calls onActionClick with clicked action", () => {
        const onActionClick = jest.fn();
        const contextMenu = mount(<ContextMenu
            anchorEl={document.createElement("div")}
            onClose={jest.fn()}
            onActionClick={onActionClick}
            actions={actions} />);
        contextMenu.find(ListItem).at(2).simulate("click");
        expect(onActionClick).toHaveBeenCalledWith(actions[1][0]);
    });
});