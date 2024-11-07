// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { ProjectIcon } from "components/icon/icon";
import { PROJECT_PANEL_DATA_ID } from "store/project-panel/project-panel-action-bind";
import { DataExplorer } from "views-components/data-explorer/data-explorer";
import { ProjectResource } from 'models/project';
import { DataColumns, SortDirection } from "components/data-table/data-column";
import { createTree } from "models/tree";
import {
    ResourceCreatedAtDate,
    ResourceDeleteDate,
    ResourceFileCount,
    ResourceFileSize,
    ResourceLastModifiedDate,
    ResourceModifiedByUserUuid,
    ResourceTrashDate,
    ResourceUUID,
    ResourceVersion,
    renderName,
    renderType,
    OwnerWithName,
    renderPortableDataHash,
} from "views-components/data-explorer/renderers";
import { getInitialDataResourceTypeFilters } from "store/resource-type-filters/resource-type-filters";

export enum ProjectPanelDataColumnNames {
    NAME = 'Name',
    STATUS = 'Status',
    TYPE = 'Type',
    OWNER = 'Owner',
    PORTABLE_DATA_HASH = 'Portable Data Hash',
    FILE_SIZE = 'File Size',
    FILE_COUNT = 'File Count',
    UUID = 'UUID',
    CONTAINER_UUID = 'Container UUID',
    RUNTIME = 'Runtime',
    OUTPUT_UUID = 'Output UUID',
    LOG_UUID = 'Log UUID',
    PARENT_PROCESS = 'Parent Process UUID',
    MODIFIED_BY_USER_UUID = 'Modified by User UUID',
    VERSION = 'Version',
    CREATED_AT = 'Date Created',
    LAST_MODIFIED = 'Last Modified',
    TRASH_AT = 'Trash at',
    DELETE_AT = 'Delete at',
}

export const projectPanelDataColumns: DataColumns<string, ProjectResource> = [
    {
        name: ProjectPanelDataColumnNames.NAME,
        selected: true,
        configurable: true,
        sort: { direction: SortDirection.NONE, field: 'name' },
        filters: createTree(),
        render: (resource)=> renderName(resource as ProjectResource),
    },
    {
        name: ProjectPanelDataColumnNames.TYPE,
        selected: true,
        configurable: true,
        filters: getInitialDataResourceTypeFilters(),
        render: (resource) => renderType(resource as ProjectResource),
    },
    {
        name: ProjectPanelDataColumnNames.OWNER,
        selected: false,
        configurable: true,
        filters: createTree(),
        render: (resource) => <OwnerWithName resource={resource as ProjectResource} />,
    },
    {
        name: ProjectPanelDataColumnNames.PORTABLE_DATA_HASH,
        selected: false,
        configurable: true,
        filters: createTree(),
        render: (resource) => renderPortableDataHash(resource as ProjectResource),
    },
    {
        name: ProjectPanelDataColumnNames.FILE_SIZE,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: (uuid) => <ResourceFileSize uuid={uuid as string} />,
    },
    {
        name: ProjectPanelDataColumnNames.FILE_COUNT,
        selected: false,
        configurable: true,
        filters: createTree(),
        render: (uuid) => <ResourceFileCount uuid={uuid as string} />,
    },
    {
        name: ProjectPanelDataColumnNames.UUID,
        selected: false,
        configurable: true,
        filters: createTree(),
        render: (uuid) => <ResourceUUID uuid={uuid as string} />,
    },
    {
        name: ProjectPanelDataColumnNames.MODIFIED_BY_USER_UUID,
        selected: false,
        configurable: true,
        filters: createTree(),
        render: (uuid) => <ResourceModifiedByUserUuid uuid={uuid as string} />,
    },
    {
        name: ProjectPanelDataColumnNames.VERSION,
        selected: false,
        configurable: true,
        filters: createTree(),
        render: (uuid) => <ResourceVersion uuid={uuid as string} />,
    },
    {
        name: ProjectPanelDataColumnNames.CREATED_AT,
        selected: false,
        configurable: true,
        sort: { direction: SortDirection.NONE, field: 'createdAt' },
        filters: createTree(),
        render: (uuid) => <ResourceCreatedAtDate uuid={uuid as string} />,
    },
    {
        name: ProjectPanelDataColumnNames.LAST_MODIFIED,
        selected: true,
        configurable: true,
        sort: { direction: SortDirection.DESC, field: 'modifiedAt' },
        filters: createTree(),
        render: (uuid) => <ResourceLastModifiedDate uuid={uuid as string} />,
    },
    {
        name: ProjectPanelDataColumnNames.TRASH_AT,
        selected: false,
        configurable: true,
        sort: { direction: SortDirection.NONE, field: 'trashAt' },
        filters: createTree(),
        render: (uuid) => <ResourceTrashDate uuid={uuid as string} />,
    },
    {
        name: ProjectPanelDataColumnNames.DELETE_AT,
        selected: false,
        configurable: true,
        sort: { direction: SortDirection.NONE, field: 'deleteAt' },
        filters: createTree(),
        render: (uuid) => <ResourceDeleteDate uuid={uuid as string} />,
    },
];

const DEFAULT_VIEW_MESSAGES = ['No data found'];

interface ProjectPanelDataProps {
    paperClassName?: string;
    onRowClick: (item: ProjectResource) => void;
    onRowDoubleClick: (uuid: string) => void;
    onContextMenu: (event: React.MouseEvent<HTMLElement>, resourceUuid: string) => void;
};

export const ProjectPanelData = class extends React.Component<ProjectPanelDataProps> {
    render () {
        return <DataExplorer
            id={PROJECT_PANEL_DATA_ID}
            onRowClick={this.props.onRowClick}
            onRowDoubleClick={this.props.onRowDoubleClick}
            onContextMenu={this.props.onContextMenu}
            contextMenuColumn={false}
            defaultViewIcon={ProjectIcon}
            defaultViewMessages={DEFAULT_VIEW_MESSAGES}
            paperClassName={this.props.paperClassName}
        />;
    }
};
