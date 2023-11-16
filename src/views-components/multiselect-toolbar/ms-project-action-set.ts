// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { MultiSelectMenuAction, MultiSelectMenuActionNames } from 'views-components/multiselect-toolbar/ms-menu-action-set';
import { openMoveProjectDialog } from 'store/projects/project-move-actions';
import { toggleProjectTrashed } from 'store/trash/trash-actions';
import { copyToClipboardAction, openInNewTabAction } from 'store/open-in-new-tab/open-in-new-tab.actions';
import { toggleFavorite } from 'store/favorites/favorites-actions';
import { favoritePanelActions } from 'store/favorite-panel/favorite-panel-action';
import {
    AddFavoriteIcon,
    AdvancedIcon,
    DetailsIcon,
    FreezeIcon,
    FolderSharedIcon,
    Link,
    MoveToIcon,
    NewProjectIcon,
    OpenIcon,
    RemoveFavoriteIcon,
    RenameIcon,
    ShareIcon,
    UnfreezeIcon,
} from 'components/icon/icon';
import { RestoreFromTrashIcon, TrashIcon } from 'components/icon/icon';
import { getResource } from 'store/resources/resources';
import { checkFavorite } from 'store/favorites/favorites-reducer';
import { toggleDetailsPanel } from 'store/details-panel/details-panel-action';
import { openAdvancedTabDialog } from 'store/advanced-tab/advanced-tab';
import { openWebDavS3InfoDialog } from 'store/collections/collection-info-actions';
import { openSharingDialog } from 'store/sharing-dialog/sharing-dialog-actions';
import { openProjectCreateDialog } from 'store/projects/project-create-actions';
import { openProjectUpdateDialog } from 'store/projects/project-update-actions';
import { freezeProject, unfreezeProject } from 'store/projects/project-lock-actions';

export const msToggleFavoriteAction = {
    name: MultiSelectMenuActionNames.ADD_TO_FAVORITES,
    icon: AddFavoriteIcon,
    hasAlts: true,
    altName: 'Remove from Favorites',
    altIcon: RemoveFavoriteIcon,
    isForMulti: false,
    useAlts: (uuid, iconProps) => {
        return checkFavorite(uuid, iconProps.favorites);
    },
    execute: (dispatch, resources) => {
        dispatch(toggleFavorite(resources[0])).then(() => {
            dispatch(favoritePanelActions.REQUEST_ITEMS());
        });
    },
};

export const msOpenInNewTabMenuAction = {
    name: MultiSelectMenuActionNames.OPEN_IN_NEW_TAB,
    icon: OpenIcon,
    hasAlts: false,
    isForMulti: false,
    execute: (dispatch, resources) => {
        dispatch(openInNewTabAction(resources[0]));
    },
};

export const msCopyToClipboardMenuAction = {
    name: MultiSelectMenuActionNames.COPY_TO_CLIPBOARD,
    icon: Link,
    hasAlts: false,
    isForMulti: false,
    execute: (dispatch, resources) => {
        dispatch(copyToClipboardAction(resources));
    },
};

export const msViewDetailsAction = {
    name: MultiSelectMenuActionNames.VIEW_DETAILS,
    icon: DetailsIcon,
    hasAlts: false,
    isForMulti: false,
    execute: (dispatch) => {
        dispatch(toggleDetailsPanel());
    },
};

export const msAdvancedAction = {
    name: MultiSelectMenuActionNames.API_DETAILS,
    icon: AdvancedIcon,
    hasAlts: false,
    isForMulti: false,
    execute: (dispatch, resources) => {
        dispatch(openAdvancedTabDialog(resources[0].uuid));
    },
};

export const msOpenWith3rdPartyClientAction = {
    name: MultiSelectMenuActionNames.OPEN_W_3RD_PARTY_CLIENT,
    icon: FolderSharedIcon,
    hasAlts: false,
    isForMulti: false,
    execute: (dispatch, resources) => {
        dispatch(openWebDavS3InfoDialog(resources[0].uuid));
    },
};

export const msEditProjectAction = {
    name: MultiSelectMenuActionNames.EDIT_PPROJECT,
    icon: RenameIcon,
    hasAlts: false,
    isForMulti: false,
    execute: (dispatch, resources) => {
        dispatch(openProjectUpdateDialog(resources[0]));
    },
};

export const msShareAction = {
    name: MultiSelectMenuActionNames.SHARE,
    icon: ShareIcon,
    hasAlts: false,
    isForMulti: false,
    execute: (dispatch, resources) => {
        dispatch(openSharingDialog(resources[0].uuid));
    },
};

export const msMoveToAction = {
    name: MultiSelectMenuActionNames.MOVE_TO,
    icon: MoveToIcon,
    hasAlts: false,
    isForMulti: true,
    execute: (dispatch, resource) => {
        dispatch(openMoveProjectDialog(resource[0]));
    },
};

export const msToggleTrashAction = {
    name: MultiSelectMenuActionNames.ADD_TO_TRASH,
    icon: TrashIcon,
    hasAlts: true,
    altName: 'Restore from Trash',
    altIcon: RestoreFromTrashIcon,
    isForMulti: true,
    useAlts: (uuid, iconProps) => {
        return uuid ? (getResource(uuid)(iconProps.resources) as any).isTrashed : false;
    },
    execute: (dispatch, resources) => {
        for (const resource of [...resources]) {
            dispatch(toggleProjectTrashed(resource.uuid, resource.ownerUuid, resource.isTrashed!!, resources.length > 1));
        }
    },
};

export const msFreezeProjectAction = {
    name: MultiSelectMenuActionNames.FREEZE_PROJECT,
    icon: FreezeIcon,
    hasAlts: true,
    altName: 'Unfreeze Project',
    altIcon: UnfreezeIcon,
    isForMulti: false,
    useAlts: (uuid, iconProps) => {
        return uuid ? !!(getResource(uuid)(iconProps.resources) as any).frozenByUuid : false;
    },
    execute: (dispatch, resources) => {
        if (resources[0].frozenByUuid) {
            dispatch(unfreezeProject(resources[0].uuid));
        } else {
            dispatch(freezeProject(resources[0].uuid));
        }
    },
};

export const msNewProjectAction: any = {
    name: MultiSelectMenuActionNames.NEW_PROJECT,
    icon: NewProjectIcon,
    hasAlts: false,
    isForMulti: false,
    execute: (dispatch, resource): void => {
        dispatch(openProjectCreateDialog(resource.uuid));
    },
};

export const msProjectActionSet: MultiSelectMenuAction[][] = [
    [
        msCopyToClipboardMenuAction,
        msToggleFavoriteAction,
        msOpenInNewTabMenuAction,
        msCopyToClipboardMenuAction,
        msViewDetailsAction,
        msAdvancedAction,
        msOpenWith3rdPartyClientAction,
        msEditProjectAction,
        msShareAction,
        msMoveToAction,
        msToggleTrashAction,
        msFreezeProjectAction,
        msNewProjectAction,
    ],
];
