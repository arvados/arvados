// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { DataTableFiltersPopover } from "./data-table-filters-popover";
import { getInitialProcessStatusFilters } from "store/resource-type-filters/resource-type-filters"
import { ThemeProvider } from "@mui/material";
import { CustomTheme } from "common/custom-theme";

describe("<DataTableFiltersPopover />", () => {
    it("renders filters according to their state", () => {
        // 1st filter (All) is selected, the rest aren't.
        const filters = getInitialProcessStatusFilters()
        const columnFilterCount = {'All': 0, 'Draft': 1, 'On hold': 2, 'Queued': 3, 'Running': 4, 'Completed': 5, 'Cancelled': 6, 'Failed': 7}

        cy.mount(
            <ThemeProvider theme={CustomTheme}>
                <DataTableFiltersPopover name="" filters={filters} columnFilterCount={columnFilterCount} />
            </ThemeProvider>
        );
        cy.get('span[role=button]').eq(0).click();
        cy.get('input[type=checkbox]').should('have.length', 8);
        // check that each filter has the correct count
        Object.keys(columnFilterCount).forEach((key, idx) => {
            if (idx === 0) {
                cy.get('[data-cy=tree-li]').contains(key).should('contain', 'All')
            } else {
                cy.get('[data-cy=tree-li]').contains(key).should('contain', columnFilterCount[key])
            }
        })
        //"All" should be the only item selected
        cy.get('input[type=checkbox]').eq(0).should('be.checked');
        cy.get('input[type=checkbox]').eq(1).should('not.be.checked');
        cy.contains('Close').click();
        cy.get('input[type=checkbox]').should('not.exist');
    });
});
