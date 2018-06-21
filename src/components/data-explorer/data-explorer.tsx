// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { Grid, Paper, Toolbar } from '@material-ui/core';
import ContextMenu, { ContextMenuActionGroup, ContextMenuAction } from "../../components/context-menu/context-menu";
import ColumnSelector from "../../components/column-selector/column-selector";
import DataTable from "../../components/data-table/data-table";
import { mockAnchorFromMouseEvent } from "../../components/popover/helpers";
import { DataColumn, toggleSortDirection } from "../../components/data-table/data-column";
import { DataTableFilterItem } from '../../components/data-table-filters/data-table-filters';

interface DataExplorerProps<T> {
    items: T[];
    columns: Array<DataColumn<T>>;
    contextActions: ContextMenuActionGroup[];
    onRowClick: (item: T) => void;
    onColumnToggle: (column: DataColumn<T>) => void;
    onContextAction: (action: ContextMenuAction, item: T) => void;
    onSortToggle: (column: DataColumn<T>) => void;
    onFiltersChange: (filters: DataTableFilterItem[], column: DataColumn<T>) => void;
}

interface DataExplorerState<T> {
    contextMenu: {
        anchorEl?: HTMLElement;
        item?: T;
    };
}

class DataExplorer<T> extends React.Component<DataExplorerProps<T>, DataExplorerState<T>> {
    state: DataExplorerState<T> = {
        contextMenu: {}
    };

    render() {
        return <Paper>
            <ContextMenu
                anchorEl={this.state.contextMenu.anchorEl}
                actions={this.props.contextActions}
                onActionClick={this.callAction}
                onClose={this.closeContextMenu} />
            <Toolbar>
                <Grid container justify="flex-end">
                    <ColumnSelector
                        columns={this.props.columns}
                        onColumnToggle={this.props.onColumnToggle} />
                </Grid>
            </Toolbar>
            <DataTable
                columns={this.props.columns}
                items={this.props.items}
                onRowClick={(_, item: T) => this.props.onRowClick(item)}
                onRowContextMenu={this.openContextMenu}
                onFiltersChange={this.props.onFiltersChange}
                onSortToggle={this.props.onSortToggle} />
            <Toolbar />
        </Paper>;
    }

    openContextMenu = (event: React.MouseEvent<HTMLElement>, item: T) => {
        event.preventDefault();
        this.setState({
            contextMenu: {
                anchorEl: mockAnchorFromMouseEvent(event),
                item
            }
        });
    }

    closeContextMenu = () => {
        this.setState({ contextMenu: {} });
    }

    callAction = (action: ContextMenuAction) => {
        const { item } = this.state.contextMenu;
        this.closeContextMenu();
        if (item) {
            this.props.onContextAction(action, item);
        }
    }

}

export default DataExplorer;
