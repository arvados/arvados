// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { IconButton, StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core';
import { DataExplorer } from "~/views-components/data-explorer/data-explorer";
import { connect, DispatchProp } from 'react-redux';
import { DataColumns } from '~/components/data-table/data-table';
import { RootState } from '~/store/store';
import { DataTableFilterItem } from '~/components/data-table-filters/data-table-filters';
import { SortDirection } from '~/components/data-table/data-column';
import { ResourceKind, TrashableResource } from '~/models/resource';
import { resourceLabel } from '~/common/labels';
import { ArvadosTheme } from '~/common/custom-theme';
import { RestoreFromTrashIcon, TrashIcon } from '~/components/icon/icon';
import { TRASH_PANEL_ID } from "~/store/trash-panel/trash-panel-action";
import { getProperty } from "~/store/properties/properties";
import { PROJECT_PANEL_CURRENT_UUID } from "~/store/project-panel/project-panel-action";
import { openContextMenu } from "~/store/context-menu/context-menu-actions";
import { getResource, ResourcesState } from "~/store/resources/resources";
import {
    ResourceDeleteDate,
    ResourceFileSize,
    ResourceName,
    ResourceTrashDate,
    ResourceType
} from "~/views-components/data-explorer/renderers";
import { navigateTo } from "~/store/navigation/navigation-action";
import { loadDetailsPanel } from "~/store/details-panel/details-panel-action";
import { toggleTrashed } from "~/store/trash/trash-actions";
import { ContextMenuKind } from "~/views-components/context-menu/context-menu";
import { Dispatch } from "redux";
import { PanelDefaultView } from '~/components/panel-default-view/panel-default-view';
import { DataTableDefaultView } from '~/components/data-table-default-view/data-table-default-view';

type CssRules = "toolbar" | "button";

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    toolbar: {
        paddingBottom: theme.spacing.unit * 3,
        textAlign: "right"
    },
    button: {
        marginLeft: theme.spacing.unit
    },
});

export enum TrashPanelColumnNames {
    NAME = "Name",
    TYPE = "Type",
    FILE_SIZE = "File size",
    TRASHED_DATE = "Trashed date",
    TO_BE_DELETED = "To be deleted"
}

export interface TrashPanelFilter extends DataTableFilterItem {
    type: ResourceKind;
}

export const ResourceRestore =
    connect((state: RootState, props: { uuid: string, dispatch?: Dispatch<any> }) => {
        const resource = getResource<TrashableResource>(props.uuid)(state.resources);
        return { resource, dispatch: props.dispatch };
    })((props: { resource?: TrashableResource, dispatch?: Dispatch<any> }) =>
        <IconButton onClick={() => {
            if (props.resource && props.dispatch) {
                props.dispatch(toggleTrashed(
                    props.resource.kind,
                    props.resource.uuid,
                    props.resource.ownerUuid,
                    props.resource.isTrashed
                ));
            }
        }}>
            <RestoreFromTrashIcon />
        </IconButton>
    );

export const trashPanelColumns: DataColumns<string, TrashPanelFilter> = [
    {
        name: TrashPanelColumnNames.NAME,
        selected: true,
        configurable: true,
        sortDirection: SortDirection.ASC,
        filters: [],
        render: uuid => <ResourceName uuid={uuid} />,
        width: "450px"
    },
    {
        name: TrashPanelColumnNames.TYPE,
        selected: true,
        configurable: true,
        sortDirection: SortDirection.NONE,
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
        render: uuid => <ResourceType uuid={uuid} />,
        width: "125px"
    },
    {
        name: TrashPanelColumnNames.FILE_SIZE,
        selected: true,
        configurable: true,
        sortDirection: SortDirection.NONE,
        filters: [],
        render: uuid => <ResourceFileSize uuid={uuid} />,
        width: "50px"
    },
    {
        name: TrashPanelColumnNames.TRASHED_DATE,
        selected: true,
        configurable: true,
        sortDirection: SortDirection.NONE,
        filters: [],
        render: uuid => <ResourceTrashDate uuid={uuid} />,
        width: "50px"
    },
    {
        name: TrashPanelColumnNames.TO_BE_DELETED,
        selected: true,
        configurable: true,
        sortDirection: SortDirection.NONE,
        filters: [],
        render: uuid => <ResourceDeleteDate uuid={uuid} />,
        width: "50px"
    },
    {
        name: '',
        selected: true,
        configurable: false,
        sortDirection: SortDirection.NONE,
        filters: [],
        render: uuid => <ResourceRestore uuid={uuid} />,
        width: "50px"
    }
];

interface TrashPanelDataProps {
    currentItemId: string;
    resources: ResourcesState;
}

type TrashPanelProps = TrashPanelDataProps & DispatchProp & WithStyles<CssRules>;

export const TrashPanel = withStyles(styles)(
    connect((state: RootState) => ({
        currentItemId: getProperty(PROJECT_PANEL_CURRENT_UUID)(state.properties),
        resources: state.resources
    }))(
        class extends React.Component<TrashPanelProps> {
            render() {
                return this.hasAnyTrashedResources()
                    ? <DataExplorer
                        id={TRASH_PANEL_ID}
                        onRowClick={this.handleRowClick}
                        onRowDoubleClick={this.handleRowDoubleClick}
                        onContextMenu={this.handleContextMenu}
                        contextMenuColumn={false}
                        dataTableDefaultView={<DataTableDefaultView icon={TrashIcon}/>} />
                    : <PanelDefaultView
                        icon={TrashIcon}
                        messages={['Your trash list is empty.']} />;
            }

            hasAnyTrashedResources = () => {
                // TODO: implement check if there is anything in the trash,
                //       without taking pagination into the account
                return true;
            }

            handleContextMenu = (event: React.MouseEvent<HTMLElement>, resourceUuid: string) => {
                const resource = getResource<TrashableResource>(resourceUuid)(this.props.resources);
                if (resource) {
                    this.props.dispatch<any>(openContextMenu(event, {
                        name: '',
                        uuid: resource.uuid,
                        ownerUuid: resource.ownerUuid,
                        isTrashed: resource.isTrashed,
                        kind: resource.kind,
                        menuKind: ContextMenuKind.TRASH
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
    )
);
