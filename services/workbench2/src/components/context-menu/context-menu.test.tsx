// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { mount, configure } from "enzyme";
import Adapter from "enzyme-adapter-react-16";
import { ContextMenu } from "./context-menu";
import { ListItem } from "@material-ui/core";
import { ShareIcon } from "../icon/icon";

configure({ adapter: new Adapter() });

describe("<ContextMenu />", () => {
    const items = [[{
        icon: ShareIcon,
        name: "Action 1.1"
    }, {
        icon: ShareIcon,
        name: "Action 1.2"
    },], [{
        icon: ShareIcon,
        name: "Action 2.1"
    }]];

    it("calls onItemClick with clicked action", () => {
        const onItemClick = jest.fn();
        const contextMenu = mount(<ContextMenu
            anchorEl={document.createElement("div")}
            open={true}
            onClose={jest.fn()}
            onItemClick={onItemClick}
            items={items} />);
        contextMenu.find(ListItem).at(2).simulate("click");
        expect(onItemClick).toHaveBeenCalledWith(items[1][0]);
    });
});
