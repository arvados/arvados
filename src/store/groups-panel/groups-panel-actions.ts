// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from 'redux';
import { reset, startSubmit, stopSubmit, FormErrors, initialize } from 'redux-form';
import { bindDataExplorerActions } from "store/data-explorer/data-explorer-action";
import { dialogActions } from 'store/dialog/dialog-actions';
import { RootState } from 'store/store';
import { ServiceRepository } from 'services/services';
import { getResource } from 'store/resources/resources';
import { GroupResource, GroupClass } from 'models/group';
import { getCommonResourceServiceError, CommonResourceServiceError } from 'services/common-service/common-resource-service';
import { snackbarActions, SnackbarKind } from 'store/snackbar/snackbar-actions';
import { PermissionLevel } from 'models/permission';
import { PermissionService } from 'services/permission-service/permission-service';
import { FilterBuilder } from 'services/api/filter-builder';
import { ProjectUpdateFormDialogData, PROJECT_UPDATE_FORM_NAME } from 'store/projects/project-update-actions';

export const GROUPS_PANEL_ID = "groupsPanel";

export const GROUP_ATTRIBUTES_DIALOG = 'groupAttributesDialog';
export const GROUP_REMOVE_DIALOG = 'groupRemoveDialog';

export const GroupsPanelActions = bindDataExplorerActions(GROUPS_PANEL_ID);

export const loadGroupsPanel = () => GroupsPanelActions.REQUEST_ITEMS();

export const openCreateGroupDialog = () =>
    (dispatch: Dispatch, getState: () => RootState) => {
        dispatch(initialize(PROJECT_UPDATE_FORM_NAME, {}));
        dispatch(dialogActions.OPEN_DIALOG({ id: PROJECT_UPDATE_FORM_NAME, data: {sourcePanel: GroupClass.ROLE, create: true} }));
    };

export const openGroupAttributes = (uuid: string) =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const { resources } = getState();
        const data = getResource<GroupResource>(uuid)(resources);
        dispatch(dialogActions.OPEN_DIALOG({ id: GROUP_ATTRIBUTES_DIALOG, data }));
    };

export const removeGroup = (uuid: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Removing ...', kind: SnackbarKind.INFO }));
        await services.groupsService.delete(uuid);
        dispatch(snackbarActions.OPEN_SNACKBAR({ message: 'Removed.', hideDuration: 2000, kind: SnackbarKind.SUCCESS }));
        dispatch<any>(loadGroupsPanel());
    };

export const openRemoveGroupDialog = (uuid: string) =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(dialogActions.OPEN_DIALOG({
            id: GROUP_REMOVE_DIALOG,
            data: {
                title: 'Remove group',
                text: 'Are you sure you want to remove this group?',
                confirmButtonLabel: 'Remove',
                uuid
            }
        }));
    };

// Group edit dialog uses project update dialog with sourcePanel set to reload the appropriate parts
export const openGroupUpdateDialog = (resource: ProjectUpdateFormDialogData) =>
    (dispatch: Dispatch, getState: () => RootState) => {
        dispatch(initialize(PROJECT_UPDATE_FORM_NAME, resource));
        dispatch(dialogActions.OPEN_DIALOG({ id: PROJECT_UPDATE_FORM_NAME, data: {sourcePanel: GroupClass.ROLE} }));
    };

export const updateGroup = (project: ProjectUpdateFormDialogData) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const uuid = project.uuid || '';
        dispatch(startSubmit(PROJECT_UPDATE_FORM_NAME));
        try {
            const updatedGroup = await services.groupsService.update(uuid, { name: project.name, description: project.description });
            dispatch(GroupsPanelActions.REQUEST_ITEMS());
            dispatch(reset(PROJECT_UPDATE_FORM_NAME));
            dispatch(dialogActions.CLOSE_DIALOG({ id: PROJECT_UPDATE_FORM_NAME }));
            return updatedGroup;
        } catch (e) {
            dispatch(stopSubmit(PROJECT_UPDATE_FORM_NAME));
            const error = getCommonResourceServiceError(e);
            if (error === CommonResourceServiceError.UNIQUE_NAME_VIOLATION) {
                dispatch(stopSubmit(PROJECT_UPDATE_FORM_NAME, { name: 'Group with the same name already exists.' } as FormErrors));
            }
            return ;
        }
    };

export const createGroup = ({ name, users = [], description }: ProjectUpdateFormDialogData) =>
    async (dispatch: Dispatch, _: {}, { groupsService, permissionService }: ServiceRepository) => {
        dispatch(startSubmit(PROJECT_UPDATE_FORM_NAME));
        try {
            const newGroup = await groupsService.create({ name, description, groupClass: GroupClass.ROLE });
            for (const user of users) {
                await addGroupMember({
                    user,
                    group: newGroup,
                    dispatch,
                    permissionService,
                });
            }
            dispatch(dialogActions.CLOSE_DIALOG({ id: PROJECT_UPDATE_FORM_NAME }));
            dispatch(reset(PROJECT_UPDATE_FORM_NAME));
            dispatch(loadGroupsPanel());
            dispatch(snackbarActions.OPEN_SNACKBAR({
                message: `${newGroup.name} group has been created`,
                kind: SnackbarKind.SUCCESS
            }));
            return newGroup;
        } catch (e) {
            const error = getCommonResourceServiceError(e);
            if (error === CommonResourceServiceError.UNIQUE_NAME_VIOLATION) {
                dispatch(stopSubmit(PROJECT_UPDATE_FORM_NAME, { name: 'Group with the same name already exists.' } as FormErrors));
            }
            return;
        }
    };

interface AddGroupMemberArgs {
    user: { uuid: string, name: string };
    group: { uuid: string, name: string };
    dispatch: Dispatch;
    permissionService: PermissionService;
}

/**
 * Group membership is determined by whether the group has can_read permission on an object.
 * If a group G can_read an object A, then we say A is a member of G.
 *
 * [Permission model docs](https://doc.arvados.org/api/permission-model.html)
 */
export const addGroupMember = async ({ user, group, ...args }: AddGroupMemberArgs) => {
    await createPermission({
        head: { ...group },
        tail: { ...user },
        permissionLevel: PermissionLevel.CAN_READ,
        ...args,
    });
};

interface CreatePermissionLinkArgs {
    head: { uuid: string, name: string };
    tail: { uuid: string, name: string };
    permissionLevel: PermissionLevel;
    dispatch: Dispatch;
    permissionService: PermissionService;
}

const createPermission = async ({ head, tail, permissionLevel, dispatch, permissionService }: CreatePermissionLinkArgs) => {
    try {
        await permissionService.create({
            tailUuid: tail.uuid,
            headUuid: head.uuid,
            name: permissionLevel,
        });
    } catch (e) {
        dispatch(snackbarActions.OPEN_SNACKBAR({
            message: `Could not add ${tail.name} -> ${head.name} relation`,
            kind: SnackbarKind.ERROR,
        }));
    }
};

interface DeleteGroupMemberArgs {
    link: { uuid: string };
    dispatch: Dispatch;
    permissionService: PermissionService;
}

export const deleteGroupMember = async ({ link, ...args }: DeleteGroupMemberArgs) => {
    await deletePermission({
        uuid: link.uuid,
        ...args,
    });
};

interface DeletePermissionLinkArgs {
    uuid: string;
    dispatch: Dispatch;
    permissionService: PermissionService;
}

export const deletePermission = async ({ uuid, dispatch, permissionService }: DeletePermissionLinkArgs) => {
    try {
        const permissionsResponse = await permissionService.list({
            filters: new FilterBuilder()
                .addEqual('uuid', uuid)
                .getFilters()
        });
        const [permission] = permissionsResponse.items;
        if (permission) {
            await permissionService.delete(permission.uuid);
        } else {
            throw new Error('Permission not found');
        }
    } catch (e) {
        dispatch(snackbarActions.OPEN_SNACKBAR({
            message: `Could not delete ${uuid} permission`,
            kind: SnackbarKind.ERROR,
        }));
    }
};
