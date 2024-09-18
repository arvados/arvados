// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0
import React from 'react';
import { Tree, TreeItemStatus } from './tree';
import { mockProjectResource } from '../../models/test-utils';
import { ThemeProvider } from '@mui/material/styles';
import { CustomTheme } from '../../common/custom-theme';

describe("Tree component", () => {

    it("should render ListItem", () => {
        const project = {
            data: mockProjectResource(),
            id: "3",
            open: true,
            active: true,
            status: TreeItemStatus.LOADED
        };
        cy.mount(
            <ThemeProvider theme={CustomTheme}>
                <Tree
                render={project => <div />}
                toggleItemOpen={cy.stub()}
                toggleItemActive={cy.stub()}
                onContextMenu={cy.stub()}
                items={[project]} />
            </ThemeProvider>
        );
        cy.get('[data-cy=tree-li]').should('have.length', 1);
    });

    it("should render arrow", () => {
        const project = {
            data: mockProjectResource(),
            id: "3",
            open: true,
            active: true,
            status: TreeItemStatus.LOADED,
        };
        cy.mount(
            <ThemeProvider theme={CustomTheme}>
                <Tree
                render={project => <div />}
                toggleItemOpen={cy.stub()}
                toggleItemActive={cy.stub()}
                onContextMenu={cy.stub()}
                items={[project]} />
            </ThemeProvider>
        );
        cy.get('i').should('have.length', 1);
    });

    it("should render checkbox", () => {
        const project = {
            data: mockProjectResource(),
            id: "3",
            open: true,
            active: true,
            status: TreeItemStatus.LOADED
        };
        cy.mount(
            <ThemeProvider theme={CustomTheme}>
                <Tree
                    showSelection={true}
                    render={() => <div />}
                    toggleItemOpen={cy.stub()}
                    toggleItemActive={cy.stub()}
                    onContextMenu={cy.stub()}
                    items={[project]} />
            </ThemeProvider>
        );
        cy.get('input[type=checkbox]').should('have.length', 1);
    });

    it("call onSelectionChanged with associated item", () => {
        const project = {
            data: mockProjectResource(),
            id: "3",
            open: true,
            active: true,
            status: TreeItemStatus.LOADED,
        };
        const spy = cy.spy().as('spy');
        const onSelectionChanged = (event, item) => spy(item);
        cy.mount(
            <ThemeProvider theme={CustomTheme}>
                <Tree
                showSelection={true}
                render={() => <div />}
                toggleItemOpen={cy.stub()}
                toggleItemActive={cy.stub()}
                onContextMenu={cy.stub()}
                toggleItemSelection={onSelectionChanged}
                items={[project]} />
            </ThemeProvider>
            );
        cy.get('input[type=checkbox]').click();
        cy.get('@spy').should('have.been.calledWith', project);
    });

});
