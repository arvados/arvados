// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { DataColumns, SortDirection } from 'components/data-table/data-column';
import { GroupMembersCount, ResourceUuid } from 'views-components/data-explorer/renderers';
import { ResourceName } from 'views-components/data-explorer/renderers';
import { createTree } from 'models/tree';
import { GroupResource } from 'models/group';


export enum GroupsPanelColumnNames {
    GROUP = "Name",
    UUID = "UUID",
    MEMBERS = "Members"
}

export const groupsPanelColumns: DataColumns<string, GroupResource> = [
    {
        name: GroupsPanelColumnNames.GROUP,
        selected: true,
        configurable: true,
        sort: { direction: SortDirection.ASC, field: "name" },
        filters: createTree(),
        render: uuid => <ResourceName uuid={uuid} />
    },
    {
        name: GroupsPanelColumnNames.UUID,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <ResourceUuid uuid={uuid} />,
    },
    {
        name: GroupsPanelColumnNames.MEMBERS,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <GroupMembersCount uuid={uuid} />,
    },
];
