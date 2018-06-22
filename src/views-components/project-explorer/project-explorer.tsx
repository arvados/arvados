// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { ProjectExplorerItem } from './project-explorer-item';
import { Grid, Typography } from '@material-ui/core';
import { formatDate, formatFileSize } from '../../common/formatters';
import DataExplorer from '../../components/data-explorer/data-explorer';
import { DataColumn, toggleSortDirection, resetSortDirection } from '../../components/data-table/data-column';
import { DataTableFilterItem } from '../../components/data-table-filters/data-table-filters';
import { ContextMenuAction } from '../../components/context-menu/context-menu';

export interface ProjectExplorerContextActions {
    onAddToFavourite: (item: ProjectExplorerItem) => void;
    onCopy: (item: ProjectExplorerItem) => void;
    onDownload: (item: ProjectExplorerItem) => void;
    onMoveTo: (item: ProjectExplorerItem) => void;
    onRemove: (item: ProjectExplorerItem) => void;
    onRename: (item: ProjectExplorerItem) => void;
    onShare: (item: ProjectExplorerItem) => void;
}

interface ProjectExplorerProps {
    items: ProjectExplorerItem[];
}

interface ProjectExplorerState {
    columns: Array<DataColumn<ProjectExplorerItem>>;
    searchValue: string;
    page: number;
    rowsPerPage: number;
}

class ProjectExplorer extends React.Component<ProjectExplorerProps, ProjectExplorerState> {
    state: ProjectExplorerState = {
        searchValue: "",
        page: 0,
        rowsPerPage: 10,
        columns: [{
            name: "Name",
            selected: true,
            sortDirection: "asc",
            render: renderName
        }, {
            name: "Status",
            selected: true,
            filters: [{
                name: "In progress",
                selected: true
            }, {
                name: "Complete",
                selected: true
            }],
            render: renderStatus
        }, {
            name: "Type",
            selected: true,
            filters: [{
                name: "Collection",
                selected: true
            }, {
                name: "Group",
                selected: true
            }],
            render: item => renderType(item.type)
        }, {
            name: "Owner",
            selected: true,
            render: item => renderOwner(item.owner)
        }, {
            name: "File size",
            selected: true,
            sortDirection: "none",
            render: item => renderFileSize(item.fileSize)
        }, {
            name: "Last modified",
            selected: true,
            render: item => renderDate(item.lastModified)
        }]
    };

    contextMenuActions = [[{
        icon: "fas fa-users fa-fw",
        name: "Share"
    }, {
        icon: "fas fa-sign-out-alt fa-fw",
        name: "Move to"
    }, {
        icon: "fas fa-star fa-fw",
        name: "Add to favourite"
    }, {
        icon: "fas fa-edit fa-fw",
        name: "Rename"
    }, {
        icon: "fas fa-copy fa-fw",
        name: "Make a copy"
    }, {
        icon: "fas fa-download fa-fw",
        name: "Download"
    }], [{
        icon: "fas fa-trash-alt fa-fw",
        name: "Remove"
    }
    ]];

    render() {
        return <DataExplorer
            items={this.props.items}
            columns={this.state.columns}
            contextActions={this.contextMenuActions}
            searchValue={this.state.searchValue}
            page={this.state.page}
            rowsPerPage={this.state.rowsPerPage}
            onColumnToggle={this.toggleColumn}
            onFiltersChange={this.changeFilters}
            onRowClick={console.log}
            onSortToggle={this.toggleSort}
            onSearch={this.search}
            onContextAction={this.executeAction}
            onChangePage={this.changePage}
            onChangeRowsPerPage={this.changeRowsPerPage} />;
    }

    toggleColumn = (toggledColumn: DataColumn<ProjectExplorerItem>) => {
        this.setState({
            columns: this.state.columns.map(column =>
                column.name === toggledColumn.name
                    ? { ...column, selected: !column.selected }
                    : column
            )
        });
    }

    toggleSort = (toggledColumn: DataColumn<ProjectExplorerItem>) => {
        this.setState({
            columns: this.state.columns.map(column =>
                column.name === toggledColumn.name
                    ? toggleSortDirection(column)
                    : resetSortDirection(column)
            )
        });
    }

    changeFilters = (filters: DataTableFilterItem[], updatedColumn: DataColumn<ProjectExplorerItem>) => {
        this.setState({
            columns: this.state.columns.map(column =>
                column.name === updatedColumn.name
                    ? { ...column, filters }
                    : column
            )
        });
    }

    executeAction = (action: ContextMenuAction, item: ProjectExplorerItem) => {
        alert(`Executing ${action.name} on ${item.name}`);
    }

    search = (searchValue: string) => {
        this.setState({ searchValue });
    }

    changePage = (page: number) => {
        this.setState({ page });
    }

    changeRowsPerPage = (rowsPerPage: number) => {
        this.setState({ rowsPerPage });
    }
}

const renderName = (item: ProjectExplorerItem) =>
    <Grid
        container
        alignItems="center"
        wrap="nowrap"
        spacing={16}>
        <Grid item>
            {renderIcon(item)}
        </Grid>
        <Grid item>
            <Typography color="primary">
                {item.name}
            </Typography>
        </Grid>
    </Grid>;

const renderIcon = (item: ProjectExplorerItem) => {
    switch (item.type) {
        case "arvados#group":
            return <i className="fas fa-folder fa-lg" />;
        case "arvados#groupList":
            return <i className="fas fa-th fa-lg" />;
        default:
            return <i />;
    }
};

const renderDate = (date: string) =>
    <Typography noWrap>
        {formatDate(date)}
    </Typography>;

const renderFileSize = (fileSize?: number) =>
    <Typography noWrap>
        {formatFileSize(fileSize)}
    </Typography>;

const renderOwner = (owner: string) =>
    <Typography noWrap color="primary">
        {owner}
    </Typography>;

const renderType = (type: string) =>
    <Typography noWrap>
        {type}
    </Typography>;

const renderStatus = (item: ProjectExplorerItem) =>
    <Typography noWrap align="center">
        {item.status || "-"}
    </Typography>;

export default ProjectExplorer;
