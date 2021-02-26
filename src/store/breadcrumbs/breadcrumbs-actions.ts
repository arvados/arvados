// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from 'redux';
import { RootState } from '~/store/store';
import { getUserUuid } from "~/common/getuser";
import { Breadcrumb } from '~/components/breadcrumbs/breadcrumbs';
import { getResource } from '~/store/resources/resources';
import { TreePicker } from '../tree-picker/tree-picker';
import { getSidePanelTreeBranch, getSidePanelTreeNodeAncestorsIds } from '../side-panel-tree/side-panel-tree-actions';
import { propertiesActions } from '../properties/properties-actions';
import { getProcess } from '~/store/processes/process';
import { ServiceRepository } from '~/services/services';
import { SidePanelTreeCategory, activateSidePanelTreeItem } from '~/store/side-panel-tree/side-panel-tree-actions';
import { updateResources } from '../resources/resources-actions';
import { ResourceKind } from '~/models/resource';
import { GroupResource } from '~/models/group';
import { extractUuidKind } from '~/models/resource';

export const BREADCRUMBS = 'breadcrumbs';

export interface ResourceBreadcrumb extends Breadcrumb {
    uuid: string;
}

export const setBreadcrumbs = (breadcrumbs: any, currentItem?: any) => {
    if (currentItem) {
        const addLastItem = { label: currentItem.name, uuid: currentItem.uuid };
        breadcrumbs.push(addLastItem);
    }
    return propertiesActions.SET_PROPERTY({ key: BREADCRUMBS, value: breadcrumbs });
};


const getSidePanelTreeBreadcrumbs = (uuid: string) => (treePicker: TreePicker): ResourceBreadcrumb[] => {
    const nodes = getSidePanelTreeBranch(uuid)(treePicker);
    return nodes.map(node =>
        typeof node.value === 'string'
            ? { label: node.value, uuid: node.id }
            : { label: node.value.name, uuid: node.value.uuid });
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
            dispatch(setBreadcrumbs(breadcrumbs, collectionItem));
        } else if (uuidKind === ResourceKind.PROCESS) {
            const processItem = await services.containerRequestService.get(currentUuid);
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
        const initialBreadcrumbs: ResourceBreadcrumb[] = [
            { label: category, uuid: category }
        ];
        const { collectionPanel: { item } } = getState();
        const path = getState().router.location!.pathname;
        const currentUuid = path.split('/')[2];
        const uuidKind = extractUuidKind(currentUuid);
        const breadcrumbs = ancestors.reduce((breadcrumbs, ancestor) =>
            ancestor.kind === ResourceKind.GROUP
                ? [...breadcrumbs, { label: ancestor.name, uuid: ancestor.uuid }]
                : breadcrumbs,
            initialBreadcrumbs);
        if (uuidKind === ResourceKind.COLLECTION) {
            const collectionItem = item ? item : await services.collectionService.get(currentUuid);
            dispatch(setBreadcrumbs(breadcrumbs, collectionItem));
        } else if (uuidKind === ResourceKind.PROCESS) {
            const processItem = await services.containerRequestService.get(currentUuid);
            dispatch(setBreadcrumbs(breadcrumbs, processItem));
        }
        dispatch(setBreadcrumbs(breadcrumbs));
    };

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

export const GROUPS_PANEL_LABEL = 'Groups';

export const setGroupsBreadcrumbs = () =>
    setBreadcrumbs([{ label: GROUPS_PANEL_LABEL }]);

export const setGroupDetailsBreadcrumbs = (groupUuid: string) =>
    (dispatch: Dispatch, getState: () => RootState) => {

        const group = getResource<GroupResource>(groupUuid)(getState().resources);

        const breadcrumbs: ResourceBreadcrumb[] = [
            { label: GROUPS_PANEL_LABEL, uuid: GROUPS_PANEL_LABEL },
            { label: group ? group.name : groupUuid, uuid: groupUuid },
        ];

        dispatch(setBreadcrumbs(breadcrumbs));

    };
