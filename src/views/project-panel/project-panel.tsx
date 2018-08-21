// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { ProjectPanelItem } from './project-panel-item';
import { Button, StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core';
import { DataExplorer } from "~/views-components/data-explorer/data-explorer";
import { DispatchProp, connect } from 'react-redux';
import { DataColumns } from '~/components/data-table/data-table';
import { RouteComponentProps } from 'react-router';
import { RootState } from '~/store/store';
import { DataTableFilterItem } from '~/components/data-table-filters/data-table-filters';
import { ProcessState } from '~/models/process';
import { SortDirection } from '~/components/data-table/data-column';
import { ResourceKind } from '~/models/resource';
import { resourceLabel } from '~/common/labels';
import { ArvadosTheme } from '~/common/custom-theme';
import { renderName, renderStatus, renderType, renderOwner, renderFileSize, renderDate } from '~/views-components/data-explorer/renderers';
import { restoreBranch } from '~/store/navigation/navigation-action';
import { ProjectIcon } from '~/components/icon/icon';

type CssRules = 'root' | "toolbar" | "button";

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        position: 'relative',
        width: '100%',
        height: '100%'
    },
    toolbar: {
        paddingBottom: theme.spacing.unit * 3,
        textAlign: "right"
    },
    button: {
        marginLeft: theme.spacing.unit
    },
});

export enum ProjectPanelColumnNames {
    NAME = "Name",
    STATUS = "Status",
    TYPE = "Type",
    OWNER = "Owner",
    FILE_SIZE = "File size",
    LAST_MODIFIED = "Last modified"
}

export interface ProjectPanelFilter extends DataTableFilterItem {
    type: ResourceKind | ProcessState;
}

export const columns: DataColumns<ProjectPanelItem, ProjectPanelFilter> = [
    {
        name: ProjectPanelColumnNames.NAME,
        selected: true,
        configurable: true,
        sortDirection: SortDirection.ASC,
        filters: [],
        render: renderName,
        width: "450px"
    },
    {
        name: "Status",
        selected: true,
        configurable: true,
        sortDirection: SortDirection.NONE,
        filters: [
            {
                name: ProcessState.COMMITTED,
                selected: true,
                type: ProcessState.COMMITTED
            },
            {
                name: ProcessState.FINAL,
                selected: true,
                type: ProcessState.FINAL
            },
            {
                name: ProcessState.UNCOMMITTED,
                selected: true,
                type: ProcessState.UNCOMMITTED
            }
        ],
        render: renderStatus,
        width: "75px"
    },
    {
        name: ProjectPanelColumnNames.TYPE,
        selected: true,
        configurable: true,
        sortDirection: SortDirection.NONE,
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
        configurable: true,
        sortDirection: SortDirection.NONE,
        filters: [],
        render: item => renderOwner(item.owner),
        width: "200px"
    },
    {
        name: ProjectPanelColumnNames.FILE_SIZE,
        selected: true,
        configurable: true,
        sortDirection: SortDirection.NONE,
        filters: [],
        render: item => renderFileSize(item.fileSize),
        width: "50px"
    },
    {
        name: ProjectPanelColumnNames.LAST_MODIFIED,
        selected: true,
        configurable: true,
        sortDirection: SortDirection.NONE,
        filters: [],
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
                return <div className={classes.root}>
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
                        columns={columns}
                        onRowClick={this.props.onItemClick}
                        onRowDoubleClick={this.props.onItemDoubleClick}
                        onContextMenu={this.props.onContextMenu}
                        extractKey={(item: ProjectPanelItem) => item.uuid}
                        defaultIcon={ProjectIcon}
                        defaultMessages={['Your project is empty.', 'Please create a project or create a collection and upload a data.']} />
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

            componentDidMount() {
                if (this.props.match.params.id && this.props.currentItemId === '') {
                    this.props.dispatch<any>(restoreBranch(this.props.match.params.id));
                }
            }
        }
    )
);
