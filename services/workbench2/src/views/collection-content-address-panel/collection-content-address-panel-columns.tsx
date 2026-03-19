// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { DataColumns, SortDirection } from 'components/data-table/data-column';
import { createTree } from 'models/tree';
import {
    ResourceName,
    ResourceOwnerName,
    ResourceLastModifiedDate,
    ResourceStatus
} from 'views-components/data-explorer/renderers';
import { CollectionResource } from 'models/collection';

enum CollectionContentAddressPanelColumnNames {
    COLLECTION_WITH_THIS_ADDRESS = "Collection with this address",
    STATUS = "Status",
    LOCATION = "Location",
    LAST_MODIFIED = "Last modified"
}

export const collectionContentAddressPanelColumns: DataColumns<string, CollectionResource> = [
    {
        name: CollectionContentAddressPanelColumnNames.COLLECTION_WITH_THIS_ADDRESS,
        selected: true,
        configurable: true,
        sort: { direction: SortDirection.NONE, field: "uuid" },
        filters: createTree(),
        render: uuid => <ResourceName uuid={uuid} />
    },
    {
        name: CollectionContentAddressPanelColumnNames.STATUS,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <ResourceStatus uuid={uuid} />
    },
    {
        name: CollectionContentAddressPanelColumnNames.LOCATION,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <ResourceOwnerName uuid={uuid} />
    },
    {
        name: CollectionContentAddressPanelColumnNames.LAST_MODIFIED,
        selected: true,
        configurable: true,
        sort: { direction: SortDirection.DESC, field: "modifiedAt" },
        filters: createTree(),
        render: uuid => <ResourceLastModifiedDate uuid={uuid} />
    }
];
