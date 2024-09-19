// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { DropdownMenu } from "./dropdown-menu";
import { MenuItem } from "@mui/material";
import { PaginationRightArrowIcon } from "../icon/icon";

describe("<DropdownMenu />", () => {
    it("renders menu icon", () => {
        cy.mount(<DropdownMenu id="test-menu" icon={<PaginationRightArrowIcon />} />);
        cy.get('[data-cy=dropdown-menu-button]').should('have.length', 1);
    });

    it("opens and closes", () => {
        cy.mount(<DropdownMenu id="test-menu" icon={<PaginationRightArrowIcon />} />);
        cy.get('[data-cy=dropdown-menu-button]').click();
        cy.get('ul[role=menu]').should('exist').click();
        cy.get('ul[role=menu]').should('not.exist');
    });

    it("render menu items", () => {
        cy.mount(
            <DropdownMenu id="test-menu" icon={<PaginationRightArrowIcon />}>
                <MenuItem>Item 1</MenuItem>
                <MenuItem>Item 2</MenuItem>
            </DropdownMenu>
        );
        cy.get('[data-cy=dropdown-menu-button]').click();
        cy.get('li[role=menuitem]').should('have.length', 2);
    });
});
