// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from 'redux';
import { RootState } from 'store/store';
import { getUserUuid } from "common/getuser";
import { getResource } from 'store/resources/resources';
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
import { AdminMenuIcon, CollectionIcon, IconType, ProcessIcon, ProjectIcon, ResourceIcon, TerminalIcon, WorkflowIcon } from 'components/icon/icon';
import { CollectionResource } from 'models/collection';
import { getSidePanelIcon } from 'views-components/side-panel-tree/side-panel-tree';
import { WorkflowResource } from 'models/workflow';
import { progressIndicatorActions } from "store/progress-indicator/progress-indicator-actions";

export const BREADCRUMBS = 'breadcrumbs';

export const setBreadcrumbs = (breadcrumbs: any, currentItem?: CollectionResource | ContainerRequestResource | GroupResource | WorkflowResource) => {
    if (currentItem) {
        const currentCrumb = resourceToBreadcrumb(currentItem)
        if (currentCrumb.label.length) breadcrumbs.push(currentCrumb);
    }
    return propertiesActions.SET_PROPERTY({ key: BREADCRUMBS, value: breadcrumbs });
};

const resourceToBreadcrumbIcon = (resource: CollectionResource | ContainerRequestResource | GroupResource | WorkflowResource): IconType | undefined => {
    switch (resource.kind) {
        case ResourceKind.PROJECT:
            return ProjectIcon;
        case ResourceKind.PROCESS:
            return ProcessIcon;
        case ResourceKind.COLLECTION:
            return CollectionIcon;
        case ResourceKind.WORKFLOW:
            return WorkflowIcon;
        default:
            return undefined;
    }
}

const resourceToBreadcrumb = (resource: (CollectionResource | ContainerRequestResource | GroupResource | WorkflowResource) & {fullName?: string}  ): Breadcrumb => ({
    label: resource.name || resource.fullName || '',
    uuid: resource.uuid,
    icon: resourceToBreadcrumbIcon(resource),
})

export const setSidePanelBreadcrumbs = (uuid: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        try {
            dispatch(progressIndicatorActions.START_WORKING(uuid + "-breadcrumbs"));
            const ancestors = await services.ancestorsService.ancestors(uuid, '');
            dispatch(updateResources(ancestors));

            let breadcrumbs: Breadcrumb[] = [];
            const { collectionPanel: { item } } = getState();

            const path = getState().router.location!.pathname;
            const currentUuid = path.split('/')[2];
            const uuidKind = extractUuidKind(currentUuid);
            const rootUuid = getUserUuid(getState());

            if (ancestors.find(ancestor => ancestor.uuid === rootUuid)) {
                // Handle home project uuid root
                breadcrumbs.push({
                    label: SidePanelTreeCategory.PROJECTS,
                    uuid: SidePanelTreeCategory.PROJECTS,
                    icon: getSidePanelIcon(SidePanelTreeCategory.PROJECTS)
                });
            } else if (Object.values(SidePanelTreeCategory).includes(uuid as SidePanelTreeCategory)) {
                // Handle SidePanelTreeCategory root
                breadcrumbs.push({
                    label: uuid,
                    uuid: uuid,
                    icon: getSidePanelIcon(uuid)
                });
            }

            breadcrumbs = ancestors.reduce((breadcrumbs, ancestor) =>
                ancestor.kind === ResourceKind.GROUP
                    ? [...breadcrumbs, resourceToBreadcrumb(ancestor)]
                    : breadcrumbs,
                breadcrumbs);

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
            } else if (uuidKind === ResourceKind.WORKFLOW) {
                const workflowItem = await services.workflowService.get(currentUuid);
                dispatch(setBreadcrumbs(breadcrumbs, workflowItem));
            }
            dispatch(setBreadcrumbs(breadcrumbs));
        } finally {
            dispatch(progressIndicatorActions.STOP_WORKING(uuid + "-breadcrumbs"));
        }
    };

export const setSharedWithMeBreadcrumbs = (uuid: string) =>
    setCategoryBreadcrumbs(uuid, SidePanelTreeCategory.SHARED_WITH_ME);

export const setTrashBreadcrumbs = (uuid: string) =>
    setCategoryBreadcrumbs(uuid, SidePanelTreeCategory.TRASH);

export const setCategoryBreadcrumbs = (uuid: string, category: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        try {
            dispatch(progressIndicatorActions.START_WORKING(uuid + "-breadcrumbs"));
            const ancestors = await services.ancestorsService.ancestors(uuid, '');
            dispatch(updateResources(ancestors));
            const initialBreadcrumbs: Breadcrumb[] = [
                {
                    label: category,
                    uuid: category,
                    icon: getSidePanelIcon(category)
                }
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
            } else if (uuidKind === ResourceKind.WORKFLOW) {
                const workflowItem = await services.workflowService.get(currentUuid);
                dispatch(setBreadcrumbs(breadcrumbs, workflowItem));
            }
            dispatch(setBreadcrumbs(breadcrumbs));
        } finally {
            dispatch(progressIndicatorActions.STOP_WORKING(uuid + "-breadcrumbs"));
        }
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
        const parentOutputPromise = services.containerRequestService.list({
            order: new OrderBuilder<ProcessResource>().addAsc('createdAt').getOrder(),
            filters: new FilterBuilder().addEqual('output_uuid', collection.uuid).getFilters(),
            select: containerRequestFieldsNoMounts,
        });
        const parentLogPromise = services.containerRequestService.list({
            order: new OrderBuilder<ProcessResource>().addAsc('createdAt').getOrder(),
            filters: new FilterBuilder().addEqual('log_uuid', collection.uuid).getFilters(),
            select: containerRequestFieldsNoMounts,
        });
        const [parentOutput, parentLog] = await Promise.all([parentOutputPromise, parentLogPromise]);
        return parentOutput.items.length > 0 ?
            parentOutput.items[0] :
            parentLog.items.length > 0 ?
                parentLog.items[0] :
                undefined;
    }


export const setProjectBreadcrumbs = (uuid: string) =>
    async (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        const ancestors = await services.ancestorsService.ancestors(uuid, '');
        const rootUuid = getUserUuid(getState());
        if (uuid === rootUuid || ancestors.find(ancestor => ancestor.uuid === rootUuid)) {
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
    setBreadcrumbs([{
        label: SidePanelTreeCategory.GROUPS,
        uuid: SidePanelTreeCategory.GROUPS,
        icon: getSidePanelIcon(SidePanelTreeCategory.GROUPS)
    }]);

export const setGroupDetailsBreadcrumbs = (groupUuid: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {

        const group = getResource<GroupResource>(groupUuid)(getState().resources);

        const breadcrumbs: Breadcrumb[] = [
            {
                label: SidePanelTreeCategory.GROUPS,
                uuid: SidePanelTreeCategory.GROUPS,
                icon: getSidePanelIcon(SidePanelTreeCategory.GROUPS)
            },
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
                { label: user ? `${user.firstName} ${user.lastName}` : userUuid, uuid: userUuid },
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

export const INSTANCE_TYPES_PANEL_LABEL = 'Instance Types';

export const setInstanceTypesBreadcrumbs = () =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(setBreadcrumbs([
            { label: INSTANCE_TYPES_PANEL_LABEL, uuid: INSTANCE_TYPES_PANEL_LABEL, icon: ResourceIcon },
        ]));
    };

export const VIRTUAL_MACHINES_USER_PANEL_LABEL = 'Shell Access';

export const setVirtualMachinesBreadcrumbs = () =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(setBreadcrumbs([
            { label: VIRTUAL_MACHINES_USER_PANEL_LABEL, uuid: VIRTUAL_MACHINES_USER_PANEL_LABEL, icon: TerminalIcon },
        ]));
    };

export const VIRTUAL_MACHINES_ADMIN_PANEL_LABEL = 'Shell Access Admin';

export const setVirtualMachinesAdminBreadcrumbs = () =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(setBreadcrumbs([
            { label: VIRTUAL_MACHINES_ADMIN_PANEL_LABEL, uuid: VIRTUAL_MACHINES_ADMIN_PANEL_LABEL, icon: TerminalIcon },
        ]));
    };

export const REPOSITORIES_PANEL_LABEL = 'Repositories';

export const setRepositoriesBreadcrumbs = () =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(setBreadcrumbs([
            { label: REPOSITORIES_PANEL_LABEL, uuid: REPOSITORIES_PANEL_LABEL, icon: AdminMenuIcon },
        ]));
    };
