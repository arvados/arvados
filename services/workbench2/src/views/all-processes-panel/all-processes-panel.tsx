// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { DataExplorer } from "views-components/data-explorer/data-explorer";
import { connect, DispatchProp } from "react-redux";
import { RouteComponentProps } from "react-router";
import { DataTableFilterItem } from "components/data-table-filters/data-table-filters";
import { DataColumns, SortDirection } from "components/data-table/data-column";
import { ResourceKind } from "models/resource";
import { ArvadosTheme } from "common/custom-theme";
import { ALL_PROCESSES_PANEL_ID } from "store/all-processes-panel/all-processes-panel-action";
import {
    ProcessStatus,
    ContainerRunTime,
    renderType,
    RenderName,
    OwnerWithName,
    renderCreatedAtDate,
} from "views-components/data-explorer/renderers";
import { ProcessIcon } from "components/icon/icon";
import { openProcessContextMenu } from "store/context-menu/context-menu-actions";
import { loadDetailsPanel } from "store/details-panel/details-panel-action";
import { navigateTo } from "store/navigation/navigation-action";
import { ContainerRequestResource, ContainerRequestState } from "models/container-request";
import { RootState } from "store/store";
import { createTree } from "models/tree";
import { getInitialProcessStatusFilters, getInitialProcessTypeFilters } from "store/resource-type-filters/resource-type-filters";
import { getProcess } from "store/processes/process";
import { ResourcesState } from "store/resources/resources";
import { toggleOne, deselectAllOthers } from "store/multiselect/multiselect-actions";

type CssRules = "toolbar" | "button" | "root";

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    toolbar: {
        paddingBottom: theme.spacing(3),
        textAlign: "right",
    },
    button: {
        marginLeft: theme.spacing(1),
    },
    root: {
        width: "100%",
    },
});

export enum AllProcessesPanelColumnNames {
    NAME = "Name",
    STATUS = "Status",
    TYPE = "Type",
    OWNER = "Owner",
    CREATED_AT = "Created at",
    RUNTIME = "Run Time",
}

export interface AllProcessesPanelFilter extends DataTableFilterItem {
    type: ResourceKind | ContainerRequestState;
}

export const allProcessesPanelColumns: DataColumns<string, ContainerRequestResource> = [
    {
        name: AllProcessesPanelColumnNames.NAME,
        selected: true,
        configurable: true,
        sort: { direction: SortDirection.NONE, field: "name" },
        filters: createTree(),
        render: (resource) => <RenderName resource={resource} />,
    },
    {
        name: AllProcessesPanelColumnNames.STATUS,
        selected: true,
        configurable: true,
        mutuallyExclusiveFilters: true,
        filters: getInitialProcessStatusFilters(),
        render: (resource: ContainerRequestResource) => <ProcessStatus uuid={resource.uuid} />,
    },
    {
        name: AllProcessesPanelColumnNames.TYPE,
        selected: true,
        configurable: true,
        filters: getInitialProcessTypeFilters(),
        render: (resource: ContainerRequestResource) => renderType(resource),
    },
    {
        name: AllProcessesPanelColumnNames.OWNER,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: (resource: ContainerRequestResource) => <OwnerWithName resource={resource} />,
    },
    {
        name: AllProcessesPanelColumnNames.CREATED_AT,
        selected: true,
        configurable: true,
        sort: { direction: SortDirection.DESC, field: "createdAt" },
        filters: createTree(),
        render: (resource: ContainerRequestResource) => renderCreatedAtDate(resource),
    },
    {
        name: AllProcessesPanelColumnNames.RUNTIME,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: (resource: ContainerRequestResource) => <ContainerRunTime uuid={resource.uuid} />,
    },
];

interface AllProcessesPanelDataProps {
    resources: ResourcesState;
}

interface AllProcessesPanelActionProps {
    onItemClick: (item: string) => void;
    onDialogOpen: (ownerUuid: string) => void;
    onItemDoubleClick: (item: string) => void;
}
const mapStateToProps = (state: RootState): AllProcessesPanelDataProps => ({
    resources: state.resources,
});

type AllProcessesPanelProps = AllProcessesPanelDataProps &
    AllProcessesPanelActionProps &
    DispatchProp &
    WithStyles<CssRules> &
    RouteComponentProps<{ id: string }>;

export const AllProcessesPanel = withStyles(styles)(
    connect(mapStateToProps)(
        class extends React.Component<AllProcessesPanelProps> {
            handleContextMenu = (event: React.MouseEvent<HTMLElement>, resource: ContainerRequestResource) => {
                const process = getProcess(resource.uuid)(this.props.resources);
                if (process) {
                    this.props.dispatch<any>(openProcessContextMenu(event, process));
                }
                this.props.dispatch<any>(loadDetailsPanel(resource.uuid));
            };

            handleRowDoubleClick = (uuid: string) => {
                this.props.dispatch<any>(navigateTo(uuid));
            };

            handleRowClick = ({uuid}: ContainerRequestResource) => {
                this.props.dispatch<any>(toggleOne(uuid))
                this.props.dispatch<any>(deselectAllOthers(uuid))
                this.props.dispatch<any>(loadDetailsPanel(uuid));
            };

            render() {
                return (
                    <div className={this.props.classes.root}>
                        <DataExplorer
                            id={ALL_PROCESSES_PANEL_ID}
                            onRowClick={this.handleRowClick}
                            onRowDoubleClick={this.handleRowDoubleClick}
                            onContextMenu={this.handleContextMenu}
                            contextMenuColumn={false}
                            defaultViewIcon={ProcessIcon}
                            defaultViewMessages={["Processes list empty."]}
                        />
                    </div>
                );
            }
        }
    )
);
