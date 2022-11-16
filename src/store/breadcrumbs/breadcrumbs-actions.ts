// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from 'redux';
import { RootState } from 'store/store';
import { getUserUuid } from "common/getuser";
import { getResource } from 'store/resources/resources';
import { TreePicker } from '../tree-picker/tree-picker';
import { getSidePanelTreeBranch, getSidePanelTreeNodeAncestorsIds } from '../side-panel-tree/side-panel-tree-actions';
import { propertiesActions } from '../properties/properties-actions';
import { getProcess } from 'store/processes/process';
import { ServiceRepository } from 'services/services';
import { SidePanelTreeCategory, activateSidePanelTreeItem } from 'store/side-panel-tree/side-panel-tree-actions';
import { updateResources } from '../resources/resources-actions';
import { ResourceKind } from 'models/resource';
import { GroupResource } from 'models/group';
import { extractUuidKind } from 'models/resource';
import { UserResource } from 'models/user';
import { FilterBuilder } from 'services/api/filter-builder';
import { ProcessResource } from 'models/process';
import { OrderBuilder } from 'services/api/order-builder';
import { Breadcrumb } from 'components/breadcrumbs/breadcrumbs';
import { ContainerRequestResource, containerRequestFieldsNoMounts } from 'models/container-request';
import { CollectionIcon, IconType, ProcessBreadcrumbIcon, ProjectIcon } from 'components/icon/icon';
import { CollectionResource } from 'models/collection';

export const BREADCRUMBS = 'breadcrumbs';

export const setBreadcrumbs = (breadcrumbs: any, currentItem?: CollectionResource | ContainerRequestResource | GroupResource) => {
    if (currentItem) {
        breadcrumbs.push(resourceToBreadcrumb(currentItem));
    }
    return propertiesActions.SET_PROPERTY({ key: BREADCRUMBS, value: breadcrumbs });
};

const resourceToBreadcrumbIcon = (resource: CollectionResource | ContainerRequestResource | GroupResource): IconType | undefined => {
    switch (resource.kind) {
        case ResourceKind.PROJECT:
            return ProjectIcon;
        case ResourceKind.PROCESS:
            return ProcessBreadcrumbIcon;
        case ResourceKind.COLLECTION:
            return CollectionIcon;
        default:
            return undefined;
    }
}

const resourceToBreadcrumb = (resource: CollectionResource | ContainerRequestResource | GroupResource): Breadcrumb => ({
    label: resource.name,
    uuid: resource.uuid,
    icon: resourceToBreadcrumbIcon(resource),
})

const getSidePanelTreeBreadcrumbs = (uuid: string) => (treePicker: TreePicker): Breadcrumb[] => {
    const nodes = getSidePanelTreeBranch(uuid)(treePicker);
    return nodes.map(node =>
        typeof node.value === 'string'
            ? { label: node.value, uuid: node.id }
            : resourceToBreadcrumb(node.value));
};

export const setSidePanelBreadcrumbs = (uuid: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const { treePicker, collectionPanel: { item } } = getState();
        const breadcrumbs = getSidePanelTreeBreadcrumbs(uuid)(treePicker);
        const path = getState().router.location!.pathname;
        const currentUuid = path.split('/')[2];
        const uuidKind = extractUuidKind(currentUuid);

        if (uuidKind === ResourceKind.COLLECTION) {
            const collectionItem = item ? item : await services.collectionService.get(currentUuid);
            const parentProcessItem = await getCollectionParent(collectionItem)(services);
            if (parentProcessItem) {
                const mainProcessItem = await getProcessParent(parentProcessItem)(services);
                mainProcessItem && breadcrumbs.push(resourceToBreadcrumb(mainProcessItem));
                breadcrumbs.push(resourceToBreadcrumb(parentProcessItem));
            }
            dispatch(setBreadcrumbs(breadcrumbs, collectionItem));
        } else if (uuidKind === ResourceKind.PROCESS) {
            const processItem = await services.containerRequestService.get(currentUuid);
            const parentProcessItem = await getProcessParent(processItem)(services);
            if (parentProcessItem) {
                breadcrumbs.push(resourceToBreadcrumb(parentProcessItem));
            }
            dispatch(setBreadcrumbs(breadcrumbs, processItem));
        }
        dispatch(setBreadcrumbs(breadcrumbs));
    };

export const setSharedWithMeBreadcrumbs = (uuid: string) =>
    setCategoryBreadcrumbs(uuid, SidePanelTreeCategory.SHARED_WITH_ME);

export const setTrashBreadcrumbs = (uuid: string) =>
    setCategoryBreadcrumbs(uuid, SidePanelTreeCategory.TRASH);

export const setCategoryBreadcrumbs = (uuid: string, category: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const ancestors = await services.ancestorsService.ancestors(uuid, '');
        dispatch(updateResources(ancestors));
        const initialBreadcrumbs: Breadcrumb[] = [
            { label: category, uuid: category }
        ];
        const { collectionPanel: { item } } = getState();
        const path = getState().router.location!.pathname;
        const currentUuid = path.split('/')[2];
        const uuidKind = extractUuidKind(currentUuid);
        let breadcrumbs = ancestors.reduce((breadcrumbs, ancestor) =>
            ancestor.kind === ResourceKind.GROUP
                ? [...breadcrumbs, resourceToBreadcrumb(ancestor)]
                : breadcrumbs,
            initialBreadcrumbs);
        if (uuidKind === ResourceKind.COLLECTION) {
            const collectionItem = item ? item : await services.collectionService.get(currentUuid);
            const parentProcessItem = await getCollectionParent(collectionItem)(services);
            if (parentProcessItem) {
                const mainProcessItem = await getProcessParent(parentProcessItem)(services);
                mainProcessItem && breadcrumbs.push(resourceToBreadcrumb(mainProcessItem));
                breadcrumbs.push(resourceToBreadcrumb(parentProcessItem));
            }
            dispatch(setBreadcrumbs(breadcrumbs, collectionItem));
        } else if (uuidKind === ResourceKind.PROCESS) {
            const processItem = await services.containerRequestService.get(currentUuid);
            const parentProcessItem = await getProcessParent(processItem)(services);
            if (parentProcessItem) {
                breadcrumbs.push(resourceToBreadcrumb(parentProcessItem));
            }
            dispatch(setBreadcrumbs(breadcrumbs, processItem));
        }
        dispatch(setBreadcrumbs(breadcrumbs));
    };

const getProcessParent = (childProcess: ContainerRequestResource) =>
    async (services: ServiceRepository): Promise<ContainerRequestResource | undefined> => {
        if (childProcess.requestingContainerUuid) {
            const parentProcesses = await services.containerRequestService.list({
                order: new OrderBuilder<ProcessResource>().addAsc('createdAt').getOrder(),
                filters: new FilterBuilder().addEqual('container_uuid', childProcess.requestingContainerUuid).getFilters(),
                select: containerRequestFieldsNoMounts,
            });
            if (parentProcesses.items.length > 0) {
                return parentProcesses.items[0];
            } else {
                return undefined;
            }
        } else {
            return undefined;
        }
    }

const getCollectionParent = (collection: CollectionResource) =>
    async (services: ServiceRepository): Promise<ContainerRequestResource | undefined> => {
        const parentProcesses = await services.containerRequestService.list({
            order: new OrderBuilder<ProcessResource>().addAsc('createdAt').getOrder(),
            filters: new FilterBuilder().addEqual('output_uuid', collection.uuid).getFilters(),
            select: containerRequestFieldsNoMounts,
        });
        if (parentProcesses.items.length > 0) {
            return parentProcesses.items[0];
        } else {
            return undefined;
        }
    }


export const setProjectBreadcrumbs = (uuid: string) =>
    (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        const ancestors = getSidePanelTreeNodeAncestorsIds(uuid)(getState().treePicker);
        const rootUuid = getUserUuid(getState());
        if (uuid === rootUuid || ancestors.find(uuid => uuid === rootUuid)) {
            dispatch(setSidePanelBreadcrumbs(uuid));
        } else {
            dispatch(setSharedWithMeBreadcrumbs(uuid));
            dispatch(activateSidePanelTreeItem(SidePanelTreeCategory.SHARED_WITH_ME));
        }
    };

export const setProcessBreadcrumbs = (processUuid: string) =>
    (dispatch: Dispatch, getState: () => RootState) => {
        const { resources } = getState();
        const process = getProcess(processUuid)(resources);
        if (process) {
            dispatch<any>(setProjectBreadcrumbs(process.containerRequest.ownerUuid));
        }
    };

export const setGroupsBreadcrumbs = () =>
    setBreadcrumbs([{ label: SidePanelTreeCategory.GROUPS }]);

export const setGroupDetailsBreadcrumbs = (groupUuid: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {

        const group = getResource<GroupResource>(groupUuid)(getState().resources);

        const breadcrumbs: Breadcrumb[] = [
            { label: SidePanelTreeCategory.GROUPS, uuid: SidePanelTreeCategory.GROUPS },
            { label: group ? group.name : (await services.groupsService.get(groupUuid)).name, uuid: groupUuid },
        ];

        dispatch(setBreadcrumbs(breadcrumbs));

    };

export const USERS_PANEL_LABEL = 'Users';

export const setUsersBreadcrumbs = () =>
    setBreadcrumbs([{ label: USERS_PANEL_LABEL, uuid: USERS_PANEL_LABEL }]);

export const setUserProfileBreadcrumbs = (userUuid: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        try {
            const user = getResource<UserResource>(userUuid)(getState().resources)
                        || await services.userService.get(userUuid, false);
            const breadcrumbs: Breadcrumb[] = [
                { label: USERS_PANEL_LABEL, uuid: USERS_PANEL_LABEL },
                { label: user ? user.username : userUuid, uuid: userUuid },
            ];
            dispatch(setBreadcrumbs(breadcrumbs));
        } catch (e) {
            const breadcrumbs: Breadcrumb[] = [
                { label: USERS_PANEL_LABEL, uuid: USERS_PANEL_LABEL },
                { label: userUuid, uuid: userUuid },
            ];
            dispatch(setBreadcrumbs(breadcrumbs));
        }
    };

export const MY_ACCOUNT_PANEL_LABEL = 'My Account';

export const setMyAccountBreadcrumbs = () =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(setBreadcrumbs([
            { label: MY_ACCOUNT_PANEL_LABEL, uuid: MY_ACCOUNT_PANEL_LABEL },
        ]));
    };
