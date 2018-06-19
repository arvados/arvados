// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { mount, configure, shallow } from "enzyme";
import * as Adapter from "enzyme-adapter-react-16";
import { ContextMenu } from "./context-menu";
import { ListItem } from "@material-ui/core";

configure({ adapter: new Adapter() });

describe("<ContextMenu />", () => {

    const item = {
        name: "",
        owner: "",
        lastModified: "",
        type: ""
    };

    const actions = {
        onAddToFavourite: jest.fn(),
        onCopy: jest.fn(),
        onDownload: jest.fn(),
        onMoveTo: jest.fn(),
        onRemove: jest.fn(),
        onRename: jest.fn(),
        onShare: jest.fn()
    };

    it("calls provided actions with provided item", () => {
        const contextMenu = mount(<ContextMenu
            anchorEl={document.createElement("div")}
            onClose={jest.fn()}
            {...{ actions, item }} />);

        for (let index = 0; index < Object.keys(actions).length; index++) {
            contextMenu.find(ListItem).at(index).simulate("click");
        }

        Object.keys(actions).forEach(key => {
            expect(actions[key]).toHaveBeenCalledWith(item);
        });
    });
});