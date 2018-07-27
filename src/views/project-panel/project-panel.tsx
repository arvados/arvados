// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { ProjectPanelItem } from './project-panel-item';
import { Grid, Typography, Button, StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core';
import { formatDate, formatFileSize } from '../../common/formatters';
import { DataExplorer } from "../../views-components/data-explorer/data-explorer";
import { DispatchProp, connect } from 'react-redux';
import { DataColumns } from '../../components/data-table/data-table';
import { RouteComponentProps } from 'react-router';
import { RootState } from '../../store/store';
import { DataTableFilterItem } from '../../components/data-table-filters/data-table-filters';
import { ContainerRequestState } from '../../models/container-request';
import { SortDirection } from '../../components/data-table/data-column';
import { ResourceKind } from '../../models/resource';
import { resourceLabel } from '../../common/labels';
import { ProjectIcon, CollectionIcon, ProcessIcon, DefaultIcon, FavoriteIcon } from '../../components/icon/icon';
import { ArvadosTheme } from '../../common/custom-theme';
import { FavoriteStar } from '../../views-components/favorite-star/favorite-star';

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

const renderName = (item: ProjectPanelItem) =>
    <Grid container alignItems="center" wrap="nowrap" spacing={16}>
        <Grid item>
            {renderIcon(item)}
        </Grid>
        <Grid item>
            <Typography color="primary">
                {item.name}
            </Typography>
        </Grid>
        <Grid item>
            <Typography variant="caption">
                <FavoriteStar resourceUuid={item.uuid} />
            </Typography>
        </Grid>
    </Grid>;


const renderIcon = (item: ProjectPanelItem) => {
    switch (item.kind) {
        case ResourceKind.PROJECT:
            return <ProjectIcon />;
        case ResourceKind.COLLECTION:
            return <CollectionIcon />;
        case ResourceKind.PROCESS:
            return <ProcessIcon />;
        default:
            return <DefaultIcon />;
    }
};

const renderDate = (date: string) => {
    return <Typography noWrap>{formatDate(date)}</Typography>;
};

const renderFileSize = (fileSize?: number) =>
    <Typography noWrap>
        {formatFileSize(fileSize)}
    </Typography>;

const renderOwner = (owner: string) =>
    <Typography noWrap color="primary" >
        {owner}
    </Typography>;

const renderType = (type: string) =>
    <Typography noWrap>
        {resourceLabel(type)}
    </Typography>;

const renderStatus = (item: ProjectPanelItem) =>
    <Typography noWrap align="center" >
        {item.status || "-"}
    </Typography>;

export enum ProjectPanelColumnNames {
    NAME = "Name",
    STATUS = "Status",
    TYPE = "Type",
    OWNER = "Owner",
    FILE_SIZE = "File size",
    LAST_MODIFIED = "Last modified"
}

export interface ProjectPanelFilter extends DataTableFilterItem {
    type: ResourceKind | ContainerRequestState;
}

export const columns: DataColumns<ProjectPanelItem, ProjectPanelFilter> = [
    {
        name: ProjectPanelColumnNames.NAME,
        selected: true,
        sortDirection: SortDirection.ASC,
        render: renderName,
        width: "450px"
    },
    {
        name: "Status",
        selected: true,
        filters: [
            {
                name: ContainerRequestState.COMMITTED,
                selected: true,
                type: ContainerRequestState.COMMITTED
            },
            {
                name: ContainerRequestState.FINAL,
                selected: true,
                type: ContainerRequestState.FINAL
            },
            {
                name: ContainerRequestState.UNCOMMITTED,
                selected: true,
                type: ContainerRequestState.UNCOMMITTED
            }
        ],
        render: renderStatus,
        width: "75px"
    },
    {
        name: ProjectPanelColumnNames.TYPE,
        selected: true,
        filters: [
            {
                name: resourceLabel(ResourceKind.COLLECTION),
                selected: true,
                type: ResourceKind.COLLECTION
            },
            {
                name: resourceLabel(ResourceKind.PROCESS),
                selected: true,
                type: ResourceKind.PROCESS
            },
            {
                name: resourceLabel(ResourceKind.PROJECT),
                selected: true,
                type: ResourceKind.PROJECT
            }
        ],
        render: item => renderType(item.kind),
        width: "125px"
    },
    {
        name: ProjectPanelColumnNames.OWNER,
        selected: true,
        render: item => renderOwner(item.owner),
        width: "200px"
    },
    {
        name: ProjectPanelColumnNames.FILE_SIZE,
        selected: true,
        render: item => renderFileSize(item.fileSize),
        width: "50px"
    },
    {
        name: ProjectPanelColumnNames.LAST_MODIFIED,
        selected: true,
        sortDirection: SortDirection.NONE,
        render: item => renderDate(item.lastModified),
        width: "150px"
    }
];

export const PROJECT_PANEL_ID = "projectPanel";

interface ProjectPanelDataProps {
    currentItemId: string;
}

interface ProjectPanelActionProps {
    onItemClick: (item: ProjectPanelItem) => void;
    onContextMenu: (event: React.MouseEvent<HTMLElement>, item: ProjectPanelItem) => void;
    onProjectCreationDialogOpen: (ownerUuid: string) => void;
    onCollectionCreationDialogOpen: (ownerUuid: string) => void;
    onItemDoubleClick: (item: ProjectPanelItem) => void;
    onItemRouteChange: (itemId: string) => void;
}

type ProjectPanelProps = ProjectPanelDataProps & ProjectPanelActionProps & DispatchProp
    & WithStyles<CssRules> & RouteComponentProps<{ id: string }>;

export const ProjectPanel = withStyles(styles)(
    connect((state: RootState) => ({ currentItemId: state.projects.currentItemId }))(
        class extends React.Component<ProjectPanelProps> {
            render() {
                const { classes } = this.props;
                return <div>
                    <div className={classes.toolbar}>
                        <Button color="primary" onClick={this.handleNewCollectionClick} variant="raised" className={classes.button}>
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
                this.props.onProjectCreationDialogOpen(this.props.currentItemId);
            }

            handleNewCollectionClick = () => {
                this.props.onCollectionCreationDialogOpen(this.props.currentItemId);
            }
            componentWillReceiveProps({ match, currentItemId, onItemRouteChange }: ProjectPanelProps) {
                if (match.params.id !== currentItemId) {
                    onItemRouteChange(match.params.id);
                }
            }
        }
    )
);