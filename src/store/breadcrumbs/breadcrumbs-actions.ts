// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from 'redux';
import { RootState } from '~/store/store';
import { Breadcrumb } from '~/components/breadcrumbs/breadcrumbs';
import { getResource } from '~/store/resources/resources';
import { TreePicker } from '../tree-picker/tree-picker';
import { getSidePanelTreeBranch } from '../side-panel-tree/side-panel-tree-actions';
import { propertiesActions } from '../properties/properties-actions';
import { getProcess } from '~/store/processes/process';
import { ServiceRepository } from '~/services/services';
import { SidePanelTreeCategory } from '~/store/side-panel-tree/side-panel-tree-actions';
import { updateResources } from '../resources/resources-actions';
import { ResourceKind } from '~/models/resource';

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
            ? { label: node.value, uuid: node.nodeId }
            : { label: node.value.name, uuid: node.value.uuid });
};

export const setSidePanelBreadcrumbs = (uuid: string) =>
    (dispatch: Dispatch, getState: () => RootState) => {
        const { treePicker } = getState();
        const breadcrumbs = getSidePanelTreeBreadcrumbs(uuid)(treePicker);
        dispatch(setBreadcrumbs(breadcrumbs));
    };

export const setSharedWithMeBreadcrumbs = (uuid: string) =>
    async (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const ancestors = await services.ancestorsService.ancestors(uuid, '');
        dispatch(updateResources(ancestors));
        const initialBreadcrumbs: ResourceBreadcrumb[] = [
            { label: SidePanelTreeCategory.SHARED_WITH_ME, uuid: SidePanelTreeCategory.SHARED_WITH_ME }
        ];
        const breadrumbs = ancestors.reduce((breadcrumbs, ancestor) =>
            ancestor.kind === ResourceKind.GROUP
                ? [...breadcrumbs, { label: ancestor.name, uuid: ancestor.uuid }]
                : breadcrumbs,
            initialBreadcrumbs);

        dispatch(setBreadcrumbs(breadrumbs));
    };

export const setProjectBreadcrumbs = setSidePanelBreadcrumbs;

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
