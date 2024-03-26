// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core';
import { DataExplorer } from "views-components/data-explorer/data-explorer";
import { connect, DispatchProp } from 'react-redux';
import { RootState } from 'store/store';
import { ArvadosTheme } from 'common/custom-theme';
import { ShareMeIcon } from 'components/icon/icon';
import { ResourcesState, getResource } from 'store/resources/resources';
import { ResourceKind } from 'models/resource';
import { navigateTo } from "store/navigation/navigation-action";
import { loadDetailsPanel } from "store/details-panel/details-panel-action";
import { SHARED_WITH_ME_PANEL_ID } from 'store/shared-with-me-panel/shared-with-me-panel-actions';
import {
    openContextMenu,
    resourceUuidToContextMenuKind
} from 'store/context-menu/context-menu-actions';
import {
    ResourceName,
    ProcessStatus as ResourceStatus,
    ResourceType,
    ResourceOwnerWithNameLink,
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
import { DataTableFilterItem } from 'components/data-table-filters/data-table-filters';
import { GroupContentsResource } from 'services/groups-service/groups-service';
import { toggleOne, deselectAllOthers } from 'store/multiselect/multiselect-actions';
import { DataColumns } from 'components/data-table/data-table';
import { ContainerRequestState } from 'models/container-request';
import { ProjectResource } from 'models/project';
import { createTree } from 'models/tree';
import { SortDirection } from 'components/data-table/data-column';
import { getInitialResourceTypeFilters, getInitialProcessStatusFilters } from 'store/resource-type-filters/resource-type-filters';

type CssRules = "toolbar" | "button" | "root";

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    toolbar: {
        paddingBottom: theme.spacing.unit * 3,
        textAlign: "right"
    },
    button: {
        marginLeft: theme.spacing.unit
    },
    root: {
        width: '100%',
    },
});

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

export interface ProjectPanelFilter extends DataTableFilterItem {
    type: ResourceKind | ContainerRequestState;
}

export const sharedWithMePanelColumns: DataColumns<string, ProjectResource> = [
    {
        name: SharedWithMePanelColumnNames.NAME,
        selected: true,
        configurable: true,
        sort: { direction: SortDirection.NONE, field: 'name' },
        filters: createTree(),
        render: (uuid) => <ResourceName uuid={uuid} />,
    },
    {
        name: SharedWithMePanelColumnNames.STATUS,
        selected: true,
        configurable: true,
        mutuallyExclusiveFilters: true,
        filters: getInitialProcessStatusFilters(),
        render: (uuid) => <ResourceStatus uuid={uuid} />,
    },
    {
        name: SharedWithMePanelColumnNames.TYPE,
        selected: true,
        configurable: true,
        filters: getInitialResourceTypeFilters(),
        render: (uuid) => <ResourceType uuid={uuid} />,
    },
    {
        name: SharedWithMePanelColumnNames.OWNER,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: (uuid) => <ResourceOwnerWithNameLink uuid={uuid} />,
    },
    {
        name: SharedWithMePanelColumnNames.PORTABLE_DATA_HASH,
        selected: false,
        configurable: true,
        filters: createTree(),
        render: (uuid) => <ResourcePortableDataHash uuid={uuid} />,
    },
    {
        name: SharedWithMePanelColumnNames.FILE_SIZE,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: (uuid) => <ResourceFileSize uuid={uuid} />,
    },
    {
        name: SharedWithMePanelColumnNames.FILE_COUNT,
        selected: false,
        configurable: true,
        filters: createTree(),
        render: (uuid) => <ResourceFileCount uuid={uuid} />,
    },
    {
        name: SharedWithMePanelColumnNames.UUID,
        selected: false,
        configurable: true,
        filters: createTree(),
        render: (uuid) => <ResourceUUID uuid={uuid} />,
    },
    {
        name: SharedWithMePanelColumnNames.CONTAINER_UUID,
        selected: false,
        configurable: true,
        filters: createTree(),
        render: (uuid) => <ResourceContainerUuid uuid={uuid} />,
    },
    {
        name: SharedWithMePanelColumnNames.RUNTIME,
        selected: false,
        configurable: true,
        filters: createTree(),
        render: (uuid) => <ContainerRunTime uuid={uuid} />,
    },
    {
        name: SharedWithMePanelColumnNames.OUTPUT_UUID,
        selected: false,
        configurable: true,
        filters: createTree(),
        render: (uuid) => <ResourceOutputUuid uuid={uuid} />,
    },
    {
        name: SharedWithMePanelColumnNames.LOG_UUID,
        selected: false,
        configurable: true,
        filters: createTree(),
        render: (uuid) => <ResourceLogUuid uuid={uuid} />,
    },
    {
        name: SharedWithMePanelColumnNames.PARENT_PROCESS,
        selected: false,
        configurable: true,
        filters: createTree(),
        render: (uuid) => <ResourceParentProcess uuid={uuid} />,
    },
    {
        name: SharedWithMePanelColumnNames.MODIFIED_BY_USER_UUID,
        selected: false,
        configurable: true,
        filters: createTree(),
        render: (uuid) => <ResourceModifiedByUserUuid uuid={uuid} />,
    },
    {
        name: SharedWithMePanelColumnNames.VERSION,
        selected: false,
        configurable: true,
        filters: createTree(),
        render: (uuid) => <ResourceVersion uuid={uuid} />,
    },
    {
        name: SharedWithMePanelColumnNames.CREATED_AT,
        selected: false,
        configurable: true,
        sort: { direction: SortDirection.NONE, field: 'createdAt' },
        filters: createTree(),
        render: (uuid) => <ResourceCreatedAtDate uuid={uuid} />,
    },
    {
        name: SharedWithMePanelColumnNames.LAST_MODIFIED,
        selected: true,
        configurable: true,
        sort: { direction: SortDirection.DESC, field: 'modifiedAt' },
        filters: createTree(),
        render: (uuid) => <ResourceLastModifiedDate uuid={uuid} />,
    },
    {
        name: SharedWithMePanelColumnNames.TRASH_AT,
        selected: false,
        configurable: true,
        sort: { direction: SortDirection.NONE, field: 'trashAt' },
        filters: createTree(),
        render: (uuid) => <ResourceTrashDate uuid={uuid} />,
    },
    {
        name: SharedWithMePanelColumnNames.DELETE_AT,
        selected: false,
        configurable: true,
        sort: { direction: SortDirection.NONE, field: 'deleteAt' },
        filters: createTree(),
        render: (uuid) => <ResourceDeleteDate uuid={uuid} />,
    },
];


interface SharedWithMePanelDataProps {
    resources: ResourcesState;
    userUuid: string;
}

type SharedWithMePanelProps = SharedWithMePanelDataProps & DispatchProp & WithStyles<CssRules>;

export const SharedWithMePanel = withStyles(styles)(
    connect((state: RootState) => ({
        resources: state.resources,
        userUuid: state.auth.user!.uuid,
    }))(
        class extends React.Component<SharedWithMePanelProps> {
            render() {
                return <div className={this.props.classes.root}><DataExplorer
                    id={SHARED_WITH_ME_PANEL_ID}
                    onRowClick={this.handleRowClick}
                    onRowDoubleClick={this.handleRowDoubleClick}
                    onContextMenu={this.handleContextMenu}
                    contextMenuColumn={false}
                    defaultViewIcon={ShareMeIcon}
                    defaultViewMessages={['No shared items']} />
                </div>;
            }

            handleContextMenu = (event: React.MouseEvent<HTMLElement>, resourceUuid: string) => {
                const { resources } = this.props;
                const resource = getResource<GroupContentsResource>(resourceUuid)(resources);
                const menuKind = this.props.dispatch<any>(resourceUuidToContextMenuKind(resourceUuid));
                if (menuKind && resource) {
                    this.props.dispatch<any>(openContextMenu(event, {
                        name: resource.name,
                        uuid: resource.uuid,
                        description: resource.description,
                        ownerUuid: resource.ownerUuid,
                        isTrashed: ('isTrashed' in resource) ? resource.isTrashed: false,
                        kind: resource.kind,
                        menuKind
                    }));
                }
                this.props.dispatch<any>(loadDetailsPanel(resourceUuid));
            }

            handleRowDoubleClick = (uuid: string) => {
                this.props.dispatch<any>(navigateTo(uuid));
            }

            handleRowClick = (uuid: string) => {
                this.props.dispatch<any>(toggleOne(uuid))
                this.props.dispatch<any>(deselectAllOthers(uuid))
                this.props.dispatch<any>(loadDetailsPanel(uuid));
            }
        }
    )
);
