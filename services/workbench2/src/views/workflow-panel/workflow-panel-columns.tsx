// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import {
    ResourceLastModifiedDate,
    ResourceWorkflowName,
    ResourceWorkflowStatus,
    ResourceShare,
    ResourceRunProcess
} from "views-components/data-explorer/renderers";
import { DataColumns, SortDirection } from 'components/data-table/data-column';
import { WorkflowResource } from 'models/workflow';
import { createTree } from 'models/tree';

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

export enum WorkflowPanelColumnNames {
    NAME = "Name",
    AUTHORISATION = "Authorisation",
    LAST_MODIFIED = "Last modified",
    SHARE = 'Share'
}

export const workflowPanelColumns: DataColumns<string, WorkflowResource> = [
    {
        name: WorkflowPanelColumnNames.NAME,
        selected: true,
        configurable: true,
        sort: { direction: SortDirection.ASC, field: "name" },
        filters: createTree(),
        render: (uuid: string) => <ResourceWorkflowName uuid={uuid} />
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
        render: (uuid: string) => <ResourceWorkflowStatus uuid={uuid} />,
    },
    {
        name: WorkflowPanelColumnNames.LAST_MODIFIED,
        selected: true,
        configurable: true,
        sort: { direction: SortDirection.NONE, field: "modifiedAt" },
        filters: createTree(),
        render: (uuid: string) => <ResourceLastModifiedDate uuid={uuid} />
    },
    {
        name: '',
        selected: true,
        configurable: false,
        filters: createTree(),
        render: (uuid: string) => <ResourceShare uuid={uuid} />
    },
    {
        name: '',
        selected: true,
        configurable: false,
        filters: createTree(),
        render: (uuid: string) => <ResourceRunProcess uuid={uuid} />
    }
];
