// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import withStyles from '@material-ui/core/styles/withStyles';
import { DispatchProp, connect } from 'react-redux';
import { RouteComponentProps } from 'react-router';
import { StyleRulesCallback, WithStyles } from '@material-ui/core';

import { DataExplorer } from 'views-components/data-explorer/data-explorer';
import { DataColumns } from 'components/data-table/data-table';
import { RootState } from 'store/store';
import { DataTableFilterItem } from 'components/data-table-filters/data-table-filters';
import { ContainerRequestState } from 'models/container-request';
import { SortDirection } from 'components/data-table/data-column';
import { ResourceKind, Resource } from 'models/resource';
import {
    ResourceName,
    ProcessStatus as ResourceStatus,
    ResourceType,
    ResourceOwnerWithName,
    ResourcePortableDataHash,
    ResourceFileSize,
    ResourceFileCount,
    ResourceUUID,
    ResourceContainerUuid,
    ContainerRunTime,
    ResourceOutputUuid,
    ResourceLogUuid,
    ResourceParentProcess,
    ResourceModifiedByUserUuid,
    ResourceVersion,
    ResourceCreatedAtDate,
    ResourceLastModifiedDate,
    ResourceTrashDate,
    ResourceDeleteDate,
} from 'views-components/data-explorer/renderers';
import { ProjectIcon } from 'components/icon/icon';
import { ResourcesState, getResource } from 'store/resources/resources';
import { loadDetailsPanel } from 'store/details-panel/details-panel-action';
import { openContextMenu, resourceUuidToContextMenuKind } from 'store/context-menu/context-menu-actions';
import { navigateTo } from 'store/navigation/navigation-action';
import { getProperty } from 'store/properties/properties';
import { PROJECT_PANEL_CURRENT_UUID } from 'store/project-panel/project-panel-action';
import { ArvadosTheme } from 'common/custom-theme';
import { createTree } from 'models/tree';
import { getInitialResourceTypeFilters, getInitialProcessStatusFilters } from 'store/resource-type-filters/resource-type-filters';
import { GroupContentsResource } from 'services/groups-service/groups-service';
import { GroupClass, GroupResource } from 'models/group';
import { CollectionResource } from 'models/collection';
import { resourceIsFrozen } from 'common/frozen-resources';
import { ProjectResource } from 'models/project';
import { deselectAllOthers, toggleOne } from 'store/multiselect/multiselect-actions';

type CssRules = 'root' | 'button' ;

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        width: '100%',
    },
    button: {
        marginLeft: theme.spacing.unit,
    },
});

export enum ProjectPanelColumnNames {
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

export interface ProjectPanelFilter extends DataTableFilterItem {
    type: ResourceKind | ContainerRequestState;
}

export const projectPanelColumns: DataColumns<string, ProjectResource> = [
    {
        name: ProjectPanelColumnNames.NAME,
        selected: true,
        configurable: true,
        sort: { direction: SortDirection.NONE, field: 'name' },
        filters: createTree(),
        render: (uuid) => <ResourceName uuid={uuid} />,
    },
    {
        name: ProjectPanelColumnNames.STATUS,
        selected: true,
        configurable: true,
        mutuallyExclusiveFilters: true,
        filters: getInitialProcessStatusFilters(),
        render: (uuid) => <ResourceStatus uuid={uuid} />,
    },
    {
        name: ProjectPanelColumnNames.TYPE,
        selected: true,
        configurable: true,
        filters: getInitialResourceTypeFilters(),
        render: (uuid) => <ResourceType uuid={uuid} />,
    },
    {
        name: ProjectPanelColumnNames.OWNER,
        selected: false,
        configurable: true,
        filters: createTree(),
        render: (uuid) => <ResourceOwnerWithName uuid={uuid} />,
    },
    {
        name: ProjectPanelColumnNames.PORTABLE_DATA_HASH,
        selected: false,
        configurable: true,
        filters: createTree(),
        render: (uuid) => <ResourcePortableDataHash uuid={uuid} />,
    },
    {
        name: ProjectPanelColumnNames.FILE_SIZE,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: (uuid) => <ResourceFileSize uuid={uuid} />,
    },
    {
        name: ProjectPanelColumnNames.FILE_COUNT,
        selected: false,
        configurable: true,
        filters: createTree(),
        render: (uuid) => <ResourceFileCount uuid={uuid} />,
    },
    {
        name: ProjectPanelColumnNames.UUID,
        selected: false,
        configurable: true,
        filters: createTree(),
        render: (uuid) => <ResourceUUID uuid={uuid} />,
    },
    {
        name: ProjectPanelColumnNames.CONTAINER_UUID,
        selected: false,
        configurable: true,
        filters: createTree(),
        render: (uuid) => <ResourceContainerUuid uuid={uuid} />,
    },
    {
        name: ProjectPanelColumnNames.RUNTIME,
        selected: false,
        configurable: true,
        filters: createTree(),
        render: (uuid) => <ContainerRunTime uuid={uuid} />,
    },
    {
        name: ProjectPanelColumnNames.OUTPUT_UUID,
        selected: false,
        configurable: true,
        filters: createTree(),
        render: (uuid) => <ResourceOutputUuid uuid={uuid} />,
    },
    {
        name: ProjectPanelColumnNames.LOG_UUID,
        selected: false,
        configurable: true,
        filters: createTree(),
        render: (uuid) => <ResourceLogUuid uuid={uuid} />,
    },
    {
        name: ProjectPanelColumnNames.PARENT_PROCESS,
        selected: false,
        configurable: true,
        filters: createTree(),
        render: (uuid) => <ResourceParentProcess uuid={uuid} />,
    },
    {
        name: ProjectPanelColumnNames.MODIFIED_BY_USER_UUID,
        selected: false,
        configurable: true,
        filters: createTree(),
        render: (uuid) => <ResourceModifiedByUserUuid uuid={uuid} />,
    },
    {
        name: ProjectPanelColumnNames.VERSION,
        selected: false,
        configurable: true,
        filters: createTree(),
        render: (uuid) => <ResourceVersion uuid={uuid} />,
    },
    {
        name: ProjectPanelColumnNames.CREATED_AT,
        selected: false,
        configurable: true,
        sort: { direction: SortDirection.NONE, field: 'createdAt' },
        filters: createTree(),
        render: (uuid) => <ResourceCreatedAtDate uuid={uuid} />,
    },
    {
        name: ProjectPanelColumnNames.LAST_MODIFIED,
        selected: true,
        configurable: true,
        sort: { direction: SortDirection.DESC, field: 'modifiedAt' },
        filters: createTree(),
        render: (uuid) => <ResourceLastModifiedDate uuid={uuid} />,
    },
    {
        name: ProjectPanelColumnNames.TRASH_AT,
        selected: false,
        configurable: true,
        sort: { direction: SortDirection.NONE, field: 'trashAt' },
        filters: createTree(),
        render: (uuid) => <ResourceTrashDate uuid={uuid} />,
    },
    {
        name: ProjectPanelColumnNames.DELETE_AT,
        selected: false,
        configurable: true,
        sort: { direction: SortDirection.NONE, field: 'deleteAt' },
        filters: createTree(),
        render: (uuid) => <ResourceDeleteDate uuid={uuid} />,
    },
];

export const PROJECT_PANEL_ID = 'projectPanel';

const DEFAULT_VIEW_MESSAGES = ['Your project is empty.', 'Please create a project or create a collection and upload a data.'];

interface ProjectPanelDataProps {
    currentItemId: string;
    resources: ResourcesState;
    project: GroupResource;
    isAdmin: boolean;
    userUuid: string;
    dataExplorerItems: any;
    working: boolean;
}

type ProjectPanelProps = ProjectPanelDataProps & DispatchProp & WithStyles<CssRules> & RouteComponentProps<{ id: string }>;

const mapStateToProps = (state: RootState) => {
    const currentItemId = getProperty<string>(PROJECT_PANEL_CURRENT_UUID)(state.properties);
    const project = getResource<GroupResource>(currentItemId || "")(state.resources);
    return {
        currentItemId,
        project,
        resources: state.resources,
        userUuid: state.auth.user!.uuid,
    };
}

export const ProjectPanel = withStyles(styles)(
    connect(mapStateToProps)(
        class extends React.Component<ProjectPanelProps> {

            render() {
                const { classes } = this.props;
                return <div data-cy='project-panel' className={classes.root}>
                    <DataExplorer
                        id={PROJECT_PANEL_ID}
                        onRowClick={this.handleRowClick}
                        onRowDoubleClick={this.handleRowDoubleClick}
                        onContextMenu={this.handleContextMenu}
                        contextMenuColumn={true}
                        defaultViewIcon={ProjectIcon}
                        defaultViewMessages={DEFAULT_VIEW_MESSAGES}
                    />
                </div>
            }

            isCurrentItemChild = (resource: Resource) => {
                return resource.ownerUuid === this.props.currentItemId;
            };

            handleContextMenu = (event: React.MouseEvent<HTMLElement>, resourceUuid: string) => {
                const { resources, isAdmin } = this.props;
                const resource = getResource<GroupContentsResource>(resourceUuid)(resources);
                // When viewing the contents of a filter group, all contents should be treated as read only.
                let readonly = false;
                const project = getResource<GroupResource>(this.props.currentItemId)(resources);
                if (project && project.groupClass === GroupClass.FILTER) {
                    readonly = true;
                }

                const menuKind = this.props.dispatch<any>(resourceUuidToContextMenuKind(resourceUuid, readonly));
                if (menuKind && resource) {
                    this.props.dispatch<any>(
                        openContextMenu(event, {
                            name: resource.name,
                            uuid: resource.uuid,
                            ownerUuid: resource.ownerUuid,
                            isTrashed: 'isTrashed' in resource ? resource.isTrashed : false,
                            kind: resource.kind,
                            menuKind,
                            isAdmin,
                            isFrozen: resourceIsFrozen(resource, resources),
                            description: resource.description,
                            storageClassesDesired: (resource as CollectionResource).storageClassesDesired,
                            properties: 'properties' in resource ? resource.properties : {},
                        })
                    );
                }
                this.props.dispatch<any>(loadDetailsPanel(resourceUuid));
            };

            handleRowDoubleClick = (uuid: string) => {
                this.props.dispatch<any>(navigateTo(uuid));
            };

            handleRowClick = (uuid: string) => {
                this.props.dispatch<any>(toggleOne(uuid))
                this.props.dispatch<any>(deselectAllOthers(uuid))
                this.props.dispatch<any>(loadDetailsPanel(uuid));
            };
        }
    )
);
