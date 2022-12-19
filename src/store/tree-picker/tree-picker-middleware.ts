// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from 'redux';
import { RootState } from 'store/store';
import { ServiceRepository } from 'services/services';
import { Middleware } from "redux";
import { getNode, getNodeDescendantsIds, TreeNodeStatus } from 'models/tree';
import { getTreePicker } from './tree-picker';
import {
    treePickerSearchActions, loadProject, loadFavoritesProject, loadPublicFavoritesProject,
    SHARED_PROJECT_ID, FAVORITES_PROJECT_ID, PUBLIC_FAVORITES_PROJECT_ID, SEARCH_PROJECT_ID
} from "./tree-picker-actions";
import { LinkResource } from "models/link";
import { GroupContentsResource } from 'services/groups-service/groups-service';
import { CollectionDirectory, CollectionFile } from 'models/collection-file';

export interface ProjectsTreePickerRootItem {
    id: string;
    name: string;
}

export type ProjectsTreePickerItem = ProjectsTreePickerRootItem | GroupContentsResource | CollectionDirectory | CollectionFile | LinkResource;

export const treePickerSearchMiddleware: Middleware = store => next => action => {
    let isSearchAction = false;
    let searchChanged = false;

    treePickerSearchActions.match(action, {
        SET_TREE_PICKER_PROJECT_SEARCH: ({ pickerId, projectSearchValue }) => {
            isSearchAction = true;
            searchChanged = store.getState().treePickerSearch.projectSearchValues[pickerId] !== projectSearchValue;
        },

        SET_TREE_PICKER_COLLECTION_FILTER: ({ pickerId, collectionFilterValue }) => {
            isSearchAction = true;
            searchChanged = store.getState().treePickerSearch.collectionFilterValues[pickerId] !== collectionFilterValue;
        },
        default: () => { }
    });

    if (isSearchAction && !searchChanged) {
        return;
    }

    // pass it on to the reducer
    const r = next(action);

    treePickerSearchActions.match(action, {
        SET_TREE_PICKER_PROJECT_SEARCH: ({ pickerId }) =>
            store.dispatch<any>((dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
                const picker = getTreePicker<ProjectsTreePickerItem>(pickerId)(getState().treePicker);
                if (picker) {
                    const loadParams = getState().treePickerSearch.loadProjectParams[pickerId];
                    dispatch<any>(loadProject({
                        ...loadParams,
                        id: SEARCH_PROJECT_ID,
                        pickerId: pickerId,
                        searchProjects: true
                    }));
                }
            }),

        SET_TREE_PICKER_COLLECTION_FILTER: ({ pickerId }) =>
            store.dispatch<any>((dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
                const picker = getTreePicker<ProjectsTreePickerItem>(pickerId)(getState().treePicker);
                if (picker) {
                    const loadParams = getState().treePickerSearch.loadProjectParams[pickerId];
                    getNodeDescendantsIds('')(picker)
                        .map(id => {
                            const node = getNode(id)(picker);
                            if (node && node.status !== TreeNodeStatus.INITIAL) {
                                if (node.id.substring(6, 11) === 'tpzed' || node.id.substring(6, 11) === 'j7d0g') {
                                    dispatch<any>(loadProject({
                                        ...loadParams,
                                        id: node.id,
                                        pickerId: pickerId,
                                    }));
                                }
                                if (node.id === SHARED_PROJECT_ID) {
                                    dispatch<any>(loadProject({
                                        ...loadParams,
                                        id: node.id,
                                        pickerId: pickerId,
                                        loadShared: true
                                    }));
                                }
                                if (node.id === SEARCH_PROJECT_ID) {
                                    dispatch<any>(loadProject({
                                        ...loadParams,
                                        id: node.id,
                                        pickerId: pickerId,
                                        searchProjects: true
                                    }));
                                }
                                if (node.id === FAVORITES_PROJECT_ID) {
                                    dispatch<any>(loadFavoritesProject({
                                        ...loadParams,
                                        pickerId: pickerId,
                                    }));
                                }
                                if (node.id === PUBLIC_FAVORITES_PROJECT_ID) {
                                    dispatch<any>(loadPublicFavoritesProject({
                                        ...loadParams,
                                        pickerId: pickerId,
                                    }));
                                }
                            }
                            return id;
                        });
                }
            }),
        default: () => { }
    });

    return r;
}
