// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from 'redux';
import { reset, startSubmit, stopSubmit, FormErrors } from 'redux-form';
import { bindDataExplorerActions } from "~/store/data-explorer/data-explorer-action";
import { dialogActions } from '~/store/dialog/dialog-actions';
import { Person } from '~/views-components/sharing-dialog/people-select';
import { RootState } from '~/store/store';
import { ServiceRepository } from '~/services/services';
import { getResource } from '~/store/resources/resources';
import { GroupResource } from '~/models/group';
import { getCommonResourceServiceError, CommonResourceServiceError } from '~/services/common-service/common-resource-service';
import { snackbarActions, SnackbarKind } from '~/store/snackbar/snackbar-actions';
import { PermissionLevel, PermissionResource } from '~/models/permission';
import { PermissionService } from '~/services/permission-service/permission-service';
import { FilterBuilder } from '~/services/api/filter-builder';

export const GROUPS_PANEL_ID = "groupsPanel";
export const CREATE_GROUP_DIALOG = "createGroupDialog";
export const CREATE_GROUP_FORM = "createGroupForm";
export const CREATE_GROUP_NAME_FIELD_NAME = 'name';
export const CREATE_GROUP_USERS_FIELD_NAME = 'users';
export const GROUP_ATTRIBUTES_DIALOG = 'groupAttributesDialog';
export const GROUP_REMOVE_DIALOG = 'groupRemoveDialog';

export const GroupsPanelActions = bindDataExplorerActions(GROUPS_PANEL_ID);

export const loadGroupsPanel = () => GroupsPanelActions.REQUEST_ITEMS();

export const openCreateGroupDialog = () =>
    (dispatch: Dispatch) => {
        dispatch(dialogActions.OPEN_DIALOG({ id: CREATE_GROUP_DIALOG, data: {} }));
        dispatch(reset(CREATE_GROUP_FORM));
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

export interface CreateGroupFormData {
    [CREATE_GROUP_NAME_FIELD_NAME]: string;
    [CREATE_GROUP_USERS_FIELD_NAME]?: Person[];
}

export const createGroup = ({ name, users = [] }: CreateGroupFormData) =>
    async (dispatch: Dispatch, _: {}, { groupsService, permissionService }: ServiceRepository) => {

        dispatch(startSubmit(CREATE_GROUP_FORM));

        try {

            const newGroup = await groupsService.create({ name });

            for (const user of users) {

                await addGroupMember({
                    user,
                    group: newGroup,
                    dispatch,
                    permissionService,
                });

            }

            dispatch(dialogActions.CLOSE_DIALOG({ id: CREATE_GROUP_DIALOG }));
            dispatch(reset(CREATE_GROUP_FORM));
            dispatch(loadGroupsPanel());
            dispatch(snackbarActions.OPEN_SNACKBAR({
                message: `${newGroup.name} group has been created`,
                kind: SnackbarKind.SUCCESS
            }));

            return newGroup;

        } catch (e) {

            const error = getCommonResourceServiceError(e);
            if (error === CommonResourceServiceError.UNIQUE_VIOLATION) {
                dispatch(stopSubmit(CREATE_GROUP_FORM, { name: 'Group with the same name already exists.' } as FormErrors));
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
        head: { ...user },
        tail: { ...group },
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
    user: { uuid: string, name: string };
    group: { uuid: string, name: string };
    dispatch: Dispatch;
    permissionService: PermissionService;
}

export const deleteGroupMember = async ({ user, group, ...args }: DeleteGroupMemberArgs) => {

    await deletePermission({
        tail: group,
        head: user,
        ...args,
    });

};

interface DeletePermissionLinkArgs {
    head: { uuid: string, name: string };
    tail: { uuid: string, name: string };
    dispatch: Dispatch;
    permissionService: PermissionService;
}

export const deletePermission = async ({ head, tail, dispatch, permissionService }: DeletePermissionLinkArgs) => {

    try {

        const permissionsResponse = await permissionService.list({

            filters: new FilterBuilder()
                .addEqual('tailUuid', tail.uuid)
                .addEqual('headUuid', head.uuid)
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
            message: `Could not delete ${tail.name} -> ${head.name} relation`,
            kind: SnackbarKind.ERROR,
        }));

    }

};