// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { connect } from "react-redux";
import { Dispatch } from "redux";
import { CollectionResource } from 'models/collection';
import { RootState } from "store/store";
import { toggleResourceTrashed } from "store/trash/trash-actions";
import { getTrashPanelTypeFilters } from 'store/resource-type-filters/resource-type-filters';
import {
    getResource,
    ResourcesState
} from "store/resources/resources";
import { IconButton, Tooltip } from "@mui/material";
import { DataColumns, SortDirection } from 'components/data-table/data-column';
import {
    ResourceDeleteDate,
    ResourceFileSize,
    ResourceName,
    ResourceTrashDate,
    ResourceType
} from "views-components/data-explorer/renderers";
import { createTree } from 'models/tree';
import { RestoreFromTrashIcon } from "components/icon/icon";
import { TrashableResource } from "models/resource";

export enum TrashPanelColumnNames {
    NAME = "Name",
    TYPE = "Type",
    FILE_SIZE = "File size",
    TRASHED_DATE = "Trashed date",
    TO_BE_DELETED = "To be deleted"
}

export const trashPanelColumns: DataColumns<string, CollectionResource> = [
    {
        name: TrashPanelColumnNames.NAME,
        selected: true,
        configurable: true,
        sort: { direction: SortDirection.NONE, field: "name" },
        filters: createTree(),
        render: uuid => <ResourceName uuid={uuid} />
    },
    {
        name: TrashPanelColumnNames.TYPE,
        selected: true,
        configurable: true,
        filters: getTrashPanelTypeFilters(),
        render: uuid => <ResourceType uuid={uuid} />,
    },
    {
        name: TrashPanelColumnNames.FILE_SIZE,
        selected: true,
        configurable: true,
        sort: { direction: SortDirection.NONE, field: "fileSizeTotal" },
        filters: createTree(),
        render: uuid => <ResourceFileSize uuid={uuid} />
    },
    {
        name: TrashPanelColumnNames.TRASHED_DATE,
        selected: true,
        configurable: true,
        sort: { direction: SortDirection.DESC, field: "trashAt" },
        filters: createTree(),
        render: uuid => <ResourceTrashDate uuid={uuid} />
    },
    {
        name: TrashPanelColumnNames.TO_BE_DELETED,
        selected: true,
        configurable: true,
        sort: { direction: SortDirection.NONE, field: "deleteAt" },
        filters: createTree(),
        render: uuid => <ResourceDeleteDate uuid={uuid} />
    },
    {
        name: '',
        selected: true,
        configurable: false,
        filters: createTree(),
        render: uuid => <ResourceRestore uuid={uuid} />
    }
];

export const ResourceRestore = connect((state: RootState, props: { uuid: string; dispatch?: Dispatch<any>; }) => {
    return { uuid: props.uuid, resources: state.resources, dispatch: props.dispatch };
})((props: { uuid: string; resources: ResourcesState; dispatch?: Dispatch<any>; }) => {
    const resource = getResource<TrashableResource>(props.uuid)(props.resources);
    return <Tooltip title="Restore">
        <IconButton
            style={{ padding: '0' }}
            onClick={() => {
                if (resource && props.dispatch) {
                    props.dispatch(toggleResourceTrashed(
                        [resource.uuid],
                        resource.isTrashed
                    ));
                }
            } }
            size="large">
            <RestoreFromTrashIcon />
        </IconButton>
    </Tooltip>;
});
