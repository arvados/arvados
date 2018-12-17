// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from 'redux';
import { RootState } from '~/store/store';
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

export const BREADCRUMBS = 'breadcrumbs';

export interface ResourceBreadcrumb extends Breadcrumb {
    uuid: string;
}

export const setBreadcrumbs = (breadcrumbs: Breadcrumb[]) =>
    propertiesActions.SET_PROPERTY({ key: BREADCRUMBS, value: breadcrumbs });

const getSidePanelTreeBreadcrumbs = (uuid: string) => (treePicker: TreePicker): ResourceBreadcrumb[] => {
    const nodes = getSidePanelTreeBranch(uuid)(treePicker);
    return nodes.map(node =>
        typeof node.value === 'string'
            ? { label: node.value, uuid: node.id }
            : { label: node.value.name, uuid: node.value.uuid });
};

export const setSidePanelBreadcrumbs = (uuid: string) =>
    (dispatch: Dispatch, getState: () => RootState) => {
        const { treePicker } = getState();
        const breadcrumbs = getSidePanelTreeBreadcrumbs(uuid)(treePicker);
        dispatch(setBreadcrumbs(breadcrumbs));
    };

export const setSharedWithMeBreadcrumbs = (uuid: string) =>
    setCategoryBreadcrumbs(uuid, SidePanelTreeCategory.SHARED_WITH_ME);

export const setTrashBreadcrumbs = (uuid: string) =>
    setCategoryBreadcrumbs(uuid, SidePanelTreeCategory.TRASH);

export const setCategoryBreadcrumbs = (uuid: string, category: SidePanelTreeCategory) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const ancestors = await services.ancestorsService.ancestors(uuid, '');
        dispatch(updateResources(ancestors));
        const initialBreadcrumbs: ResourceBreadcrumb[] = [
            { label: category, uuid: category }
        ];
        const breadrumbs = ancestors.reduce((breadcrumbs, ancestor) =>
            ancestor.kind === ResourceKind.GROUP
                ? [...breadcrumbs, { label: ancestor.name, uuid: ancestor.uuid }]
                : breadcrumbs,
            initialBreadcrumbs);

        dispatch(setBreadcrumbs(breadrumbs));
    };

export const setProjectBreadcrumbs = (uuid: string) =>
    (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        const ancestors = getSidePanelTreeNodeAncestorsIds(uuid)(getState().treePicker);
        const rootUuid = services.authService.getUuid();
        if (uuid === rootUuid || ancestors.find(uuid => uuid === rootUuid)) {
            dispatch(setSidePanelBreadcrumbs(uuid));
        } else {
            dispatch(setSharedWithMeBreadcrumbs(uuid));
            dispatch(activateSidePanelTreeItem(SidePanelTreeCategory.SHARED_WITH_ME));
        }
    };

export const setCollectionBreadcrumbs = (collectionUuid: string) =>
    (dispatch: Dispatch, getState: () => RootState) => {
        const { resources } = getState();
        const collection = getResource(collectionUuid)(resources);
        if (collection) {
            dispatch<any>(setProjectBreadcrumbs(collection.ownerUuid));
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
