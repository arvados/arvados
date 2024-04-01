// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { MultiSelectMenuAction, MultiSelectMenuActionSet, msCommonActionSet } from 'views-components/multiselect-toolbar/ms-menu-actions';
import { ContextMenuActionNames } from "views-components/context-menu/context-menu-action-set";
import { openMoveProjectDialog } from 'store/projects/project-move-actions';
import { toggleProjectTrashed } from 'store/trash/trash-actions';
import {
    FreezeIcon,
    MoveToIcon,
    NewProjectIcon,
    RenameIcon,
    UnfreezeIcon,
    ShareIcon,
} from 'components/icon/icon';
import { RestoreFromTrashIcon, TrashIcon, FolderSharedIcon, Link } from 'components/icon/icon';
import { getResource } from 'store/resources/resources';
import { openProjectCreateDialog } from 'store/projects/project-create-actions';
import { openProjectUpdateDialog } from 'store/projects/project-update-actions';
import { freezeProject, unfreezeProject } from 'store/projects/project-lock-actions';
import { openWebDavS3InfoDialog } from 'store/collections/collection-info-actions';
import { copyToClipboardAction } from 'store/open-in-new-tab/open-in-new-tab.actions';
import { openSharingDialog } from 'store/sharing-dialog/sharing-dialog-actions';

const {
    ADD_TO_FAVORITES,
    ADD_TO_PUBLIC_FAVORITES,
    OPEN_IN_NEW_TAB,
    COPY_LINK_TO_CLIPBOARD,
    VIEW_DETAILS,
    API_DETAILS,
    OPEN_WITH_3RD_PARTY_CLIENT,
    EDIT_PROJECT,
    SHARE,
    MOVE_TO,
    MOVE_TO_TRASH,
    FREEZE_PROJECT,
    NEW_PROJECT,
} = ContextMenuActionNames;

const msCopyToClipboardMenuAction: MultiSelectMenuAction  = {
    name: COPY_LINK_TO_CLIPBOARD,
    icon: Link,
    hasAlts: false,
    isForMulti: false,
    execute: (dispatch, resources) => {
        dispatch<any>(copyToClipboardAction(resources));
    },
};

const msEditProjectAction: MultiSelectMenuAction = {
    name: EDIT_PROJECT,
    icon: RenameIcon,
    hasAlts: false,
    isForMulti: false,
    execute: (dispatch, resources) => {
        dispatch<any>(openProjectUpdateDialog(resources[0]));
    },
};

const msMoveToAction: MultiSelectMenuAction = {
    name: MOVE_TO,
    icon: MoveToIcon,
    hasAlts: false,
    isForMulti: true,
    execute: (dispatch, resource) => {
        dispatch<any>(openMoveProjectDialog(resource[0]));
    },
};

const msOpenWith3rdPartyClientAction: MultiSelectMenuAction  = {
    name: OPEN_WITH_3RD_PARTY_CLIENT,
    icon: FolderSharedIcon,
    hasAlts: false,
    isForMulti: false,
    execute: (dispatch, resources) => {
        dispatch<any>(openWebDavS3InfoDialog(resources[0].uuid));
    },
};

export const msToggleTrashAction: MultiSelectMenuAction = {
    name: MOVE_TO_TRASH,
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
            dispatch<any>(toggleProjectTrashed(resource.uuid, resource.ownerUuid, resource.isTrashed!!, resources.length > 1));
        }
    },
};

const msFreezeProjectAction: MultiSelectMenuAction = {
    name: FREEZE_PROJECT,
    icon: FreezeIcon,
    hasAlts: true,
    altName: 'Unfreeze Project',
    altIcon: UnfreezeIcon,
    isForMulti: false,
    useAlts: (uuid, iconProps) => {
        return uuid ? !!(getResource(uuid)(iconProps.resources) as any).frozenByUuid : false;
    },
    execute: (dispatch, resources) => {
        if ((resources[0] as any).frozenByUuid) {
            dispatch<any>(unfreezeProject(resources[0].uuid));
        } else {
            dispatch<any>(freezeProject(resources[0].uuid));
        }
    },
};

const msNewProjectAction: MultiSelectMenuAction = {
    name: NEW_PROJECT,
    icon: NewProjectIcon,
    hasAlts: false,
    isForMulti: false,
    execute: (dispatch, resources): void => {
        dispatch<any>(openProjectCreateDialog(resources[0].uuid));
    },
};

const msShareAction: MultiSelectMenuAction  = {
    name: SHARE,
    icon: ShareIcon,
    hasAlts: false,
    isForMulti: false,
    execute: (dispatch, resources) => {
        dispatch<any>(openSharingDialog(resources[0].uuid));
    },
};

export const msProjectActionSet: MultiSelectMenuActionSet = [
    [
        ...msCommonActionSet,
        msEditProjectAction,
        msMoveToAction,
        msToggleTrashAction,
        msNewProjectAction,
        msFreezeProjectAction,
        msOpenWith3rdPartyClientAction,
        msCopyToClipboardMenuAction,
        msShareAction,
    ],
];

export const msCommonProjectActionFilter = new Set<string>([
    ADD_TO_FAVORITES,
    MOVE_TO_TRASH,
    API_DETAILS,
    COPY_LINK_TO_CLIPBOARD,
    EDIT_PROJECT,
    FREEZE_PROJECT,
    MOVE_TO,
    NEW_PROJECT,
    OPEN_IN_NEW_TAB,
    OPEN_WITH_3RD_PARTY_CLIENT,
    SHARE,
    VIEW_DETAILS,
]);
export const msReadOnlyProjectActionFilter = new Set<string>([ADD_TO_FAVORITES, API_DETAILS, COPY_LINK_TO_CLIPBOARD, OPEN_IN_NEW_TAB, OPEN_WITH_3RD_PARTY_CLIENT, VIEW_DETAILS,]);
export const msFrozenProjectActionFilter = new Set<string>([ADD_TO_FAVORITES, API_DETAILS, COPY_LINK_TO_CLIPBOARD, OPEN_IN_NEW_TAB, OPEN_WITH_3RD_PARTY_CLIENT, VIEW_DETAILS, SHARE, FREEZE_PROJECT])
export const msAdminFrozenProjectActionFilter = new Set<string>([ADD_TO_FAVORITES, API_DETAILS, COPY_LINK_TO_CLIPBOARD, OPEN_IN_NEW_TAB, OPEN_WITH_3RD_PARTY_CLIENT, VIEW_DETAILS, SHARE, FREEZE_PROJECT, ADD_TO_PUBLIC_FAVORITES])

export const msFilterGroupActionFilter = new Set<string>([ADD_TO_FAVORITES, API_DETAILS, COPY_LINK_TO_CLIPBOARD, OPEN_IN_NEW_TAB, OPEN_WITH_3RD_PARTY_CLIENT, VIEW_DETAILS, SHARE, MOVE_TO_TRASH, EDIT_PROJECT, MOVE_TO])
export const msAdminFilterGroupActionFilter = new Set<string>([ADD_TO_FAVORITES, API_DETAILS, COPY_LINK_TO_CLIPBOARD, OPEN_IN_NEW_TAB, OPEN_WITH_3RD_PARTY_CLIENT, VIEW_DETAILS, SHARE, MOVE_TO_TRASH, EDIT_PROJECT, MOVE_TO, ADD_TO_PUBLIC_FAVORITES])