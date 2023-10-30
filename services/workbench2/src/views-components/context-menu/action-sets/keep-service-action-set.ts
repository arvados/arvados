// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { openKeepServiceAttributesDialog, openKeepServiceRemoveDialog } from 'store/keep-services/keep-services-actions';
import { openAdvancedTabDialog } from 'store/advanced-tab/advanced-tab';
import { ContextMenuActionSet } from 'views-components/context-menu/context-menu-action-set';
import { AdvancedIcon, RemoveIcon, AttributesIcon } from 'components/icon/icon';

export const keepServiceActionSet: ContextMenuActionSet = [
    [
        {
            name: 'Attributes',
            icon: AttributesIcon,
            execute: (dispatch, resources) => {
                 dispatch<any>(openKeepServiceAttributesDialog(resources[0].uuid));
            },
        },
        {
            name: 'API Details',
            icon: AdvancedIcon,
            execute: (dispatch, resources) => {
                 dispatch<any>(openAdvancedTabDialog(resources[0].uuid));
            },
        },
        {
            name: 'Remove',
            icon: RemoveIcon,
            execute: (dispatch, resources) => {
                 dispatch<any>(openKeepServiceRemoveDialog(resources[0].uuid));
            },
        },
    ],
];
