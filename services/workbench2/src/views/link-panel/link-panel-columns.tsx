// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { DataColumns, SortDirection } from 'components/data-table/data-column';
import { createTree } from 'models/tree';
import {
    ResourceLinkUuid, ResourceLinkHead, ResourceLinkTail,
    ResourceLinkClass, ResourceLinkName
} from 'views-components/data-explorer/renderers';
import { LinkResource } from 'models/link';


export enum LinkPanelColumnNames {
    NAME = "Name",
    LINK_CLASS = "Link Class",
    TAIL = "Tail",
    HEAD = 'Head',
    UUID = "UUID"
}

export const linkPanelColumns: DataColumns<string, LinkResource> = [
    {
        name: LinkPanelColumnNames.NAME,
        selected: true,
        configurable: true,
        sort: { direction: SortDirection.NONE, field: "name" },
        filters: createTree(),
        render: uuid => <ResourceLinkName uuid={uuid} />
    },
    {
        name: LinkPanelColumnNames.LINK_CLASS,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <ResourceLinkClass uuid={uuid} />
    },
    {
        name: LinkPanelColumnNames.TAIL,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <ResourceLinkTail uuid={uuid} />
    },
    {
        name: LinkPanelColumnNames.HEAD,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <ResourceLinkHead uuid={uuid} />
    },
    {
        name: LinkPanelColumnNames.UUID,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <ResourceLinkUuid uuid={uuid} />
    }
];
