// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { ProjectIcon } from "components/icon/icon";
import { PROJECT_PANEL_RUN_ID } from "store/project-panel/project-panel-action-bind";
import { DataExplorer } from "views-components/data-explorer/data-explorer";
import { ProjectResource } from 'models/project';
import { DataColumns, SortDirection } from "components/data-table/data-column";
import { createTree } from "models/tree";
import {
    ContainerRunTime,
    renderType,
    RenderName,
    RenderOwnerName,
    renderUuidWithCopy,
    renderModifiedByUserUuid,
    renderCreatedAtDate,
    renderLastModifiedDate,
    renderTrashDate,
    renderDeleteDate,
    renderResourceStatus,
    renderContainerUuid,
    ResourceOutputUuid,
    ResourceLogUuid,
    renderResourceParentProcess,
} from "views-components/data-explorer/renderers";
import { getInitialProcessStatusFilters, getInitialProcessTypeFilters } from "store/resource-type-filters/resource-type-filters";
import { SubprocessProgressBar } from "components/subprocess-progress-bar/subprocess-progress-bar";
import { connect } from "react-redux";
import { RootState } from "store/store";
import { getProjectPanelCurrentUuid } from "store/project-panel/project-panel";
import { getResource } from "store/resources/resources";
import { GroupContentsResource } from "services/groups-service/groups-service";
import { ContainerRequestResource } from "models/container-request";

export enum ProjectPanelRunColumnNames {
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

export const projectPanelRunColumns: DataColumns<ProjectResource> = [
    {
        name: ProjectPanelRunColumnNames.NAME,
        selected: true,
        configurable: true,
        sort: { direction: SortDirection.NONE, field: 'name' },
        filters: createTree(),
        render: (resource) => <RenderName resource={resource} />,
    },
    {
        name: ProjectPanelRunColumnNames.STATUS,
        selected: true,
        configurable: true,
        mutuallyExclusiveFilters: true,
        filters: getInitialProcessStatusFilters(),
        render: (resource) => renderResourceStatus(resource),
    },
    {
        name: ProjectPanelRunColumnNames.TYPE,
        selected: true,
        configurable: true,
        filters: getInitialProcessTypeFilters(),
        render: (resource) => renderType(resource),
    },
    {
        name: ProjectPanelRunColumnNames.OWNER,
        selected: false,
        configurable: true,
        filters: createTree(),
        render: (resource) => <RenderOwnerName resource={resource} />,
    },
    {
        name: ProjectPanelRunColumnNames.UUID,
        selected: false,
        configurable: true,
        filters: createTree(),
        render: (resource) => renderUuidWithCopy({uuid: resource.uuid}),
    },
    {
        name: ProjectPanelRunColumnNames.CONTAINER_UUID,
        selected: false,
        configurable: true,
        filters: createTree(),
        render: (resource: GroupContentsResource) => renderContainerUuid(resource),
    },
    {
        name: ProjectPanelRunColumnNames.RUNTIME,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: (resource) => <ContainerRunTime uuid={resource.uuid} />,
    },
    {
        name: ProjectPanelRunColumnNames.OUTPUT_UUID,
        selected: false,
        configurable: true,
        filters: createTree(),
        render: (resource: ContainerRequestResource) => <ResourceOutputUuid resource={resource} />,
    },
    {
        name: ProjectPanelRunColumnNames.LOG_UUID,
        selected: false,
        configurable: true,
        filters: createTree(),
        render: (resource: ContainerRequestResource) => <ResourceLogUuid resource={resource} />,
    },
    {
        name: ProjectPanelRunColumnNames.PARENT_PROCESS,
        selected: false,
        configurable: true,
        filters: createTree(),
        render: (resource) => renderResourceParentProcess(resource),
    },
    {
        name: ProjectPanelRunColumnNames.MODIFIED_BY_USER_UUID,
        selected: false,
        configurable: true,
        filters: createTree(),
        render: (resource) => renderModifiedByUserUuid(resource),
    },
    {
        name: ProjectPanelRunColumnNames.CREATED_AT,
        selected: false,
        configurable: true,
        sort: { direction: SortDirection.NONE, field: 'createdAt' },
        filters: createTree(),
        render: (resource) => renderCreatedAtDate(resource),
    },
    {
        name: ProjectPanelRunColumnNames.LAST_MODIFIED,
        selected: true,
        configurable: true,
        sort: { direction: SortDirection.DESC, field: 'modifiedAt' },
        filters: createTree(),
        render: (resource) => renderLastModifiedDate(resource),
    },
    {
        name: ProjectPanelRunColumnNames.TRASH_AT,
        selected: false,
        configurable: true,
        sort: { direction: SortDirection.NONE, field: 'trashAt' },
        filters: createTree(),
        render: (resource) => renderTrashDate(resource),
    },
    {
        name: ProjectPanelRunColumnNames.DELETE_AT,
        selected: false,
        configurable: true,
        sort: { direction: SortDirection.NONE, field: 'deleteAt' },
        filters: createTree(),
        render: (resource) => renderDeleteDate(resource),
    },
];

const DEFAULT_VIEW_MESSAGES = ['No workflow runs found'];

interface ProjectPanelRunProps {
    project?: ProjectResource;
    paperClassName?: string;
    onRowClick: (item: ContainerRequestResource) => void;
    onRowDoubleClick: ({uuid}: ContainerRequestResource) => void;
    onContextMenu: (event: React.MouseEvent<HTMLElement>, resource: ContainerRequestResource) => void;
}

const mapStateToProps = (state: RootState): Pick<ProjectPanelRunProps, 'project'> => {
    const projectUuid = getProjectPanelCurrentUuid(state) || "";
    const project = getResource<ProjectResource>(projectUuid)(state.resources);
    return {
        project,
    };
};

export const ProjectPanelRun = connect(mapStateToProps)((props: ProjectPanelRunProps) => {
    return <DataExplorer
        id={PROJECT_PANEL_RUN_ID}
        onRowClick={props.onRowClick}
        onRowDoubleClick={props.onRowDoubleClick}
        onContextMenu={props.onContextMenu}
        contextMenuColumn={false}
        defaultViewIcon={ProjectIcon}
        defaultViewMessages={DEFAULT_VIEW_MESSAGES}
        progressBar={<SubprocessProgressBar parentResource={props.project} dataExplorerId={PROJECT_PANEL_RUN_ID} />}
        paperClassName={props.paperClassName}
    />;
});
