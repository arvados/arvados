// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuActionSet, ContextMenuActionNames } from '../context-menu-action-set';
import { DetailsIcon, AdvancedIcon, OpenIcon, Link } from 'components/icon/icon';
import { openAdvancedTabDialog } from 'store/advanced-tab/advanced-tab';
import { toggleDetailsPanel } from 'store/details-panel/details-panel-action';
import { copyToClipboardAction, openInNewTabAction } from 'store/open-in-new-tab/open-in-new-tab.actions';

export const searchResultsActionSet: ContextMenuActionSet = [
    [
        {
            icon: OpenIcon,
            name: ContextMenuActionNames.OPEN_IN_NEW_TAB,
            execute: (dispatch, resources) => {
                resources.forEach((resource) => dispatch<any>(openInNewTabAction(resource)));
            },
        },
        {
            icon: Link,
            name: ContextMenuActionNames.COPY_LINK_TO_CLIPBOARD,
            execute: (dispatch, resources) => {
                dispatch<any>(copyToClipboardAction(resources));
            },
        },
        {
            icon: DetailsIcon,
            name: ContextMenuActionNames.VIEW_DETAILS,
            execute: (dispatch, resources) => {
                dispatch<any>(toggleDetailsPanel(resources[0].uuid));
            },
        },
        {
            icon: AdvancedIcon,
            name: ContextMenuActionNames.API_DETAILS,
            execute: (dispatch, resources) => {
                dispatch<any>(openAdvancedTabDialog(resources[0].uuid));
            },
        },
    ],
];
