// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { TrashPanelItem } from './trash-panel-item';
import { StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core';
import { DataExplorer } from "~/views-components/data-explorer/data-explorer";
import { DispatchProp, connect } from 'react-redux';
import { DataColumns } from '~/components/data-table/data-table';
import { RouteComponentProps } from 'react-router';
import { RootState } from '~/store/store';
import { DataTableFilterItem } from '~/components/data-table-filters/data-table-filters';
import { SortDirection } from '~/components/data-table/data-column';
import { ResourceKind } from '~/models/resource';
import { resourceLabel } from '~/common/labels';
import { ArvadosTheme } from '~/common/custom-theme';
import { renderName, renderType, renderFileSize, renderDate } from '~/views-components/data-explorer/renderers';
import { TrashIcon } from '~/components/icon/icon';
import { TRASH_PANEL_ID } from "~/store/trash-panel/trash-panel-action";
import { getProperty } from "~/store/properties/properties";
import { PROJECT_PANEL_CURRENT_UUID } from "~/store/project-panel/project-panel-action";
import { openContextMenu, resourceKindToContextMenuKind } from "~/store/context-menu/context-menu-actions";

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

export const trashPanelColumns: DataColumns<TrashPanelItem, TrashPanelFilter> = [
    {
        name: TrashPanelColumnNames.NAME,
        selected: true,
        configurable: true,
        sortDirection: SortDirection.ASC,
        filters: [],
        render: renderName,
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
        render: item => renderType(item.kind),
        width: "125px"
    },
    {
        name: TrashPanelColumnNames.FILE_SIZE,
        selected: true,
        configurable: true,
        sortDirection: SortDirection.NONE,
        filters: [],
        render: item => renderFileSize(item.fileSize),
        width: "50px"
    },
    {
        name: TrashPanelColumnNames.TRASHED_DATE,
        selected: true,
        configurable: true,
        sortDirection: SortDirection.NONE,
        filters: [],
        render: item => renderDate(item.trashAt),
        width: "50px"
    },
    {
        name: TrashPanelColumnNames.TO_BE_DELETED,
        selected: true,
        configurable: true,
        sortDirection: SortDirection.NONE,
        filters: [],
        render: item => renderDate(item.deleteAt),
        width: "50px"
    },
];

interface TrashPanelDataProps {
    currentItemId: string;
}

interface TrashPanelActionProps {
    onItemClick: (item: TrashPanelItem) => void;
    onContextMenu: (event: React.MouseEvent<HTMLElement>, item: TrashPanelItem) => void;
    onDialogOpen: (ownerUuid: string) => void;
    onItemDoubleClick: (item: TrashPanelItem) => void;
    onItemRouteChange: (itemId: string) => void;
}

type TrashPanelProps = TrashPanelDataProps & TrashPanelActionProps & DispatchProp & WithStyles<CssRules>;

export const TrashPanel = withStyles(styles)(
    connect((state: RootState) => ({
        currentItemId: getProperty(PROJECT_PANEL_CURRENT_UUID)(state.properties),
        resources: state.resources
    }))(
        class extends React.Component<TrashPanelProps> {
            render() {
                return <DataExplorer
                    id={TRASH_PANEL_ID}
                    onRowClick={this.props.onItemClick}
                    onRowDoubleClick={this.props.onItemDoubleClick}
                    onContextMenu={this.props.onContextMenu}
                    extractKey={(item: TrashPanelItem) => item.uuid}
                    defaultIcon={TrashIcon}
                    defaultMessages={['Your trash list is empty.']}/>
                ;
            }

            handleContextMenu = (event: React.MouseEvent<HTMLElement>, resourceUuid: string) => {
                const kind = resourceKindToContextMenuKind(resourceUuid);
                if (kind) {
                    this.props.dispatch<any>(openContextMenu(event, { name: '', uuid: resourceUuid, kind }));
                }
            }
        }
    )
);
