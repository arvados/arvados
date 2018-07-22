// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { Grid, Paper, Toolbar, StyleRulesCallback, withStyles, WithStyles, TablePagination, IconButton } from '@material-ui/core';
import MoreVertIcon from "@material-ui/icons/MoreVert";
import { ColumnSelector } from "../column-selector/column-selector";
import { DataTable, DataColumns } from "../data-table/data-table";
import { DataColumn } from "../data-table/data-column";
import { DataTableFilterItem } from '../data-table-filters/data-table-filters';
import { SearchInput } from '../search-input/search-input';
import { ArvadosTheme } from "../../common/custom-theme";

type CssRules = "searchBox" | "toolbar";

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    searchBox: {
        paddingBottom: theme.spacing.unit * 2
    },
    toolbar: {
        paddingTop: theme.spacing.unit * 2
    }
});

interface DataExplorerDataProps<T> {
    items: T[];
    itemsAvailable: number;
    columns: DataColumns<T>;
    searchValue: string;
    rowsPerPage: number;
    rowsPerPageOptions?: number[];
    page: number;
    onSearch: (value: string) => void;
    onRowClick: (item: T) => void;
    onRowDoubleClick: (item: T) => void;
    onColumnToggle: (column: DataColumn<T>) => void;
    onContextMenu: (event: React.MouseEvent<HTMLElement>, item: T) => void;
    onSortToggle: (column: DataColumn<T>) => void;
    onFiltersChange: (filters: DataTableFilterItem[], column: DataColumn<T>) => void;
    onChangePage: (page: number) => void;
    onChangeRowsPerPage: (rowsPerPage: number) => void;
    extractKey?: (item: T) => React.Key;
}

type DataExplorerProps<T> = DataExplorerDataProps<T> & WithStyles<CssRules>;

export const DataExplorer = withStyles(styles)(
    class DataExplorerGeneric<T> extends React.Component<DataExplorerProps<T>> {
        render() {
            return <Paper>
                <Toolbar className={this.props.classes.toolbar}>
                    <Grid container justify="space-between" wrap="nowrap" alignItems="center">
                        <div className={this.props.classes.searchBox}>
                            <SearchInput
                                value={this.props.searchValue}
                                onSearch={this.props.onSearch}/>
                        </div>
                        <ColumnSelector
                            columns={this.props.columns}
                            onColumnToggle={this.props.onColumnToggle}/>
                    </Grid>
                </Toolbar>
                <DataTable
                    columns={[...this.props.columns, this.contextMenuColumn]}
                    items={this.props.items}
                    onRowClick={(_, item: T) => this.props.onRowClick(item)}
                    onContextMenu={this.props.onContextMenu}
                    onRowDoubleClick={(_, item: T) => this.props.onRowDoubleClick(item)}
                    onFiltersChange={this.props.onFiltersChange}
                    onSortToggle={this.props.onSortToggle}
                    extractKey={this.props.extractKey}/>
                <Toolbar>
                    {this.props.items.length > 0 &&
                    <Grid container justify="flex-end">
                        <TablePagination
                            count={this.props.itemsAvailable}
                            rowsPerPage={this.props.rowsPerPage}
                            rowsPerPageOptions={this.props.rowsPerPageOptions}
                            page={this.props.page}
                            onChangePage={this.changePage}
                            onChangeRowsPerPage={this.changeRowsPerPage}
                            component="div"
                        />
                    </Grid>}
                </Toolbar>
            </Paper>;
        }

        changePage = (event: React.MouseEvent<HTMLButtonElement>, page: number) => {
            this.props.onChangePage(page);
        }

        changeRowsPerPage: React.ChangeEventHandler<HTMLTextAreaElement | HTMLInputElement> = (event) => {
            this.props.onChangeRowsPerPage(parseInt(event.target.value, 10));
        }

        renderContextMenuTrigger = (item: T) =>
            <Grid container justify="flex-end">
                <IconButton onClick={event => this.props.onContextMenu(event, item)}>
                    <MoreVertIcon/>
                </IconButton>
            </Grid>

        contextMenuColumn = {
            name: "Actions",
            selected: true,
            key: "context-actions",
            renderHeader: () => null,
            render: this.renderContextMenuTrigger,
            width: "auto"
        };
    }
);
