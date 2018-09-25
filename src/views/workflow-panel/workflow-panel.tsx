// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { DataExplorer } from "~/views-components/data-explorer/data-explorer";
import { connect, DispatchProp } from 'react-redux';
import { RootState } from '~/store/store';
import { WorkflowIcon } from '~/components/icon/icon';
import { ResourcesState } from '~/store/resources/resources';
import { DataTableDefaultView } from '~/components/data-table-default-view/data-table-default-view';
import { WORKFLOW_PANEL_ID } from '~/store/workflow-panel/workflow-panel-actions';
import {
    ResourceLastModifiedDate,
    RosurceWorkflowName,
    ResourceWorkflowStatus,
    ResourceShare
} from "~/views-components/data-explorer/renderers";
import { SortDirection } from '~/components/data-table/data-column';
import { DataColumns } from '~/components/data-table/data-table';
import { DataTableFilterItem } from '~/components/data-table-filters/data-table-filters';
import { Grid } from '@material-ui/core';
import { WorkflowDetailsCard } from './workflow-description-card';
import { loadDetailsPanel } from '~/store/details-panel/details-panel-action';
import { navigateTo } from '~/store/navigation/navigation-action';

export enum WorkflowPanelColumnNames {
    NAME = "Name",
    AUTHORISATION = "Authorisation",
    LAST_MODIFIED = "Last modified",
    SHARE = 'Share'
}

export interface WorkflowPanelFilter extends DataTableFilterItem {
    type: ResourceStatus;
}

interface WorkflowPanelDataProps {
    resources: ResourcesState;
}

export enum ResourceStatus {
    PUBLIC = "Public",
    PRIVATE = "Private",
    SHARED = "Shared"
}

const resourceStatus = (type: string) => {
    switch (type) {
        case ResourceStatus.PUBLIC:
            return "Public";
        case ResourceStatus.PRIVATE:
            return "Private";
        case ResourceStatus.SHARED:
            return "Shared";
        default:
            return "Unknown";
    }
};

export const workflowPanelColumns: DataColumns<string, WorkflowPanelFilter> = [
    {
        name: WorkflowPanelColumnNames.NAME,
        selected: true,
        configurable: true,
        sortDirection: SortDirection.ASC,
        filters: [],
        render: (uuid: string) => <RosurceWorkflowName uuid={uuid} />
    },
    {
        name: WorkflowPanelColumnNames.AUTHORISATION,
        selected: true,
        configurable: true,
        filters: [
            {
                name: resourceStatus(ResourceStatus.PUBLIC),
                selected: true,
                type: ResourceStatus.PUBLIC
            },
            {
                name: resourceStatus(ResourceStatus.PRIVATE),
                selected: true,
                type: ResourceStatus.PRIVATE
            },
            {
                name: resourceStatus(ResourceStatus.SHARED),
                selected: true,
                type: ResourceStatus.SHARED
            }
        ],
        render: (uuid: string) => <ResourceWorkflowStatus uuid={uuid} />,
    },
    {
        name: WorkflowPanelColumnNames.LAST_MODIFIED,
        selected: true,
        configurable: true,
        sortDirection: SortDirection.NONE,
        filters: [],
        render: (uuid: string) => <ResourceLastModifiedDate uuid={uuid} />
    },
    {
        name: '',
        selected: true,
        configurable: false,
        filters: [],
        render: (uuid: string) => <ResourceShare uuid={uuid} />
    }
];

type WorkflowPanelProps = WorkflowPanelDataProps & DispatchProp;

export const WorkflowPanel = connect((state: RootState) => ({
    resources: state.resources
}))(
    class extends React.Component<WorkflowPanelProps> {
        render() {
            return <Grid container spacing={16}>
                <Grid item xs={6}>
                    <DataExplorer
                        id={WORKFLOW_PANEL_ID}
                        onRowClick={this.handleRowClick}
                        onRowDoubleClick={this.handleRowDoubleClick}
                        contextMenuColumn={false}
                        dataTableDefaultView={<DataTableDefaultView icon={WorkflowIcon} />} />
                </Grid>
                <Grid item xs={6}>
                    <WorkflowDetailsCard />
                </Grid>
            </Grid>;
        }


        handleRowDoubleClick = (uuid: string) => {
            this.props.dispatch<any>(navigateTo(uuid));
        }

        handleRowClick = (uuid: string) => {
            this.props.dispatch(loadDetailsPanel(uuid));
        }
    }
);