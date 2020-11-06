// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuActionSet } from "../context-menu-action-set";
import { NewProjectIcon, RenameIcon, MoveToIcon, DetailsIcon, AdvancedIcon, OpenIcon, Link } from '~/components/icon/icon';
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
import { TogglePublicFavoriteAction } from "~/views-components/context-menu/actions/public-favorite-action";
import { togglePublicFavorite } from "~/store/public-favorites/public-favorites-actions";
import { publicFavoritePanelActions } from "~/store/public-favorites-panel/public-favorites-action";
import { copyToClipboardAction, openInNewTabAction } from "~/store/open-in-new-tab/open-in-new-tab.actions";

export const projectAdminActionSet: ContextMenuActionSet = [[
    {
        icon: NewProjectIcon,
        name: "New project",
        execute: (dispatch, resource) => {
            dispatch<any>(openProjectCreateDialog(resource.uuid));
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
        component: ToggleFavoriteAction,
        name: 'ToggleFavoriteAction',
        execute: (dispatch, resource) => {
            dispatch<any>(toggleFavorite(resource)).then(() => {
                dispatch<any>(favoritePanelActions.REQUEST_ITEMS());
            });
        }
    },
    {
        component: TogglePublicFavoriteAction,
        name: 'TogglePublicFavoriteAction',
        execute: (dispatch, resource) => {
            dispatch<any>(togglePublicFavorite(resource)).then(() => {
                dispatch<any>(publicFavoritePanelActions.REQUEST_ITEMS());
            });
        }
    },
    {
        icon: MoveToIcon,
        name: "Move to",
        execute: (dispatch, resource) => {
            dispatch<any>(openMoveProjectDialog(resource));
        }
    },
    // {
    //     icon: CopyIcon,
    //     name: "Copy to project",
    //     execute: (dispatch, resource) => {
    //         // add code
    //     }
    // },
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
        component: ToggleTrashAction,
        name: 'ToggleTrashAction',
        execute: (dispatch, resource) => {
            dispatch<any>(toggleProjectTrashed(resource.uuid, resource.ownerUuid, resource.isTrashed!!));
        }
    },
]];
