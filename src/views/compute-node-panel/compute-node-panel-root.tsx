// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { ShareMeIcon } from '~/components/icon/icon';
import { DataExplorer } from '~/views-components/data-explorer/data-explorer';
import { DataTableDefaultView } from '~/components/data-table-default-view/data-table-default-view';
import { COMPUTE_NODE_PANEL_ID } from '~/store/compute-nodes/compute-nodes-actions';
import { DataColumns } from '~/components/data-table/data-table';
import { SortDirection } from '~/components/data-table/data-column';
import { createTree } from '~/models/tree';
import { 
    ResourceUuid, ResourceNodeInfo, ResourceNodeDomain, ResourceNodeHostname, ResourceNodeJobUuid,
    ResourceNodeFirstPingAt, ResourceNodeLastPingAt, ResourceNodeIpAddress
} from '~/views-components/data-explorer/renderers';
import { ResourcesState } from '~/store/resources/resources';

export enum ComputeNodePanelColumnNames {
    INFO = 'Info',
    UUID = 'UUID',
    DOMAIN = 'Domain',
    FIRST_PING_AT = 'First ping at',
    HOSTNAME = 'Hostname',
    IP_ADDRESS = 'IP Address',
    JOB = 'Job',
    LAST_PING_AT = 'Last ping at'
}

export const computeNodePanelColumns: DataColumns<string> = [
    {
        name: ComputeNodePanelColumnNames.INFO,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <ResourceNodeInfo uuid={uuid} />
    },
    {
        name: ComputeNodePanelColumnNames.UUID,
        selected: true,
        configurable: true,
        sortDirection: SortDirection.NONE,
        filters: createTree(),
        render: uuid => <ResourceUuid uuid={uuid} />
    },
    {
        name: ComputeNodePanelColumnNames.DOMAIN,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <ResourceNodeDomain uuid={uuid} />
    },
    {
        name: ComputeNodePanelColumnNames.FIRST_PING_AT,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <ResourceNodeFirstPingAt uuid={uuid} />
    },
    {
        name: ComputeNodePanelColumnNames.HOSTNAME,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <ResourceNodeHostname uuid={uuid} />
    },
    {
        name: ComputeNodePanelColumnNames.IP_ADDRESS,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <ResourceNodeIpAddress uuid={uuid} />
    },
    {
        name: ComputeNodePanelColumnNames.JOB,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <ResourceNodeJobUuid uuid={uuid} />
    },
    {
        name: ComputeNodePanelColumnNames.LAST_PING_AT,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <ResourceNodeLastPingAt uuid={uuid} />
    }
];

export interface ComputeNodePanelRootActionProps {
    onItemClick: (item: string) => void;
    onContextMenu: (event: React.MouseEvent<HTMLElement>, item: string) => void;
    onItemDoubleClick: (item: string) => void;
}

export interface ComputeNodePanelRootDataProps {
    resources: ResourcesState;
}

type ComputeNodePanelRootProps = ComputeNodePanelRootActionProps & ComputeNodePanelRootDataProps;

export const ComputeNodePanelRoot = (props: ComputeNodePanelRootProps) => {
    return <DataExplorer
        id={COMPUTE_NODE_PANEL_ID}
        onRowClick={props.onItemClick}
        onRowDoubleClick={props.onItemDoubleClick}
        onContextMenu={props.onContextMenu}
        contextMenuColumn={true}
        hideColumnSelector
        hideSearchInput
        dataTableDefaultView={
            <DataTableDefaultView
                icon={ShareMeIcon}
                messages={['Your compute node list is empty.']} />
        } />;
};