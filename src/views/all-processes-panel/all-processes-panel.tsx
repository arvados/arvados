// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core';
import { DataExplorer } from "views-components/data-explorer/data-explorer";
import { connect, DispatchProp } from 'react-redux';
import { DataColumns } from 'components/data-table/data-table';
import { RouteComponentProps } from 'react-router';
import { DataTableFilterItem } from 'components/data-table-filters/data-table-filters';
import { SortDirection } from 'components/data-table/data-column';
import { ResourceKind } from 'models/resource';
import { ArvadosTheme } from 'common/custom-theme';
import { ALL_PROCESSES_PANEL_ID } from 'store/all-processes-panel/all-processes-panel-action';
import {
    ProcessStatus,
    ResourceName,
    ResourceOwnerWithName,
    ResourceType,
    ContainerRunTime,
    ResourceCreatedAtDate
} from 'views-components/data-explorer/renderers';
import { ProcessIcon } from 'components/icon/icon';
import { openProcessContextMenu } from 'store/context-menu/context-menu-actions';
import { loadDetailsPanel } from 'store/details-panel/details-panel-action';
import { navigateTo } from 'store/navigation/navigation-action';
import { ContainerRequestState } from "models/container-request";
import { RootState } from 'store/store';
import { createTree } from 'models/tree';
import { getInitialProcessStatusFilters, getInitialProcessTypeFilters } from 'store/resource-type-filters/resource-type-filters';
import { getProcess } from 'store/processes/process';
import { ResourcesState } from 'store/resources/resources';

type CssRules = "toolbar" | "button" | "root";

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    toolbar: {
        paddingBottom: theme.spacing.unit * 3,
        textAlign: "right"
    },
    button: {
        marginLeft: theme.spacing.unit
    },
    root: {
        width: '100%',
    }
});

export enum AllProcessesPanelColumnNames {
    NAME = "Name",
    STATUS = "Status",
    TYPE = "Type",
    OWNER = "Owner",
    CREATED_AT = "Created at",
    RUNTIME = "Run Time"
}

export interface AllProcessesPanelFilter extends DataTableFilterItem {
    type: ResourceKind | ContainerRequestState;
}

export const allProcessesPanelColumns: DataColumns<string> = [
    {
        name: AllProcessesPanelColumnNames.NAME,
        selected: true,
        configurable: true,
        sortDirection: SortDirection.NONE,
        filters: createTree(),
        render: uuid => <ResourceName uuid={uuid} />
    },
    {
        name: AllProcessesPanelColumnNames.STATUS,
        selected: true,
        configurable: true,
        mutuallyExclusiveFilters: true,
        filters: getInitialProcessStatusFilters(),
        render: uuid => <ProcessStatus uuid={uuid} />
    },
    {
        name: AllProcessesPanelColumnNames.TYPE,
        selected: true,
        configurable: true,
        filters: getInitialProcessTypeFilters(),
        render: uuid => <ResourceType uuid={uuid} />
    },
    {
        name: AllProcessesPanelColumnNames.OWNER,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <ResourceOwnerWithName uuid={uuid} />
    },
    {
        name: AllProcessesPanelColumnNames.CREATED_AT,
        selected: true,
        configurable: true,
        sortDirection: SortDirection.DESC,
        filters: createTree(),
        render: uuid => <ResourceCreatedAtDate uuid={uuid} />
    },
    {
        name: AllProcessesPanelColumnNames.RUNTIME,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <ContainerRunTime uuid={uuid} />
    }
];

interface AllProcessesPanelDataProps {
    resources: ResourcesState;
}

interface AllProcessesPanelActionProps {
    onItemClick: (item: string) => void;
    onDialogOpen: (ownerUuid: string) => void;
    onItemDoubleClick: (item: string) => void;
}
const mapStateToProps = (state : RootState): AllProcessesPanelDataProps => ({
    resources: state.resources
});

type AllProcessesPanelProps = AllProcessesPanelDataProps & AllProcessesPanelActionProps & DispatchProp
    & WithStyles<CssRules> & RouteComponentProps<{ id: string }>;

export const AllProcessesPanel = withStyles(styles)(
    connect(mapStateToProps)(
        class extends React.Component<AllProcessesPanelProps> {
            handleContextMenu = (event: React.MouseEvent<HTMLElement>, resourceUuid: string) => {
                const process = getProcess(resourceUuid)(this.props.resources);
                if (process) {
                    this.props.dispatch<any>(openProcessContextMenu(event, process));
                }
                this.props.dispatch<any>(loadDetailsPanel(resourceUuid));
            }

            handleRowDoubleClick = (uuid: string) => {
                this.props.dispatch<any>(navigateTo(uuid));
            }

            handleRowClick = (uuid: string) => {
                this.props.dispatch<any>(loadDetailsPanel(uuid));
            }

            render() {
                return <div className={this.props.classes.root}><DataExplorer
                    id={ALL_PROCESSES_PANEL_ID}
                    onRowClick={this.handleRowClick}
                    onRowDoubleClick={this.handleRowDoubleClick}
                    onContextMenu={this.handleContextMenu}
                    contextMenuColumn={true}
                    defaultViewIcon={ProcessIcon}
                    defaultViewMessages={['Processes list empty.']} />
                </div>
            }
        }
    )
);
