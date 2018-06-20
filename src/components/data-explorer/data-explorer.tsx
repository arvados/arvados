// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { Grid, Paper, Toolbar } from '@material-ui/core';
import ContextMenu, { ContextMenuActionGroup } from "../../components/context-menu/context-menu";
import ColumnSelector from "../../components/column-selector/column-selector";
import DataTable from "../../components/data-table/data-table";
import { mockAnchorFromMouseEvent } from "../../components/popover/helpers";
import { DataColumn, toggleSortDirection } from "../../components/data-table/data-column";
import { DataTableFilterItem } from '../../components/data-table-filters/data-table-filters';
import { DataExplorerColumn } from './data-explorer-column';

interface DataExplorerProps<T> {
    items: T[];
    columns: Array<DataExplorerColumn<T>>;
    contextActions: Array<ContextMenuActionGroup<T>>;
    onRowClick: (item: T) => void;
    onColumnToggle: (column: DataExplorerColumn<T>) => void;
    onSortingToggle: (column: DataExplorerColumn<T>) => void;
    onFiltersChange: (columns: DataExplorerColumn<T>) => void;
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
                {...this.state.contextMenu}
                actions={this.contextActions}
                onClose={this.closeContextMenu} />
            <Toolbar>
                <Grid container justify="flex-end">
                    <ColumnSelector
                        columns={this.columns}
                        onColumnToggle={this.toggleColumn} />
                </Grid>
            </Toolbar>
            <DataTable
                columns={this.columns}
                items={this.props.items}
                onRowClick={(_, row: T) => this.props.onRowClick(row)}
                onRowContextMenu={this.openItemMenuOnRowClick} />
            <Toolbar />
        </Paper>;
    }

    get columns(): Array<DataColumn<T>> {
        return this.props.columns.map((column): DataColumn<T> => ({
            configurable: column.configurable,
            filters: column.filters,
            name: column.name,
            onFiltersChange: column.filterable ? this.changeFilters(column) : undefined,
            onSortToggle: column.sortable ? this.toggleSort(column) : undefined,
            render: column.render,
            renderHeader: column.renderHeader,
            selected: column.selected,
            sortDirection: column.sortDirection
        }));
    }

    get contextActions() {
        return this.props.contextActions.map(actionGroup =>
            actionGroup.map(action => ({
                ...action,
                onClick: (item: T) => {
                    this.closeContextMenu();
                    action.onClick(item);
                }
            })));
    }

    toggleColumn = (column: DataExplorerColumn<T>) => {
        this.props.onColumnToggle(column);
    }

    toggleSort = (column: DataExplorerColumn<T>) => () => {
        this.props.onSortingToggle(toggleSortDirection(column));
    }

    changeFilters = (column: DataExplorerColumn<T>) => (filters: DataTableFilterItem[]) => {
        this.props.onFiltersChange({ ...column, filters });
    }

    openItemMenuOnRowClick = (event: React.MouseEvent<HTMLElement>, item: T) => {
        event.preventDefault();
        this.setState({
            contextMenu: {
                anchorEl: mockAnchorFromMouseEvent(event),
                item
            }
        });
    }

    openItemMenuOnActionsClick = (event: React.MouseEvent<HTMLElement>, item: T) => {
        this.setState({
            contextMenu: {
                anchorEl: event.currentTarget,
                item
            }
        });
    }

    closeContextMenu = () => {
        this.setState({ contextMenu: {} });
    }

}

export default DataExplorer;
