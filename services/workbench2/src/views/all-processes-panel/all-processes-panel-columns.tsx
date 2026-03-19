// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { DataColumns, SortDirection } from "components/data-table/data-column";
import {
    ProcessStatus,
    ResourceName,
    ResourceOwnerWithName,
    ResourceType,
    ContainerRunTime,
    ResourceCreatedAtDate
} from "views-components/data-explorer/renderers";
import { ContainerRequestResource } from "models/container-request";
import { createTree } from "models/tree";
import { getInitialProcessStatusFilters, getInitialProcessTypeFilters } from "store/resource-type-filters/resource-type-filters";


export enum AllProcessesPanelColumnNames {
    NAME = "Name",
    STATUS = "Status",
    TYPE = "Type",
    OWNER = "Owner",
    CREATED_AT = "Created at",
    RUNTIME = "Run Time"
}

export const allProcessesPanelColumns: DataColumns<string, ContainerRequestResource> = [
    {
        name: AllProcessesPanelColumnNames.NAME,
        selected: true,
        configurable: true,
        sort: { direction: SortDirection.NONE, field: "name" },
        filters: createTree(),
        render: uuid => <ResourceName uuid={uuid} />,
    },
    {
        name: AllProcessesPanelColumnNames.STATUS,
        selected: true,
        configurable: true,
        mutuallyExclusiveFilters: true,
        filters: getInitialProcessStatusFilters(),
        render: uuid => <ProcessStatus uuid={uuid} />,
    },
    {
        name: AllProcessesPanelColumnNames.TYPE,
        selected: true,
        configurable: true,
        filters: getInitialProcessTypeFilters(),
        render: uuid => <ResourceType uuid={uuid} />,
    },
    {
        name: AllProcessesPanelColumnNames.OWNER,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <ResourceOwnerWithName uuid={uuid} />,
    },
    {
        name: AllProcessesPanelColumnNames.CREATED_AT,
        selected: true,
        configurable: true,
        sort: { direction: SortDirection.DESC, field: "createdAt" },
        filters: createTree(),
        render: uuid => <ResourceCreatedAtDate uuid={uuid} />,
    },
    {
        name: AllProcessesPanelColumnNames.RUNTIME,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <ContainerRunTime uuid={uuid} />,
    },
];
