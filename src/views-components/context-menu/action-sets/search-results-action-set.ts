// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuActionItemSet } from '../context-menu-action-set';
import { DetailsIcon, AdvancedIcon, OpenIcon, Link } from 'components/icon/icon';
import { openAdvancedTabDialog } from 'store/advanced-tab/advanced-tab';
import { toggleDetailsPanel } from 'store/details-panel/details-panel-action';
import { copyToClipboardAction, openInNewTabAction } from 'store/open-in-new-tab/open-in-new-tab.actions';

export const searchResultsActionSet: ContextMenuActionItemSet = [
    [
        {
            icon: OpenIcon,
            name: 'Open in new tab',
            execute: (dispatch, resources) => {
                resources.forEach((resource) => dispatch<any>(openInNewTabAction(resource)));
            },
        },
        {
            icon: Link,
            name: 'Copy to clipboard',
            execute: (dispatch, resources) => {
                dispatch<any>(copyToClipboardAction(resources));
            },
        },
        {
            icon: DetailsIcon,
            name: 'View details',
            execute: (dispatch) => {
                dispatch<any>(toggleDetailsPanel());
            },
        },
        {
            icon: AdvancedIcon,
            name: 'API Details',
            execute: (dispatch, resources) => {
                dispatch<any>(openAdvancedTabDialog(resources[0].uuid));
            },
        },
    ],
];
