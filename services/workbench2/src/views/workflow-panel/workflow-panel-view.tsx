// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { DataExplorer } from "views-components/data-explorer/data-explorer";
import { WorkflowIcon } from 'components/icon/icon';
import { WORKFLOW_PANEL_ID } from 'store/workflow-panel/workflow-panel-actions';
import {
    ResourceWorkflowStatus,
    ResourceShare,
    renderLastModifiedDate,
    renderWorkflowName,
    ResourceRunProcess,
} from "views-components/data-explorer/renderers";
import { DataColumns, SortDirection } from 'components/data-table/data-column';
import { DataTableFilterItem } from 'components/data-table-filters/data-table-filters';
import { Grid, Paper } from '@mui/material';
import { WorkflowDetailsCard } from './workflow-description-card';
import { WorkflowResource } from 'models/workflow';
import { createTree } from 'models/tree';

export enum WorkflowPanelColumnNames {
    NAME = "Name",
    AUTHORISATION = "Authorisation",
    LAST_MODIFIED = "Last modified",
    SHARE = 'Share'
}

export interface WorkflowPanelFilter extends DataTableFilterItem {
    type: ResourceStatus;
}

export interface WorkflowPanelDataProps {
    workflow?: WorkflowResource;
}

export interface WorfklowPanelActionProps {
    handleRowDoubleClick: (workflow: WorkflowResource) => void;
    handleRowClick: (workflow: WorkflowResource) => void;
}

export type WorkflowPanelProps = WorkflowPanelDataProps & WorfklowPanelActionProps;

export enum ResourceStatus {
    PUBLIC = "Public",
    PRIVATE = "Private",
    SHARED = "Shared"
}

// TODO: restore filters
// const resourceStatus = (type: string) => {
//     switch (type) {
//         case ResourceStatus.PUBLIC:
//             return "Public";
//         case ResourceStatus.PRIVATE:
//             return "Private";
//         case ResourceStatus.SHARED:
//             return "Shared";
//         default:
//             return "Unknown";
//     }
// };

export const workflowPanelColumns: DataColumns<WorkflowResource> = [
    {
        name: WorkflowPanelColumnNames.NAME,
        selected: true,
        configurable: true,
        sort: {direction: SortDirection.ASC, field: "name"},
        filters: createTree(),
        render: (resource) => renderWorkflowName(resource),
    },
    {
        name: WorkflowPanelColumnNames.AUTHORISATION,
        selected: true,
        configurable: true,
        filters: createTree(),
        // TODO: restore filters
        // filters: [
        //     {
        //         name: resourceStatus(ResourceStatus.PUBLIC),
        //         selected: true,
        //         type: ResourceStatus.PUBLIC
        //     },
        //     {
        //         name: resourceStatus(ResourceStatus.PRIVATE),
        //         selected: true,
        //         type: ResourceStatus.PRIVATE
        //     },
        //     {
        //         name: resourceStatus(ResourceStatus.SHARED),
        //         selected: true,
        //         type: ResourceStatus.SHARED
        //     }
        // ],
        render: (resource) => <ResourceWorkflowStatus resource={resource}/>,
    },
    {
        name: WorkflowPanelColumnNames.LAST_MODIFIED,
        selected: true,
        configurable: true,
        sort: {direction: SortDirection.NONE, field: "modifiedAt"},
        filters: createTree(),
        render: (resource) => renderLastModifiedDate(resource),
    },
    {
        name: '',
        selected: true,
        configurable: false,
        filters: createTree(),
        render: (resource) => <ResourceShare resource={resource} />
    },
    {
        name: '',
        selected: true,
        configurable: false,
        filters: createTree(),
        render: (resource) => <ResourceRunProcess uuid={resource.uuid} />
    }
];

export const WorkflowPanelView = (props: WorkflowPanelProps) => {
    return <Grid container spacing={2} style={{ minHeight: '500px' }}>
        <Grid item xs={6}>
            <DataExplorer
                id={WORKFLOW_PANEL_ID}
                onRowClick={props.handleRowClick}
                onRowDoubleClick={props.handleRowDoubleClick}
                contextMenuColumn={false}
                onContextMenu={e => e}
                defaultViewIcon={WorkflowIcon}
                defaultViewMessages={['Workflow list is empty.']} />
        </Grid>
        <Grid item xs={6}>
            <Paper style={{ height: '100%' }}>
                <WorkflowDetailsCard workflow={props.workflow} />
            </Paper>
        </Grid>
    </Grid>;
};
