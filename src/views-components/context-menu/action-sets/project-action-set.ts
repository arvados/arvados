// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuActionSet } from "../context-menu-action-set";
import { NewProjectIcon, RenameIcon, MoveToIcon, DetailsIcon, AdvancedIcon, OpenIcon, Link, FolderSharedIcon } from '~/components/icon/icon';
import { ToggleFavoriteAction } from "../actions/favorite-action";
import { toggleFavorite } from "~/store/favorites/favorites-actions";
import { favoritePanelActions } from "~/store/favorite-panel/favorite-panel-action";
import { openMoveProjectDialog } from '~/store/projects/project-move-actions';
import { openProjectCreateDialog } from '~/store/projects/project-create-actions';
import { openProjectUpdateDialog } from '~/store/projects/project-update-actions';
import { ToggleTrashAction } from "~/views-components/context-menu/actions/trash-action";
import { toggleProjectTrashed } from "~/store/trash/trash-actions";
import { ShareIcon } from '~/components/icon/icon';
import { openSharingDialog } from "~/store/sharing-dialog/sharing-dialog-actions";
import { openAdvancedTabDialog } from "~/store/advanced-tab/advanced-tab";
import { toggleDetailsPanel } from '~/store/details-panel/details-panel-action';
import { copyToClipboardAction, openInNewTabAction } from "~/store/open-in-new-tab/open-in-new-tab.actions";
import { openWebDavS3InfoDialog } from "~/store/collections/collection-info-actions";

export const readOnlyProjectActionSet: ContextMenuActionSet = [[
    {
        component: ToggleFavoriteAction,
        name: 'ToggleFavoriteAction',
        execute: (dispatch, resource) => {
            dispatch<any>(toggleFavorite(resource)).then(() => {
                dispatch<any>(favoritePanelActions.REQUEST_ITEMS());
            });
        }
    },
    {
        icon: OpenIcon,
        name: "Open in new tab",
        execute: (dispatch, resource) => {
            dispatch<any>(openInNewTabAction(resource));
        }
    },
    {
        icon: Link,
        name: "Copy to clipboard",
        execute: (dispatch, resource) => {
            dispatch<any>(copyToClipboardAction(resource));
        }
    },
    {
        icon: DetailsIcon,
        name: "View details",
        execute: dispatch => {
            dispatch<any>(toggleDetailsPanel());
        }
    },
    {
        icon: AdvancedIcon,
        name: "Advanced",
        execute: (dispatch, resource) => {
            dispatch<any>(openAdvancedTabDialog(resource.uuid));
        }
    },
    {
        icon: FolderSharedIcon,
        name: "Open as network folder or S3 bucket",
        execute: (dispatch, resource) => {
            dispatch<any>(openWebDavS3InfoDialog(resource.uuid));
        }
    },
]];

export const filterGroupActionSet: ContextMenuActionSet = [
    [
        ...readOnlyProjectActionSet.reduce((prev, next) => prev.concat(next), []),
        {
            icon: RenameIcon,
            name: "Edit project",
            execute: (dispatch, resource) => {
                dispatch<any>(openProjectUpdateDialog(resource));
            }
        },
        {
            icon: ShareIcon,
            name: "Share",
            execute: (dispatch, { uuid }) => {
                dispatch<any>(openSharingDialog(uuid));
            }
        },
        {
            icon: MoveToIcon,
            name: "Move to",
            execute: (dispatch, resource) => {
                dispatch<any>(openMoveProjectDialog(resource));
            }
        },
        {
            component: ToggleTrashAction,
            name: 'ToggleTrashAction',
            execute: (dispatch, resource) => {
                dispatch<any>(toggleProjectTrashed(resource.uuid, resource.ownerUuid, resource.isTrashed!!));
            }
        },
    ]
];

export const projectActionSet: ContextMenuActionSet = [
    [
        ...filterGroupActionSet.reduce((prev, next) => prev.concat(next), []),
        {
            icon: NewProjectIcon,
            name: "New project",
            execute: (dispatch, resource) => {
                dispatch<any>(openProjectCreateDialog(resource.uuid));
            }
        },
    ]
];
