// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { openLinkAttributesDialog, openLinkRemoveDialog } from 'store/link-panel/link-panel-actions';
import { openAdvancedTabDialog } from 'store/advanced-tab/advanced-tab';
import { ContextMenuActionSet, ContextMenuActionNames } from 'views-components/context-menu/context-menu-action-set';
import { AdvancedIcon, RemoveIcon, AttributesIcon } from 'components/icon/icon';

export const linkActionSet: ContextMenuActionSet = [
    [
        {
            name: ContextMenuActionNames.ATTRIBUTES,
            icon: AttributesIcon,
            execute: (dispatch, resources) => {
                 dispatch<any>(openLinkAttributesDialog(resources[0].uuid));
            },
        },
        {
            name: ContextMenuActionNames.API_DETAILS,
            icon: AdvancedIcon,
            execute: (dispatch, resources) => {
                 dispatch<any>(openAdvancedTabDialog(resources[0].uuid));
            },
        },
        {
            name: ContextMenuActionNames.REMOVE,
            icon: RemoveIcon,
            execute: (dispatch, resources) => {
                 dispatch<any>(openLinkRemoveDialog(resources[0].uuid));
            },
        },
    ],
];
