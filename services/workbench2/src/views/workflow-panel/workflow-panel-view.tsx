// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { DataExplorer } from "views-components/data-explorer/data-explorer";
import { WorkflowIcon } from 'components/icon/icon';
import { WORKFLOW_PANEL_ID } from 'store/workflow-panel/workflow-panel-actions';
import { DataTableFilterItem } from 'components/data-table-filters/data-table-filters';
import { Grid, Paper } from '@mui/material';
import { WorkflowDetailsCard } from './workflow-description-card';
import { WorkflowResource } from 'models/workflow';

export interface WorkflowPanelFilter extends DataTableFilterItem {
    type: ResourceStatus;
}

export interface WorkflowPanelDataProps {
    uuid?: string;
    workflows?: WorkflowResource[];
}

export interface WorfklowPanelActionProps {
    handleRowDoubleClick: (workflowUuid: string) => void;
    handleRowClick: (workflowUuid: string) => void;
}

export type WorkflowPanelProps = WorkflowPanelDataProps & WorfklowPanelActionProps;

export enum ResourceStatus {
    PUBLIC = "Public",
    PRIVATE = "Private",
    SHARED = "Shared"
}

export const WorkflowPanelView = (props: WorkflowPanelProps) => {
    const workflow = props.uuid ? props.workflows?.find(workflow => workflow.uuid === props.uuid) : undefined;
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
                <WorkflowDetailsCard workflow={workflow} />
            </Paper>
        </Grid>
    </Grid>;
};
