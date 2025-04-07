// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuActionSet, ContextMenuActionNames } from '../context-menu-action-set';
import { DetailsIcon, ProvenanceGraphIcon, AdvancedIcon, RestoreFromTrashIcon } from 'components/icon/icon';
import { toggleResourceTrashed } from 'store/trash/trash-actions';
import { openAdvancedTabDialog } from 'store/advanced-tab/advanced-tab';
import { openDetailsPanel } from 'store/details-panel/details-panel-action';

export const trashedCollectionActionSet: ContextMenuActionSet = [
    [
        {
            icon: DetailsIcon,
            name: ContextMenuActionNames.VIEW_DETAILS,
            execute: (dispatch, resources) => {
                dispatch<any>(openDetailsPanel(resources[0].uuid));
            },
        },
        {
            icon: ProvenanceGraphIcon,
            name: ContextMenuActionNames.PROVENANCE_GRAPH,
            execute: (dispatch, resource) => {
                // add code
            },
        },
        {
            icon: AdvancedIcon,
            name: ContextMenuActionNames.API_DETAILS,
            execute: (dispatch, resources) => {
                dispatch<any>(openAdvancedTabDialog(resources[0].uuid));
            },
        },
        {
            icon: RestoreFromTrashIcon,
            name: ContextMenuActionNames.RESTORE,
            execute: (dispatch, resources) => {
                dispatch<any>(toggleResourceTrashed(resources.map(res => res.uuid), resources.some(res => res.isTrashed)));
            },
        },
    ],
];
