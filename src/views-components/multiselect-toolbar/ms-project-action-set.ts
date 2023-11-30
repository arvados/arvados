// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { MultiSelectMenuActionSet, MultiSelectMenuActionNames, msCommonActionSet } from 'views-components/multiselect-toolbar/ms-menu-actions';
import { openMoveProjectDialog } from 'store/projects/project-move-actions';
import { toggleProjectTrashed } from 'store/trash/trash-actions';
import {
    FreezeIcon,
    MoveToIcon,
    NewProjectIcon,
    RenameIcon,
    UnfreezeIcon,
} from 'components/icon/icon';
import { RestoreFromTrashIcon, TrashIcon } from 'components/icon/icon';
import { getResource } from 'store/resources/resources';
import { openProjectCreateDialog } from 'store/projects/project-create-actions';
import { openProjectUpdateDialog } from 'store/projects/project-update-actions';
import { freezeProject, unfreezeProject } from 'store/projects/project-lock-actions';

const {
    ADD_TO_FAVORITES,
    OPEN_IN_NEW_TAB,
    COPY_TO_CLIPBOARD,
    VIEW_DETAILS,
    API_DETAILS,
    OPEN_W_3RD_PARTY_CLIENT,
    EDIT_PPROJECT,
    SHARE,
    MOVE_TO,
    ADD_TO_TRASH,
    FREEZE_PROJECT,
    NEW_PROJECT,
} = MultiSelectMenuActionNames;

const msEditProjectAction = {
    name: EDIT_PPROJECT,
    icon: RenameIcon,
    hasAlts: false,
    isForMulti: false,
    execute: (dispatch, resources) => {
        dispatch(openProjectUpdateDialog(resources[0]));
    },
};

const msMoveToAction = {
    name: MOVE_TO,
    icon: MoveToIcon,
    hasAlts: false,
    isForMulti: true,
    execute: (dispatch, resource) => {
        dispatch(openMoveProjectDialog(resource[0]));
    },
};

export const msToggleTrashAction = {
    name: ADD_TO_TRASH,
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

const msFreezeProjectAction = {
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
        if (resources[0].frozenByUuid) {
            dispatch(unfreezeProject(resources[0].uuid));
        } else {
            dispatch(freezeProject(resources[0].uuid));
        }
    },
};

const msNewProjectAction: any = {
    name: NEW_PROJECT,
    icon: NewProjectIcon,
    hasAlts: false,
    isForMulti: false,
    execute: (dispatch, resource): void => {
        dispatch(openProjectCreateDialog(resource.uuid));
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
    ],
];

export const msProjectActionFilter = new Set<string>([
    ADD_TO_FAVORITES,
    ADD_TO_TRASH,
    API_DETAILS,
    COPY_TO_CLIPBOARD,
    EDIT_PPROJECT,
    FREEZE_PROJECT,
    MOVE_TO,
    NEW_PROJECT,
    OPEN_IN_NEW_TAB,
    OPEN_W_3RD_PARTY_CLIENT,
    SHARE,
    VIEW_DETAILS,
]);
export const msReadOnlyProjectActionFilter = new Set<string>([ADD_TO_FAVORITES, OPEN_IN_NEW_TAB, COPY_TO_CLIPBOARD, VIEW_DETAILS, API_DETAILS, OPEN_W_3RD_PARTY_CLIENT]);
export const msFrozenProjectActionFilter = new Set<string>([ADD_TO_FAVORITES, OPEN_IN_NEW_TAB, COPY_TO_CLIPBOARD, VIEW_DETAILS, API_DETAILS, OPEN_W_3RD_PARTY_CLIENT, SHARE, FREEZE_PROJECT])
export const msFilterGroupActionFilter = new Set<string>([ADD_TO_FAVORITES, OPEN_IN_NEW_TAB, COPY_TO_CLIPBOARD, VIEW_DETAILS, API_DETAILS, OPEN_W_3RD_PARTY_CLIENT, EDIT_PPROJECT, SHARE, MOVE_TO, ADD_TO_TRASH])
