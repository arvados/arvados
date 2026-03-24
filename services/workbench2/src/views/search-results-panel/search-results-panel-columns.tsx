// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { DataColumns, SortDirection } from 'components/data-table/data-column';
import {
    ResourceCluster,
    ResourceFileSize,
    ResourceLastModifiedDate,
    ResourceName,
    ResourceOwnerWithName,
    ResourceStatus,
    ResourceType
} from 'views-components/data-explorer/renderers';
import { createTree } from 'models/tree';
import { getInitialSearchTypeFilters } from 'store/resource-type-filters/resource-type-filters';
import { GroupContentsResource } from 'services/groups-service/groups-service';

export enum SearchResultsPanelColumnNames {
    CLUSTER = "Cluster",
    NAME = "Name",
    STATUS = "Status",
    TYPE = 'Type',
    OWNER = "Owner",
    FILE_SIZE = "File size",
    LAST_MODIFIED = "Last modified"
}

export const searchResultsPanelColumns: DataColumns<string, GroupContentsResource> = [
    {
        name: SearchResultsPanelColumnNames.CLUSTER,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: (uuid: string) => <ResourceCluster uuid={uuid} />
    },
    {
        name: SearchResultsPanelColumnNames.NAME,
        selected: true,
        configurable: true,
        sort: { direction: SortDirection.NONE, field: "name" },
        filters: createTree(),
        render: (uuid: string) => <ResourceName uuid={uuid} />
    },
    {
        name: SearchResultsPanelColumnNames.STATUS,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <ResourceStatus uuid={uuid} />
    },
    {
        name: SearchResultsPanelColumnNames.TYPE,
        selected: true,
        configurable: true,
        filters: getInitialSearchTypeFilters(),
        render: (uuid: string) => <ResourceType uuid={uuid} />,
    },
    {
        name: SearchResultsPanelColumnNames.OWNER,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <ResourceOwnerWithName uuid={uuid} />
    },
    {
        name: SearchResultsPanelColumnNames.FILE_SIZE,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <ResourceFileSize uuid={uuid} />
    },
    {
        name: SearchResultsPanelColumnNames.LAST_MODIFIED,
        selected: true,
        configurable: true,
        sort: { direction: SortDirection.DESC, field: "modifiedAt" },
        filters: createTree(),
        render: uuid => <ResourceLastModifiedDate uuid={uuid} />
    }
];
