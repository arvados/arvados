// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { DataExplorer } from "~/views-components/data-explorer/data-explorer";
import { DispatchProp, connect } from 'react-redux';
import { DataColumns } from '~/components/data-table/data-table';
import { RouteComponentProps } from 'react-router';
import { RootState } from '~/store/store';
import { DataTableFilterItem } from '~/components/data-table-filters/data-table-filters';
import { ContainerRequestState } from '~/models/container-request';
import { SortDirection } from '~/components/data-table/data-column';
import { ResourceKind, Resource } from '~/models/resource';
import { resourceLabel } from '~/common/labels';
import { ResourceFileSize, ResourceLastModifiedDate, ProcessStatus, ResourceType, ResourceOwner } from '~/views-components/data-explorer/renderers';
import { ProjectIcon } from '~/components/icon/icon';
import { ResourceName } from '~/views-components/data-explorer/renderers';
import { ResourcesState, getResource } from '~/store/resources/resources';
import { loadDetailsPanel } from '~/store/details-panel/details-panel-action';
import { resourceKindToContextMenuKind, openContextMenu } from '~/store/context-menu/context-menu-actions';
import { ProjectResource } from '~/models/project';
import { navigateTo } from '~/store/navigation/navigation-action';
import { getProperty } from '~/store/properties/properties';
import { PROJECT_PANEL_CURRENT_UUID } from '~/store/project-panel/project-panel-action';
import { filterResources } from '~/store/resources/resources';
import { PanelDefaultView } from '~/components/panel-default-view/panel-default-view';
import { DataTableDefaultView } from '~/components/data-table-default-view/data-table-default-view';

export enum ProjectPanelColumnNames {
    NAME = "Name",
    STATUS = "Status",
    TYPE = "Type",
    OWNER = "Owner",
    FILE_SIZE = "File size",
    LAST_MODIFIED = "Last modified"
}

export interface ProjectPanelFilter extends DataTableFilterItem {
    type: ResourceKind | ContainerRequestState;
}

export const projectPanelColumns: DataColumns<string, ProjectPanelFilter> = [
    {
        name: ProjectPanelColumnNames.NAME,
        selected: true,
        configurable: true,
        sortDirection: SortDirection.ASC,
        filters: [],
        render: uuid => <ResourceName uuid={uuid} />
    },
    {
        name: "Status",
        selected: true,
        configurable: true,
        filters: [],
        render: uuid => <ProcessStatus uuid={uuid} />,
    },
    {
        name: ProjectPanelColumnNames.TYPE,
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
        render: uuid => <ResourceType uuid={uuid} />
    },
    {
        name: ProjectPanelColumnNames.OWNER,
        selected: true,
        configurable: true,
        filters: [],
        render: uuid => <ResourceOwner uuid={uuid} />
    },
    {
        name: ProjectPanelColumnNames.FILE_SIZE,
        selected: true,
        configurable: true,
        filters: [],
        render: uuid => <ResourceFileSize uuid={uuid} />
    },
    {
        name: ProjectPanelColumnNames.LAST_MODIFIED,
        selected: true,
        configurable: true,
        sortDirection: SortDirection.NONE,
        filters: [],
        render: uuid => <ResourceLastModifiedDate uuid={uuid} />
    }
];

export const PROJECT_PANEL_ID = "projectPanel";

interface ProjectPanelDataProps {
    currentItemId: string;
    resources: ResourcesState;
}

type ProjectPanelProps = ProjectPanelDataProps & DispatchProp & RouteComponentProps<{ id: string }>;

export const ProjectPanel = connect((state: RootState) => ({
        currentItemId: getProperty(PROJECT_PANEL_CURRENT_UUID)(state.properties),
        resources: state.resources
    }))(
        class extends React.Component<ProjectPanelProps> {
            render() {
                return this.hasAnyItems()
                    ? <DataExplorer
                        id={PROJECT_PANEL_ID}
                        onRowClick={this.handleRowClick}
                        onRowDoubleClick={this.handleRowDoubleClick}
                        onContextMenu={this.handleContextMenu}
                        contextMenuColumn={true}
                        dataTableDefaultView={<DataTableDefaultView icon={ProjectIcon}/>} />
                    : <PanelDefaultView
                        icon={ProjectIcon}
                        messages={['Your project is empty.', 'Please create a project or create a collection and upload a data.']} />;
            }

            hasAnyItems = () => {
                const resources = filterResources(this.isCurrentItemChild)(this.props.resources);
                return resources.length > 0;
            }

            isCurrentItemChild = (resource: Resource) => {
                return resource.ownerUuid === this.props.currentItemId;
            }

            handleContextMenu = (event: React.MouseEvent<HTMLElement>, resourceUuid: string) => {
                const menuKind = resourceKindToContextMenuKind(resourceUuid);
                const resource = getResource<ProjectResource>(resourceUuid)(this.props.resources);
                if (menuKind && resource) {
                    this.props.dispatch<any>(openContextMenu(event, {
                        name: resource.name,
                        uuid: resource.uuid,
                        ownerUuid: resource.ownerUuid,
                        isTrashed: resource.isTrashed,
                        kind: resource.kind,
                        menuKind
                    }));
                }
            }

            handleRowDoubleClick = (uuid: string) => {
                this.props.dispatch<any>(navigateTo(uuid));
            }

            handleRowClick = (uuid: string) => {
                this.props.dispatch(loadDetailsPanel(uuid));
            }

        }
    );
