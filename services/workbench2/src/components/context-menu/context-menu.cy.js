// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { ContextMenu } from "./context-menu";
import { ShareIcon } from "../icon/icon";

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
        const onItemClick = cy.spy().as("onItemClick")
        cy.mount(<ContextMenu
            anchorEl={document.createElement("div")}
            open={true}
            onClose={cy.stub()}
            onItemClick={onItemClick}
            items={items} />);
        cy.get('div[role=button]').eq(2).click();
        cy.get('@onItemClick').should('have.been.calledWith', items[1][0]);
    });
});
