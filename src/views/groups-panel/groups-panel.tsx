// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import withStyles from "@material-ui/core/styles/withStyles";
import { DispatchProp, connect } from 'react-redux';
import { RouteComponentProps } from 'react-router';
import { StyleRulesCallback, WithStyles, Typography } from "@material-ui/core";

import { DataExplorer } from "~/views-components/data-explorer/data-explorer";
import { DataColumns } from '~/components/data-table/data-table';
import { RootState } from '~/store/store';
import { DataTableFilterItem } from '~/components/data-table-filters/data-table-filters';
import { ContainerRequestState } from '~/models/container-request';
import { SortDirection } from '~/components/data-table/data-column';
import { ResourceKind, Resource } from '~/models/resource';
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
import { DataTableDefaultView } from '~/components/data-table-default-view/data-table-default-view';
import { ArvadosTheme } from "~/common/custom-theme";
import { createTree } from '~/models/tree';
import { getInitialResourceTypeFilters } from '~/store/resource-type-filters/resource-type-filters';
import { GROUPS_PANEL_ID } from '~/store/groups-panel/groups-panel-actions';
import { noop } from 'lodash/fp';
import { GroupResource } from '~/models/group';

export enum ProjectPanelColumnNames {
    GROUP = "Name",
    OWNER = "Owner",
    MEMBERS = "Members",
}

export const groupsPanelColumns: DataColumns<string> = [
    {
        name: ProjectPanelColumnNames.GROUP,
        selected: true,
        configurable: true,
        sortDirection: SortDirection.ASC,
        filters: createTree(),
        render: uuid => <ResourceName uuid={uuid} />
    },
    {
        name: ProjectPanelColumnNames.OWNER,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <ResourceOwner uuid={uuid} />,
    },
    {
        name: ProjectPanelColumnNames.MEMBERS,
        selected: true,
        configurable: true,
        filters: createTree(),
        render: uuid => <span>0</span>,
    },
];

export class GroupsPanel extends React.Component {

    render() {
        return (
            <DataExplorer
                id={GROUPS_PANEL_ID}
                onRowClick={noop}
                onRowDoubleClick={noop}
                onContextMenu={noop}
                contextMenuColumn={true} />
        );
    }
}
