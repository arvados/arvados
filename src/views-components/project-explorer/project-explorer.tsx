// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { ProjectExplorerItem } from './project-explorer-item';
import { Grid, Typography, Button, Toolbar, StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core';
import { formatDate, formatFileSize } from '../../common/formatters';
import DataExplorer from '../data-explorer/data-explorer';
import { DataColumn } from '../../components/data-table/data-column';
import { DataTableFilterItem } from '../../components/data-table-filters/data-table-filters';
import { ContextMenuAction } from '../../components/context-menu/context-menu';
import { DispatchProp, connect } from 'react-redux';
import actions from "../../store/data-explorer/data-explorer-action";
import { DataColumns } from '../../components/data-table/data-table';

export const PROJECT_EXPLORER_ID = "projectExplorer";
class ProjectExplorer extends React.Component<DispatchProp & WithStyles<CssRules>> {
    render() {
        return <div>
            <div className={this.props.classes.toolbar}>
                <Button color="primary" variant="raised" className={this.props.classes.button}>
                    Create a collection
                </Button>
                <Button color="primary" variant="raised" className={this.props.classes.button}>
                    Run a process
                </Button>
                <Button color="primary" variant="raised" className={this.props.classes.button}>
                    Create a project
                </Button>
            </div>
            <DataExplorer
                id={PROJECT_EXPLORER_ID}
                contextActions={contextMenuActions}
                onColumnToggle={this.toggleColumn}
                onFiltersChange={this.changeFilters}
                onRowClick={console.log}
                onSortToggle={this.toggleSort}
                onSearch={this.search}
                onContextAction={this.executeAction}
                onChangePage={this.changePage}
                onChangeRowsPerPage={this.changeRowsPerPage} />;
        </div>;
    }

    componentDidMount() {
        this.props.dispatch(actions.SET_COLUMNS({ id: PROJECT_EXPLORER_ID, columns }));
    }

    toggleColumn = (toggledColumn: DataColumn<ProjectExplorerItem>) => {
        this.props.dispatch(actions.TOGGLE_COLUMN({ id: PROJECT_EXPLORER_ID, columnName: toggledColumn.name }));
    }

    toggleSort = (toggledColumn: DataColumn<ProjectExplorerItem>) => {
        this.props.dispatch(actions.TOGGLE_SORT({ id: PROJECT_EXPLORER_ID, columnName: toggledColumn.name }));
    }

    changeFilters = (filters: DataTableFilterItem[], updatedColumn: DataColumn<ProjectExplorerItem>) => {
        this.props.dispatch(actions.SET_FILTERS({ id: PROJECT_EXPLORER_ID, columnName: updatedColumn.name, filters }));
    }

    executeAction = (action: ContextMenuAction, item: ProjectExplorerItem) => {
        alert(`Executing ${action.name} on ${item.name}`);
    }

    search = (searchValue: string) => {
        this.props.dispatch(actions.SET_SEARCH_VALUE({ id: PROJECT_EXPLORER_ID, searchValue }));
    }

    changePage = (page: number) => {
        this.props.dispatch(actions.SET_PAGE({ id: PROJECT_EXPLORER_ID, page }));
    }

    changeRowsPerPage = (rowsPerPage: number) => {
        this.props.dispatch(actions.SET_ROWS_PER_PAGE({ id: PROJECT_EXPLORER_ID, rowsPerPage }));
    }
}

type CssRules = "toolbar" | "button";

const styles: StyleRulesCallback<CssRules> = theme => ({
    toolbar: {
        paddingBottom: theme.spacing.unit * 3,
        textAlign: "right"
    },
    button: {
        marginLeft: theme.spacing.unit
    }
});

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

const columns: DataColumns<ProjectExplorerItem> = [{
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
}];

const contextMenuActions = [[{
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

export default withStyles(styles)(connect()(ProjectExplorer));
