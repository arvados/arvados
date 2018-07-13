// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { ProjectPanelItem } from './project-panel-item';
import { Grid, Typography, Button, StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core';
import { formatDate, formatFileSize } from '../../common/formatters';
import DataExplorer from "../../views-components/data-explorer/data-explorer";
import { DispatchProp, connect } from 'react-redux';
import { DataColumns } from '../../components/data-table/data-table';
import { RouteComponentProps } from 'react-router';
import { RootState } from '../../store/store';
import { DataTableFilterItem } from '../../components/data-table-filters/data-table-filters';
import { ContainerRequestState } from '../../models/container-request';
import { SortDirection } from '../../components/data-table/data-column';
import { ResourceKind } from '../../models/resource';
import { resourceLabel } from '../../common/labels';

export const PROJECT_PANEL_ID = "projectPanel";

export interface ProjectPanelFilter extends DataTableFilterItem {
    type: ResourceKind | ContainerRequestState;
}

type ProjectPanelProps = {
    currentItemId: string,
    onItemClick: (item: ProjectPanelItem) => void,
    onContextMenu: (event: React.MouseEvent<HTMLElement>, item: ProjectPanelItem) => void;
    onDialogOpen: (ownerUuid: string) => void;
    onItemDoubleClick: (item: ProjectPanelItem) => void,
    onItemRouteChange: (itemId: string) => void
}
    & DispatchProp
    & WithStyles<CssRules>
    & RouteComponentProps<{ id: string }>;

class ProjectPanel extends React.Component<ProjectPanelProps> {
    render() {
        const { classes, currentItemId, onItemClick, onItemDoubleClick, onContextMenu, onDialogOpen } = this.props;
        return <div>
            <div className={classes.toolbar}>
                <Button color="primary" variant="raised" className={classes.button}>
                    Create a collection
                </Button>
                <Button color="primary" variant="raised" className={classes.button}>
                    Run a process
                </Button>
                <Button color="primary" onClick={() => onDialogOpen(currentItemId)} variant="raised" className={classes.button}>
                    New project
                </Button>
            </div>
            <DataExplorer
                id={PROJECT_PANEL_ID}
                onRowClick={this.props.onItemClick}
                onRowDoubleClick={this.props.onItemDoubleClick}
                onContextMenu={this.props.onContextMenu}
                extractKey={(item: ProjectPanelItem) => item.uuid} />
        </div>;
    }

    componentWillReceiveProps({ match, currentItemId, onItemRouteChange }: ProjectPanelProps) {
        if (match.params.id !== currentItemId) {
            onItemRouteChange(match.params.id);
        }
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
    },
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
        case ResourceKind.Project:
            return <i className="fas fa-folder fa-lg" />;
        case ResourceKind.Collection:
            return <i className="fas fa-archive fa-lg" />;
        case ResourceKind.Process:
            return <i className="fas fa-cogs fa-lg" />;
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

const renderType = (type: string) => {
    return <Typography noWrap>
        {resourceLabel(type)}
    </Typography>;
};

const renderStatus = (item: ProjectPanelItem) =>
    <Typography noWrap align="center">
        {item.status || "-"}
    </Typography>;

export enum ProjectPanelColumnNames {
    Name = "Name",
    Status = "Status",
    Type = "Type",
    Owner = "Owner",
    FileSize = "File size",
    LastModified = "Last modified"

}

export const columns: DataColumns<ProjectPanelItem, ProjectPanelFilter> = [{
    name: ProjectPanelColumnNames.Name,
    selected: true,
    sortDirection: SortDirection.Asc,
    render: renderName,
    width: "450px"
}, {
    name: "Status",
    selected: true,
    filters: [{
        name: ContainerRequestState.Committed,
        selected: true,
        type: ContainerRequestState.Committed
    }, {
        name: ContainerRequestState.Final,
        selected: true,
        type: ContainerRequestState.Final
    }, {
        name: ContainerRequestState.Uncommitted,
        selected: true,
        type: ContainerRequestState.Uncommitted
    }],
    render: renderStatus,
    width: "75px"
}, {
    name: ProjectPanelColumnNames.Type,
    selected: true,
    filters: [{
        name: resourceLabel(ResourceKind.Collection),
        selected: true,
        type: ResourceKind.Collection
    }, {
        name: resourceLabel(ResourceKind.Process),
        selected: true,
        type: ResourceKind.Process
    }, {
        name: resourceLabel(ResourceKind.Project),
        selected: true,
        type: ResourceKind.Project
    }],
    render: item => renderType(item.kind),
    width: "125px"
}, {
    name: ProjectPanelColumnNames.Owner,
    selected: true,
    render: item => renderOwner(item.owner),
    width: "200px"
}, {
    name: ProjectPanelColumnNames.FileSize,
    selected: true,
    render: item => renderFileSize(item.fileSize),
    width: "50px"
}, {
    name: ProjectPanelColumnNames.LastModified,
    selected: true,
    sortDirection: SortDirection.None,
    render: item => renderDate(item.lastModified),
    width: "150px"
}];


export default withStyles(styles)(
    connect((state: RootState) => ({ currentItemId: state.projects.currentItemId }))(
        ProjectPanel));
