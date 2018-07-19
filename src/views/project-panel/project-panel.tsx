// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { ProjectPanelItem } from './project-panel-item';
import { Grid, Typography, Button, StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core';
import { FORMAT_DATE, FORMAT_FILE_SIZE } from '../../common/formatters';
import DataExplorer from "../../views-components/data-explorer/data-explorer";
import { DispatchProp, connect } from 'react-redux';
import { DataColumns } from '../../components/data-table/data-table';
import { RouteComponentProps } from 'react-router';
import { RootState } from '../../store/store';
import { DataTableFilterItem } from '../../components/data-table-filters/data-table-filters';
import { ContainerRequestState } from '../../models/container-request';
import { SortDirection } from '../../components/data-table/data-column';
import { ResourceKind } from '../../models/resource';
import { RESOURCE_LABEL } from '../../common/labels';
import { ProjectIcon, CollectionIcon, ProcessIcon, DefaultIcon } from '../../components/icon/icon';
import { ArvadosTheme } from '../../common/custom-theme';

export const PROJECT_PANEL_ID = "projectPanel";

export interface ProjectPanelFilter extends DataTableFilterItem {
    type: ResourceKind | ContainerRequestState;
}

interface ProjectPanelDataProps {
    currentItemId: string;
}

interface ProjectPanelActionProps {
    onItemClick: (item: ProjectPanelItem) => void;
    onContextMenu: (event: React.MouseEvent<HTMLElement>, item: ProjectPanelItem) => void;
    onDialogOpen: (ownerUuid: string) => void;
    onItemDoubleClick: (item: ProjectPanelItem) => void;
    onItemRouteChange: (itemId: string) => void;
}

type ProjectPanelProps = ProjectPanelDataProps & ProjectPanelActionProps & DispatchProp
                        & WithStyles<CssRules> & RouteComponentProps<{ id: string }>;

class ProjectPanel extends React.Component<ProjectPanelProps> {
    render() {
        const { classes } = this.props;
        return <div>
            <div className={classes.toolbar}>
                <Button color="primary" variant="raised" className={classes.button}>
                    Create a collection
                </Button>
                <Button color="primary" variant="raised" className={classes.button}>
                    Run a process
                </Button>
                <Button color="primary" onClick={this.handleNewProjectClick} variant="raised" className={classes.button}>
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
    
    handleNewProjectClick = () => {
        this.props.onDialogOpen(this.props.currentItemId);
    }
    componentWillReceiveProps({ match, currentItemId, onItemRouteChange }: ProjectPanelProps) {
        if (match.params.id !== currentItemId) {
            onItemRouteChange(match.params.id);
        }
    }
}

type CssRules = "toolbar" | "button";

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    toolbar: {
        paddingBottom: theme.spacing.unit * 3,
        textAlign: "right"
    },
    button: {
        marginLeft: theme.spacing.unit
    },
});

const RENDER_NAME = (item: ProjectPanelItem) =>
    <Grid container alignItems="center" wrap="nowrap" spacing={16}>
        <Grid item>
            {RENDER_ICON(item)}
        </Grid>
        <Grid item>
            <Typography color="primary">
                {item.name}
            </Typography>
        </Grid>
    </Grid>;


const RENDER_ICON = (item: ProjectPanelItem) => {
    switch (item.kind) {
        case ResourceKind.Project:
            return <ProjectIcon />;
        case ResourceKind.Collection:
            return <CollectionIcon />;
        case ResourceKind.Process:
            return <ProcessIcon />;
        default:
            return <DefaultIcon />;
    }
};

const RENDER_DATE = (date: string) => {
    return <Typography noWrap>{FORMAT_DATE(date)}</Typography>;
};

const RENDER_FILE_SIZE = (fileSize?: number) =>
    <Typography noWrap>
        {FORMAT_FILE_SIZE(fileSize)}
    </Typography>;

const RENDER_OWNER = (owner: string) =>
    <Typography noWrap color="primary" >
        {owner}
    </Typography>;

const RENDER_TYPE = (type: string) =>
    <Typography noWrap>
        {RESOURCE_LABEL(type)}
    </Typography>;

const RENDER_STATUS = (item: ProjectPanelItem) =>
    <Typography noWrap align="center" >
        {item.status || "-"}
    </Typography>;

export enum ColumnNames {
    NAME = "Name",
    STATUS = "Status",
    TYPE = "Type",
    OWNER = "Owner",
    FILE_SIZE = "File size",
    LAST_MODIFIED = "Last modified"

}

export const COLUMNS: DataColumns<ProjectPanelItem, ProjectPanelFilter> = [
    {
        name: ColumnNames.NAME,
        selected: true,
        sortDirection: SortDirection.Asc,
        render: RENDER_NAME,
        width: "450px"
    },
    {
        name: "Status",
        selected: true,
        filters: [
            {
                name: ContainerRequestState.Committed,
                selected: true,
                type: ContainerRequestState.Committed
            },
            {
                name: ContainerRequestState.Final,
                selected: true,
                type: ContainerRequestState.Final
            },
            {
                name: ContainerRequestState.Uncommitted,
                selected: true,
                type: ContainerRequestState.Uncommitted
            }
        ],
        render: RENDER_STATUS,
        width: "75px"
    },
    {
        name: ColumnNames.TYPE,
        selected: true,
        filters: [
            {
                name: RESOURCE_LABEL(ResourceKind.Collection),
                selected: true,
                type: ResourceKind.Collection
            },
            {
                name: RESOURCE_LABEL(ResourceKind.Process),
                selected: true,
                type: ResourceKind.Process
            },
            {
                name: RESOURCE_LABEL(ResourceKind.Project),
                selected: true,
                type: ResourceKind.Project
            }
        ],
        render: item => RENDER_TYPE(item.kind),
        width: "125px"
    },
    {
        name: ColumnNames.OWNER,
        selected: true,
        render: item => RENDER_OWNER(item.owner),
        width: "200px"
    },
    {
        name: ColumnNames.FILE_SIZE,
        selected: true,
        render: item => RENDER_FILE_SIZE(item.fileSize),
        width: "50px"
    },
    {
        name: ColumnNames.LAST_MODIFIED,
        selected: true,
        sortDirection: SortDirection.None,
        render: item => RENDER_DATE(item.lastModified),
        width: "150px"
    }
];

export default withStyles(styles)(
    connect((state: RootState) => ({ currentItemId: state.projects.currentItemId }))(ProjectPanel));
