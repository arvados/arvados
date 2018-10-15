// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { SortDirection } from '~/components/data-table/data-column';
import { DataColumns } from '~/components/data-table/data-table';
import { DataTableFilterItem } from '~/components/data-table-filters/data-table-filters';
import { ResourceKind } from '~/models/resource';
import { ContainerRequestState } from '~/models/container-request';
import { resourceLabel } from '~/common/labels';
import { SearchBarAdvanceFormData } from '~/store/search-bar/search-bar-actions';
import { SEARCH_RESULTS_PANEL_ID } from '~/store/search-results-panel/search-results-panel-actions';
import { DataExplorer } from '~/views-components/data-explorer/data-explorer';
import {
    ProcessStatus,
    ResourceFileSize,
    ResourceLastModifiedDate,
    ResourceName,
    ResourceOwner,
    ResourceType
} from '~/views-components/data-explorer/renderers';

export enum SearchResultsPanelColumnNames {
    NAME = "Name",
    PROJECT = "Project",
    STATUS = "Status",
    TYPE = 'Type',
    OWNER = "Owner",
    FILE_SIZE = "File size",
    LAST_MODIFIED = "Last modified"
}

export interface SearchResultsPanelDataProps {
    data: SearchBarAdvanceFormData;
}

export interface SearchResultsPanelActionProps {
    onItemClick: (item: string) => void;
    onContextMenu: (event: React.MouseEvent<HTMLElement>, item: string) => void;
    onDialogOpen: (ownerUuid: string) => void;
    onItemDoubleClick: (item: string) => void;
}

export type SearchResultsPanelProps = SearchResultsPanelDataProps & SearchResultsPanelActionProps;

export interface WorkflowPanelFilter extends DataTableFilterItem {
    type: ResourceKind | ContainerRequestState;
}

export const searchResultsPanelColumns: DataColumns<string, WorkflowPanelFilter> = [
    {
        name: SearchResultsPanelColumnNames.NAME,
        selected: true,
        configurable: true,
        sortDirection: SortDirection.ASC,
        filters: [],
        render: (uuid: string) => <ResourceName uuid={uuid} />
    },
    {
        name: SearchResultsPanelColumnNames.PROJECT,
        selected: true,
        configurable: true,
        filters: [],
        render: uuid => <ResourceFileSize uuid={uuid} />
    },
    {
        name: SearchResultsPanelColumnNames.STATUS,
        selected: true,
        configurable: true,
        filters: [],
        render: uuid => <ProcessStatus uuid={uuid} />
    },
    {
        name: SearchResultsPanelColumnNames.TYPE,
        selected: true,
        configurable: true,
        filters: [
            {
                name: resourceLabel(ResourceKind.COLLECTION),
                selected: true,
                type: ResourceKind.COLLECTION
            },
            {
                name: resourceLabel(ResourceKind.PROCESS),
                selected: true,
                type: ResourceKind.PROCESS
            },
            {
                name: resourceLabel(ResourceKind.PROJECT),
                selected: true,
                type: ResourceKind.PROJECT
            }
        ],
        render: (uuid: string) => <ResourceType uuid={uuid} />,
    },
    {
        name: SearchResultsPanelColumnNames.OWNER,
        selected: true,
        configurable: true,
        filters: [],
        render: uuid => <ResourceOwner uuid={uuid} />
    },
    {
        name: SearchResultsPanelColumnNames.FILE_SIZE,
        selected: true,
        configurable: true,
        filters: [],
        render: uuid => <ResourceFileSize uuid={uuid} />
    },
    {
        name: SearchResultsPanelColumnNames.LAST_MODIFIED,
        selected: true,
        configurable: true,
        sortDirection: SortDirection.NONE,
        filters: [],
        render: uuid => <ResourceLastModifiedDate uuid={uuid} />
    }
];

export const SearchResultsPanelView = (props: SearchResultsPanelProps) => {
    return <DataExplorer
        id={SEARCH_RESULTS_PANEL_ID}
        onRowClick={props.onItemClick}
        onRowDoubleClick={props.onItemDoubleClick}
        onContextMenu={props.onContextMenu}
        contextMenuColumn={true} />;
};