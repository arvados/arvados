// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { LINK_PANEL_ID } from '~/store/link-panel/link-panel-actions';
import { DataExplorer } from '~/views-components/data-explorer/data-explorer';
import { SortDirection } from '~/components/data-table/data-column';
import { DataColumns } from '~/components/data-table/data-table';
import { DataTableDefaultView } from '~/components/data-table-default-view/data-table-default-view';
import { ResourcesState } from '~/store/resources/resources';
import { ShareMeIcon } from '~/components/icon/icon';
import { createTree } from '~/models/tree';
import { 
    ResourceLinkUuid, ResourceLinkHead, ResourceLinkTail, 
    ResourceLinkClass, ResourceLinkName } 
from '~/views-components/data-explorer/renderers';

export enum LinkPanelColumnNames {
    NAME = "Name",
    LINK_CLASS = "Link Class",
    TAIL = "Tail",
    HEAD = 'Head',
    UUID = "UUID"
}

export const linkPanelColumns: DataColumns<string> = [
    {
        name: LinkPanelColumnNames.NAME,
        selected: true,
        configurable: true,
        sortDirection: SortDirection.NONE,
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

export interface LinkPanelRootDataProps {
    resources: ResourcesState;
}

export interface LinkPanelRootActionProps {
    onItemClick: (item: string) => void;
    onContextMenu: (event: React.MouseEvent<HTMLElement>, item: string) => void;
    onItemDoubleClick: (item: string) => void;
}

export type LinkPanelRootProps = LinkPanelRootDataProps & LinkPanelRootActionProps;

export const LinkPanelRoot = (props: LinkPanelRootProps) => {
    return <DataExplorer
        id={LINK_PANEL_ID}
        onRowClick={props.onItemClick}
        onRowDoubleClick={props.onItemDoubleClick}
        onContextMenu={props.onContextMenu}
        contextMenuColumn={true} 
        hideColumnSelector
        hideSearchInput
        dataTableDefaultView={
            <DataTableDefaultView
                icon={ShareMeIcon}
                messages={['Your link list is empty.']} />
        }/>;
};