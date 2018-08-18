// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { Grid, Paper, Toolbar, StyleRulesCallback, withStyles, WithStyles, TablePagination, IconButton } from '@material-ui/core';
import MoreVertIcon from "@material-ui/icons/MoreVert";
import { ColumnSelector } from "../column-selector/column-selector";
import { DataTable, DataColumns } from "../data-table/data-table";
import { DataColumn, SortDirection } from "../data-table/data-column";
import { DataTableFilterItem } from '../data-table-filters/data-table-filters';
import { SearchInput } from '../search-input/search-input';
import { ArvadosTheme } from "~/common/custom-theme";
import { DefaultView } from '../default-view/default-view';
import { IconType } from '../icon/icon';

type CssRules = 'searchBox' | "toolbar" | 'defaultRoot' | 'defaultMessage' | 'defaultIcon';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    searchBox: {
        paddingBottom: theme.spacing.unit * 2
    },
    toolbar: {
        paddingTop: theme.spacing.unit * 2
    },
    defaultRoot: {
        position: 'absolute',
        width: '80%',
        left: '50%',
        top: '50%',
        transform: 'translate(-50%, -50%)'
    },
    defaultMessage: {
        fontSize: '1.75rem',
    },
    defaultIcon: {
        fontSize: '6rem'
    }
});

interface DataExplorerDataProps<T> {
    items: T[];
    itemsAvailable: number;
    columns: DataColumns<T>;
    searchValue: string;
    rowsPerPage: number;
    rowsPerPageOptions: number[];
    page: number;
    defaultIcon: IconType;
    defaultMessages: string[];
}

interface DataExplorerActionProps<T> {
    onSetColumns: (columns: DataColumns<T>) => void;
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

type DataExplorerProps<T> = DataExplorerDataProps<T> & DataExplorerActionProps<T> & WithStyles<CssRules>;

export const DataExplorer = withStyles(styles)(
    class DataExplorerGeneric<T> extends React.Component<DataExplorerProps<T>> {
        componentDidMount() {
            if (this.props.onSetColumns) {
                this.props.onSetColumns(this.props.columns);
            }
        }
        render() {
            const {
                columns, onContextMenu, onFiltersChange, onSortToggle, extractKey,
                rowsPerPage, rowsPerPageOptions, onColumnToggle, searchValue, onSearch,
                items, itemsAvailable, onRowClick, onRowDoubleClick, defaultIcon, defaultMessages, classes
            } = this.props;
            return <div>
                { items.length > 0 ? (
                    <Paper>
                        <Toolbar className={classes.toolbar}>
                            <Grid container justify="space-between" wrap="nowrap" alignItems="center">
                                <div className={classes.searchBox}>
                                    <SearchInput
                                        value={searchValue}
                                        onSearch={onSearch}/>
                                </div>
                                <ColumnSelector
                                    columns={columns}
                                    onColumnToggle={onColumnToggle}/>
                            </Grid>
                        </Toolbar>
                        <DataTable
                            columns={[...columns, this.contextMenuColumn]}
                            items={items}
                            onRowClick={(_, item: T) => onRowClick(item)}
                            onContextMenu={onContextMenu}
                            onRowDoubleClick={(_, item: T) => onRowDoubleClick(item)}
                            onFiltersChange={onFiltersChange}
                            onSortToggle={onSortToggle}
                            extractKey={extractKey}/>
                        <Toolbar>
                            <Grid container justify="flex-end">
                                <TablePagination
                                    count={itemsAvailable}
                                    rowsPerPage={rowsPerPage}
                                    rowsPerPageOptions={rowsPerPageOptions}
                                    page={this.props.page}
                                    onChangePage={this.changePage}
                                    onChangeRowsPerPage={this.changeRowsPerPage}
                                    component="div" />
                            </Grid>
                        </Toolbar>
                    </Paper>
                ) : (
                    <DefaultView
                        classRoot={classes.defaultRoot}
                        icon={defaultIcon}
                        classIcon={classes.defaultIcon}
                        messages={defaultMessages}
                        classMessage={classes.defaultMessage} />
                )}
            </div>;
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
            configurable: false,
            sortDirection: SortDirection.NONE,
            filters: [],
            key: "context-actions",
            render: this.renderContextMenuTrigger,
            width: "auto"
        };
    }
);
