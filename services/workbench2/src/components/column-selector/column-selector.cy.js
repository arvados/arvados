// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { ColumnSelector } from "./column-selector";

describe("<ColumnSelector />", () => {
    it("shows only configurable columns", () => {
        const columns = [
            {
                name: "Column 1",
                render: () => <span />,
                selected: true,
                configurable: true
            },
            {
                name: "Column 2",
                render: () => <span />,
                selected: true,
                configurable: true,
            },
            {
                name: "Column 3",
                render: () => <span />,
                selected: true,
                configurable: false
            }
        ];
        cy.mount(<ColumnSelector columns={columns} onColumnToggle={cy.stub()} />);
        cy.get('button[aria-label="Select columns"]').click();
        cy.get('[data-cy=column-selector-li]').should('have.length', 2);
    });

    it("renders checked checkboxes next to selected columns", () => {
        const columns = [
            {
                name: "Column 1",
                render: () => <span />,
                selected: true,
                configurable: true
            },
            {
                name: "Column 2",
                render: () => <span />,
                selected: false,
                configurable: true
            },
            {
                name: "Column 3",
                render: () => <span />,
                selected: true,
                configurable: true
            }
        ];
        cy.mount(<ColumnSelector columns={columns} onColumnToggle={cy.stub()} />);
        cy.get('button[aria-label="Select columns"]').click();
        cy.get('input[type=checkbox]').should('have.length', 3);
        cy.get('input[type=checkbox]').eq(0).should('be.checked');
        cy.get('input[type=checkbox]').eq(1).should('not.be.checked');
        cy.get('input[type=checkbox]').eq(2).should('be.checked');
    });

    it("calls onColumnToggle with clicked column", () => {
        const columns = [
            {
                name: "Column 1",
                render: () => <span />,
                selected: true,
                configurable: true
            }
        ];
        const onColumnToggle = cy.spy().as("onColumnToggle");
        cy.mount(<ColumnSelector columns={columns} onColumnToggle={onColumnToggle} />);
        cy.get('button[aria-label="Select columns"]').click();
        cy.get('[data-cy=column-selector-li]').click();
        cy.get('@onColumnToggle').should('have.been.calledWith', columns[0]);
    });
});
