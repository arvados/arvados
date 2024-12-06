// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { DataColumns } from 'components/data-table/data-column';
import {
    ProcessStatus as ResourceStatus,
    ContainerRunTime,
    renderType,
    RenderName,
    renderPortableDataHash,
    RenderOwnerName,
    renderFileSize,
    renderFileCount,
    renderUuidWithCopy,
    renderModifiedByUserUuid,
    renderVersion,
    renderCreatedAtDate,
    renderLastModifiedDate,
    renderTrashDate,
    renderDeleteDate,
    renderContainerUuid,
    ResourceOutputUuid,
    ResourceLogUuid,
    renderResourceParentProcess,
} from 'views-components/data-explorer/renderers';
import { ProjectResource } from 'models/project';
import { CollectionResource } from 'models/collection';
import { createTree } from 'models/tree';
import { SortDirection } from 'components/data-table/data-column';
import { getInitialResourceTypeFilters, getInitialProcessStatusFilters } from 'store/resource-type-filters/resource-type-filters';
import { GroupContentsResource } from 'services/groups-service/groups-service';
import { ContainerRequestResource } from 'models/container-request';

export enum SharedWithMePanelColumnNames {
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

export const sharedWithMePanelColumns: DataColumns<ProjectResource | CollectionResource> = [
    {
        name: SharedWithMePanelColumnNames.NAME,
        selected: true,
        configurable: true,
        sort: { direction: SortDirection.NONE, field: 'name' },
        filters: createTree(),
        render: (resource: ProjectResource | CollectionResource) => <RenderName resource={resource} />,
    },
    {
        name: SharedWithMePanelColumnNames.STATUS,
        selected: true,
        configurable: true,
        mutuallyExclusiveFilters: true,
        filters: getInitialProcessStatusFilters(),
        render: (resource: ProjectResource) => <ResourceStatus uuid={resource.uuid} />,
    },
    {
        name: SharedWithMePanelColumnNames.TYPE,
        selected: true,
        configurable: true,
        filters: getInitialResourceTypeFilters(),
        render: (resource: ProjectResource) => renderType(resource),
    },
    {
        name: SharedWithMePanelColumnNames.OWNER,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: (resource: ProjectResource) => <RenderOwnerName resource={resource} link={true} />,
    },
    {
        name: SharedWithMePanelColumnNames.PORTABLE_DATA_HASH,
        selected: false,
        configurable: true,
        filters: createTree(),
        render: (resource: ProjectResource) => renderPortableDataHash(resource),
    },
    {
        name: SharedWithMePanelColumnNames.FILE_SIZE,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: (resource: ProjectResource) => renderFileSize(resource),
    },
    {
        name: SharedWithMePanelColumnNames.FILE_COUNT,
        selected: false,
        configurable: true,
        filters: createTree(),
        render: (resource: ProjectResource) => renderFileCount(resource),
    },
    {
        name: SharedWithMePanelColumnNames.UUID,
        selected: false,
        configurable: true,
        filters: createTree(),
        render: (resource: ProjectResource) => renderUuidWithCopy({uuid: resource.uuid}),
    },
    {
        name: SharedWithMePanelColumnNames.CONTAINER_UUID,
        selected: false,
        configurable: true,
        filters: createTree(),
        render: (resource: GroupContentsResource) => renderContainerUuid(resource),
    },
    {
        name: SharedWithMePanelColumnNames.RUNTIME,
        selected: false,
        configurable: true,
        filters: createTree(),
        render: (resource: GroupContentsResource) => <ContainerRunTime uuid={resource.uuid} />,
    },
    {
        name: SharedWithMePanelColumnNames.OUTPUT_UUID,
        selected: false,
        configurable: true,
        filters: createTree(),
        render: (resource: ContainerRequestResource) => <ResourceOutputUuid resource={resource} />,
    },
    {
        name: SharedWithMePanelColumnNames.LOG_UUID,
        selected: false,
        configurable: true,
        filters: createTree(),
        render: (resource: ContainerRequestResource) => <ResourceLogUuid resource={resource} />,
    },
    {
        name: SharedWithMePanelColumnNames.PARENT_PROCESS,
        selected: false,
        configurable: true,
        filters: createTree(),
        render: (resource: GroupContentsResource) => renderResourceParentProcess(resource),
    },
    {
        name: SharedWithMePanelColumnNames.MODIFIED_BY_USER_UUID,
        selected: false,
        configurable: true,
        filters: createTree(),
        render: (resource: ProjectResource) => renderModifiedByUserUuid(resource),
    },
    {
        name: SharedWithMePanelColumnNames.VERSION,
        selected: false,
        configurable: true,
        filters: createTree(),
        render: (resource: CollectionResource) => renderVersion(resource),
    },
    {
        name: SharedWithMePanelColumnNames.CREATED_AT,
        selected: false,
        configurable: true,
        sort: { direction: SortDirection.NONE, field: 'createdAt' },
        filters: createTree(),
        render: (resource: ProjectResource) => renderCreatedAtDate(resource),
    },
    {
        name: SharedWithMePanelColumnNames.LAST_MODIFIED,
        selected: true,
        configurable: true,
        sort: { direction: SortDirection.DESC, field: 'modifiedAt' },
        filters: createTree(),
        render: (resource: ProjectResource) => renderLastModifiedDate(resource),
    },
    {
        name: SharedWithMePanelColumnNames.TRASH_AT,
        selected: false,
        configurable: true,
        sort: { direction: SortDirection.NONE, field: 'trashAt' },
        filters: createTree(),
        render: (resource: ProjectResource) => renderTrashDate(resource),
    },
    {
        name: SharedWithMePanelColumnNames.DELETE_AT,
        selected: false,
        configurable: true,
        sort: { direction: SortDirection.NONE, field: 'deleteAt' },
        filters: createTree(),
        render: (resource: ProjectResource) => renderDeleteDate(resource),
    },
];
