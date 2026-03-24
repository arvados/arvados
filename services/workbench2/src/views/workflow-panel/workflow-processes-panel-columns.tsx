// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { DataColumns, SortDirection } from 'components/data-table/data-column';
import { ResourceCreatedAtDate, ProcessStatus, ContainerRunTime } from 'views-components/data-explorer/renderers';
import { ResourceName } from 'views-components/data-explorer/renderers';
import { createTree } from 'models/tree';
import { getInitialProcessStatusFilters } from 'store/resource-type-filters/resource-type-filters';
import { ProcessResource } from 'models/process';


export enum WorkflowProcessesPanelColumnNames {
    NAME = "Name",
    STATUS = "Status",
    CREATED_AT = "Created At",
    RUNTIME = "Run Time"
}

export const workflowProcessesPanelColumns: DataColumns<string, ProcessResource> = [
    {
        name: WorkflowProcessesPanelColumnNames.NAME,
        selected: true,
        configurable: true,
        sort: { direction: SortDirection.NONE, field: "name" },
        filters: createTree(),
        render: uuid => <ResourceName uuid={uuid} />
    },
    {
        name: WorkflowProcessesPanelColumnNames.STATUS,
        selected: true,
        configurable: true,
        mutuallyExclusiveFilters: true,
        filters: getInitialProcessStatusFilters(),
        render: uuid => <ProcessStatus uuid={uuid} />,
    },
    {
        name: WorkflowProcessesPanelColumnNames.CREATED_AT,
        selected: true,
        configurable: true,
        sort: { direction: SortDirection.DESC, field: "createdAt" },
        filters: createTree(),
        render: uuid => <ResourceCreatedAtDate uuid={uuid} />
    },
    {
        name: WorkflowProcessesPanelColumnNames.RUNTIME,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <ContainerRunTime uuid={uuid} />
    }
];
