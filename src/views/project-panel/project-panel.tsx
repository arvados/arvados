// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { ProjectPanelItem } from './project-panel-item';
import { Grid, Typography, Button, Toolbar, StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core';
import { formatDate, formatFileSize } from '../../common/formatters';
import DataExplorer from "../../views-components/data-explorer/data-explorer";
import { DataColumn, toggleSortDirection } from '../../components/data-table/data-column';
import { DataTableFilterItem } from '../../components/data-table-filters/data-table-filters';
import { ContextMenuAction } from '../../components/context-menu/context-menu';
import { DispatchProp, connect } from 'react-redux';
import actions from "../../store/data-explorer/data-explorer-action";
import { DataColumns } from '../../components/data-table/data-table';
import { ResourceKind } from "../../models/resource";
import { RouteComponentProps } from 'react-router';
import { RootState } from '../../store/store';

export const PROJECT_PANEL_ID = "projectPanel";

type ProjectPanelProps = {
    currentItemId: string,
    onItemClick: (item: ProjectPanelItem) => void,
    onItemRouteChange: (itemId: string) => void
}
    & DispatchProp
    & WithStyles<CssRules>
    & RouteComponentProps<{ id: string }>;
class ProjectPanel extends React.Component<ProjectPanelProps> {
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
                id={PROJECT_PANEL_ID}
                contextActions={contextMenuActions}
                onColumnToggle={this.toggleColumn}
                onFiltersChange={this.changeFilters}
                onRowClick={this.props.onItemClick}
                onSortToggle={this.toggleSort}
                onSearch={this.search}
                onContextAction={this.executeAction}
                onChangePage={this.changePage}
                onChangeRowsPerPage={this.changeRowsPerPage} />;
        </div>;
    }

    componentDidMount() {
        this.props.dispatch(actions.SET_COLUMNS({ id: PROJECT_PANEL_ID, columns }));
    }

    componentWillReceiveProps({ match, currentItemId }: ProjectPanelProps) {
        if (match.params.id !== currentItemId) {
            this.props.onItemRouteChange(match.params.id);
        }
    }

    toggleColumn = (toggledColumn: DataColumn<ProjectPanelItem>) => {
        this.props.dispatch(actions.TOGGLE_COLUMN({ id: PROJECT_PANEL_ID, columnName: toggledColumn.name }));
    }

    toggleSort = (column: DataColumn<ProjectPanelItem>) => {
        this.props.dispatch(actions.TOGGLE_SORT({ id: PROJECT_PANEL_ID, columnName: column.name }));
    }

    changeFilters = (filters: DataTableFilterItem[], column: DataColumn<ProjectPanelItem>) => {
        this.props.dispatch(actions.SET_FILTERS({ id: PROJECT_PANEL_ID, columnName: column.name, filters }));
    }

    executeAction = (action: ContextMenuAction, item: ProjectPanelItem) => {
        alert(`Executing ${action.name} on ${item.name}`);
    }

    search = (searchValue: string) => {
        this.props.dispatch(actions.SET_SEARCH_VALUE({ id: PROJECT_PANEL_ID, searchValue }));
    }

    changePage = (page: number) => {
        this.props.dispatch(actions.SET_PAGE({ id: PROJECT_PANEL_ID, page }));
    }

    changeRowsPerPage = (rowsPerPage: number) => {
        this.props.dispatch(actions.SET_ROWS_PER_PAGE({ id: PROJECT_PANEL_ID, rowsPerPage }));
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

const renderName = (item: ProjectPanelItem) =>
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


const renderIcon = (item: ProjectPanelItem) => {
    switch (item.kind) {
        case ResourceKind.PROJECT:
            return <i className="fas fa-folder fa-lg" />;
        case ResourceKind.COLLECTION:
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

const renderStatus = (item: ProjectPanelItem) =>
    <Typography noWrap align="center">
        {item.status || "-"}
    </Typography>;

const columns: DataColumns<ProjectPanelItem> = [{
    name: "Name",
    selected: true,
    sortDirection: "desc",
    render: renderName,
    width: "450px"
}, {
    name: "Status",
    selected: true,
    render: renderStatus,
    width: "75px"
}, {
    name: "Type",
    selected: true,
    filters: [{
        name: "Collection",
        selected: true
    }, {
        name: "Project",
        selected: true
    }],
    render: item => renderType(item.kind),
    width: "125px"
}, {
    name: "Owner",
    selected: true,
    render: item => renderOwner(item.owner),
    width: "200px"
}, {
    name: "File size",
    selected: true,
    render: item => renderFileSize(item.fileSize),
    width: "50px"
}, {
    name: "Last modified",
    selected: true,
    sortDirection: "none",
    render: item => renderDate(item.lastModified),
    width: "150px"
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

export default withStyles(styles)(
    connect((state: RootState) => ({ currentItemId: state.projects.currentItemId }))(
        ProjectPanel));
