// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from 'redux';
import { RootState } from 'store/store';
import { AuthState } from 'store/auth/auth-reducer';
import { getResource } from 'store/resources/resources';
import { Resource, ResourceKind } from 'models/resource';
import { resourceIsFrozen } from 'common/frozen-resources';
import { GroupResource, GroupClass, isGroupResource, isUserGroup } from 'models/group';
import { ContextMenuKind } from 'views-components/context-menu/menu-item-sort';
import { getProcess, isProcessCancelable } from 'store/processes/process';
import { isCollectionResource } from 'models/collection';
import { ResourcesState } from 'store/resources/resources';

type ProjectToMenuArgs = {
    isAdmin: boolean;
    readonly: boolean;
    isFrozen: boolean;
    canManage: boolean;
    canWrite: boolean;
    isFilterGroup: boolean;
    unfreezeRequiresAdmin: boolean;
    isEditable: boolean;
};

type CollectionToMenuArgs = {
    isAdmin: boolean;
    isEditable: boolean;
    isOnlyWriteable: boolean;
    isOldVersion: boolean;
    isTrashed: boolean;
};

type ProcessToMenuArgs = {
    isAdmin: boolean;
    isRunning: boolean;
    canWriteProcess: boolean;
};

type ProjectMenuKind = ContextMenuKind.PROJECT
    | ContextMenuKind.PROJECT_ADMIN
    | ContextMenuKind.FROZEN_PROJECT
    | ContextMenuKind.FROZEN_PROJECT_ADMIN
    | ContextMenuKind.FROZEN_MANAGEABLE_PROJECT
    | ContextMenuKind.MANAGEABLE_PROJECT
    | ContextMenuKind.READONLY_PROJECT
    | ContextMenuKind.WRITEABLE_PROJECT
    | ContextMenuKind.FILTER_GROUP
    | ContextMenuKind.FILTER_GROUP_ADMIN;

type CollectionMenuKind = ContextMenuKind.COLLECTION
    | ContextMenuKind.READONLY_COLLECTION
    | ContextMenuKind.WRITEABLE_COLLECTION
    | ContextMenuKind.OLD_VERSION_COLLECTION
    | ContextMenuKind.TRASHED_COLLECTION
    | ContextMenuKind.COLLECTION_ADMIN;

type ProcessMenuKind = ContextMenuKind.PROCESS_RESOURCE
    | ContextMenuKind.PROCESS_ADMIN
    | ContextMenuKind.RUNNING_PROCESS_RESOURCE
    | ContextMenuKind.RUNNING_PROCESS_ADMIN
    | ContextMenuKind.READONLY_PROCESS_RESOURCE;

export const resourceToMenuKind = (uuid: string, readonly = false) =>
    (dispatch: Dispatch, getState: () => RootState): ContextMenuKind | undefined => {
        const { auth, resources } = getState();
        const resource = getResource<Resource>(uuid)(resources);
        if (!resource) return;
        const isAdmin = auth.user?.isAdmin || false;
        const isFrozen = resourceIsFrozen(resource, resources);
        const isEditable = getIsEditable(isAdmin, resource, resources, readonly, isFrozen);

        if (isUserGroup(resource)) {
            return ContextMenuKind.GROUPS
        }
        if (isGroupResource(resource)) {
            const { canManage = false, canWrite = false } = resource;
            const unfreezeRequiresAdmin = getUnfreezeRequiresAdmin(auth);
            const isFilterGroup = resource.groupClass === GroupClass.FILTER;
            return getProjectMenuKind({ isAdmin, isFrozen, isEditable, canManage, canWrite, unfreezeRequiresAdmin, isFilterGroup, readonly });
        }
        if (isCollectionResource(resource)){
            const collectionParent = getResource<GroupResource>(resource.ownerUuid)(resources);
            const isOnlyWriteable = collectionParent?.canWrite === true && collectionParent.canManage === false;
            const isOldVersion = resource.uuid !== resource.currentVersionUuid;
            const isTrashed = resource.isTrashed || false;
            return getCollectionMenuKind({ isAdmin, isEditable, isOldVersion, isTrashed, isOnlyWriteable });
        }
        switch (resource.kind) {
            case ResourceKind.PROCESS:
                const process = getProcess(uuid)(resources);
                const canWriteProcess = !!(process && getResource<GroupResource>(process.containerRequest.ownerUuid)(resources)?.canWrite);
                const isRunning = process ? isProcessCancelable(process) : false;
                return getProcessMenuKind({ isAdmin, isRunning, canWriteProcess });
            case ResourceKind.USER:
                return ContextMenuKind.USER_DETAILS;
            case ResourceKind.LINK:
                return ContextMenuKind.LINK;
            case ResourceKind.WORKFLOW:
                return isEditable ? ContextMenuKind.WORKFLOW : ContextMenuKind.READONLY_WORKFLOW;
            default:
                return;
        }
    };

const getProjectMenuKind = ({ isAdmin, readonly, isFrozen, canManage, canWrite, unfreezeRequiresAdmin, isEditable, isFilterGroup }: ProjectToMenuArgs): ProjectMenuKind => {
    if (isFrozen) {
        if (isAdmin) {
            return ContextMenuKind.FROZEN_PROJECT_ADMIN;
        }
        if (canManage) {
            if (unfreezeRequiresAdmin) return ContextMenuKind.MANAGEABLE_PROJECT;
            return ContextMenuKind.FROZEN_MANAGEABLE_PROJECT;
        }
        if (isEditable) {
            return ContextMenuKind.FROZEN_PROJECT;
        }
        return ContextMenuKind.READONLY_PROJECT;
    }

    if (isAdmin && !readonly) {
        if (isFilterGroup) return ContextMenuKind.FILTER_GROUP_ADMIN;
        return ContextMenuKind.PROJECT_ADMIN;
    }

    if (canManage === false && canWrite === true) {
        return ContextMenuKind.WRITEABLE_PROJECT;
    }

    if (!isEditable) {
        return ContextMenuKind.READONLY_PROJECT;
    }

    if (isFilterGroup) return ContextMenuKind.FILTER_GROUP;

    return ContextMenuKind.PROJECT;
};

const getCollectionMenuKind = ({ isAdmin, isEditable, isOnlyWriteable, isOldVersion, isTrashed }: CollectionToMenuArgs): CollectionMenuKind => {
    if (isOldVersion) {
        return ContextMenuKind.OLD_VERSION_COLLECTION;
    }

    if (isTrashed && isEditable) {
        return ContextMenuKind.TRASHED_COLLECTION;
    }

    if (isAdmin && isEditable) {
        return ContextMenuKind.COLLECTION_ADMIN;
    }

    if (!isEditable) {
        return ContextMenuKind.READONLY_COLLECTION;
    }

    return isOnlyWriteable ? ContextMenuKind.WRITEABLE_COLLECTION : ContextMenuKind.COLLECTION;
};

const getProcessMenuKind = ({ isAdmin, isRunning, canWriteProcess }: ProcessToMenuArgs): ProcessMenuKind => {
    if (isAdmin) {
        return isRunning ? ContextMenuKind.RUNNING_PROCESS_ADMIN : ContextMenuKind.PROCESS_ADMIN;
    }

    if (isRunning) {
        return ContextMenuKind.RUNNING_PROCESS_RESOURCE;
    }

    return canWriteProcess ? ContextMenuKind.PROCESS_RESOURCE : ContextMenuKind.READONLY_PROCESS_RESOURCE;
};

//Utils--------------------------------------------------------------
const getUnfreezeRequiresAdmin = (auth: AuthState) => {
    const { remoteHostsConfig } = auth;
    if (!remoteHostsConfig) return false;
    return Object.keys(remoteHostsConfig).some((k) => remoteHostsConfig[k].clusterConfig.API.UnfreezeProjectRequiresAdmin);
};

const getIsEditable = (isAdmin: boolean, resource: Resource, resources: ResourcesState, readonly: boolean, isFrozen: boolean) => {
    const isEditable = (resources[resource.ownerUuid] as GroupResource)?.canWrite || (isGroupResource(resource) && resource.canWrite);
    return (isAdmin || isEditable) && !readonly && !isFrozen;
};
