// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { DataExplorer } from "views-components/data-explorer/data-explorer";
import { connect, DispatchProp } from 'react-redux';
import { RootState } from 'store/store';
import { DataTableFilterItem } from 'components/data-table-filters/data-table-filters';
import { DataColumns, SortDirection } from 'components/data-table/data-column';
import { ResourceKind, TrashableResource } from 'models/resource';
import { ArvadosTheme } from 'common/custom-theme';
import { TrashIcon } from 'components/icon/icon';
import { TRASH_PANEL_ID } from "store/trash-panel/trash-panel-action";
import { getProperty } from "store/properties/properties";
import { PROJECT_PANEL_CURRENT_UUID } from "store/project-panel/project-panel";
import { openContextMenu } from "store/context-menu/context-menu-actions";
import { getResource, ResourcesState } from "store/resources/resources";
import {
    renderType,
    renderName,
    renderFileSize,
    renderTrashDate,
    renderDeleteDate,
    renderRestoreFromTrash,
} from "views-components/data-explorer/renderers";
import { navigateTo } from "store/navigation/navigation-action";
import { loadDetailsPanel } from "store/details-panel/details-panel-action";
import { ContextMenuKind } from 'views-components/context-menu/menu-item-sort';
import { createTree } from 'models/tree';
import {
    getTrashPanelTypeFilters
} from 'store/resource-type-filters/resource-type-filters';
import { CollectionResource } from 'models/collection';
import { toggleOne, deselectAllOthers } from 'store/multiselect/multiselect-actions';

type CssRules = "toolbar" | "button" | "root";

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    toolbar: {
        paddingBottom: theme.spacing(3),
        textAlign: "right"
    },
    button: {
        marginLeft: theme.spacing(1)
    },
    root: {
        width: '100%',
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

export const trashPanelColumns: DataColumns<string, CollectionResource> = [
    {
        name: TrashPanelColumnNames.NAME,
        selected: true,
        configurable: true,
        sort: {direction: SortDirection.NONE, field: "name"},
        filters: createTree(),
        render: (resource) => renderName(resource as CollectionResource),
    },
    {
        name: TrashPanelColumnNames.TYPE,
        selected: true,
        configurable: true,
        filters: getTrashPanelTypeFilters(),
        render: (resource) => renderType(resource as CollectionResource),
    },
    {
        name: TrashPanelColumnNames.FILE_SIZE,
        selected: true,
        configurable: true,
        sort: {direction: SortDirection.NONE, field: "fileSizeTotal"},
        filters: createTree(),
        render: (resource) => renderFileSize(resource as CollectionResource),
    },
    {
        name: TrashPanelColumnNames.TRASHED_DATE,
        selected: true,
        configurable: true,
        sort: {direction: SortDirection.DESC, field: "trashAt"},
        filters: createTree(),
        render: (resource) => renderTrashDate(resource as CollectionResource),
    },
    {
        name: TrashPanelColumnNames.TO_BE_DELETED,
        selected: true,
        configurable: true,
        sort: {direction: SortDirection.NONE, field: "deleteAt"},
        filters: createTree(),
        render: (resource) => renderDeleteDate(resource as CollectionResource),
    },
    {
        name: '',
        selected: true,
        configurable: false,
        filters: createTree(),
        render: (resource) => renderRestoreFromTrash(resource as TrashableResource)
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
                return <div className={this.props.classes.root}><DataExplorer
                    id={TRASH_PANEL_ID}
                    onRowClick={this.handleRowClick}
                    onRowDoubleClick={this.handleRowDoubleClick}
                    onContextMenu={this.handleContextMenu}
                    contextMenuColumn={false}
                    defaultViewIcon={TrashIcon}
                    defaultViewMessages={['Your trash list is empty.']} />
                </div>;
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
                this.props.dispatch<any>(loadDetailsPanel(resourceUuid));
            }

            handleRowDoubleClick = (uuid: string) => {
                this.props.dispatch<any>(navigateTo(uuid));
            }

            handleRowClick = ({uuid}: CollectionResource) => {
                this.props.dispatch<any>(toggleOne(uuid))
                this.props.dispatch<any>(deselectAllOthers(uuid))
                this.props.dispatch<any>(loadDetailsPanel(uuid));
            }
        }
    )
);
