// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { unionize, ofType, UnionOf } from "unionize";
import { Dispatch } from "redux";
import { ResourceKind } from "../../models/resource";
import { CollectionResource } from "../../models/collection";
import { collectionPanelFilesAction } from "./collection-panel-files/collection-panel-files-actions";
import { createTree } from "../../models/tree";
import { RootState } from "../store";
import { ServiceRepository } from "../../services/services";
import { TagResource, TagProperty } from "../../models/tag";
import { snackbarActions } from "../snackbar/snackbar-actions";

export const collectionPanelActions = unionize({
    LOAD_COLLECTION: ofType<{ uuid: string, kind: ResourceKind }>(),
    LOAD_COLLECTION_SUCCESS: ofType<{ item: CollectionResource }>(),
    LOAD_COLLECTION_TAGS: ofType<{ uuid: string }>(),
    LOAD_COLLECTION_TAGS_SUCCESS: ofType<{ tags: TagResource[] }>(),
    CREATE_COLLECTION_TAG: ofType<{ data: any }>(),
    CREATE_COLLECTION_TAG_SUCCESS: ofType<{ tag: TagResource }>(),
    DELETE_COLLECTION_TAG: ofType<{ uuid: string }>(),
    DELETE_COLLECTION_TAG_SUCCESS: ofType<{ uuid: string }>()
}, { tag: 'type', value: 'payload' });

export type CollectionPanelAction = UnionOf<typeof collectionPanelActions>;

export const COLLECTION_TAG_FORM_NAME = 'collectionTagForm';

export const loadCollection = (uuid: string, kind: ResourceKind) =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(collectionPanelActions.LOAD_COLLECTION({ uuid, kind }));
        dispatch(collectionPanelFilesAction.SET_COLLECTION_FILES({ files: createTree() }));
        return services.collectionService
            .get(uuid)
            .then(item => {
                dispatch(collectionPanelActions.LOAD_COLLECTION_SUCCESS({ item }));
                return services.collectionService.files(item.uuid);
            })
            .then(files => {
                dispatch(collectionPanelFilesAction.SET_COLLECTION_FILES(files));
            });
    };

export const loadCollectionTags = (uuid: string) => 
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(collectionPanelActions.LOAD_COLLECTION_TAGS({ uuid }));
        return services.tagService
            .list(uuid)
            .then(tags => {
                dispatch(collectionPanelActions.LOAD_COLLECTION_TAGS_SUCCESS({ tags }));
            });
    };


export const createCollectionTag = (data: TagProperty) => 
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(collectionPanelActions.CREATE_COLLECTION_TAG({ data }));
        const item = getState().collectionPanel.item;
        const uuid = item ? item.uuid : '';
        return services.tagService
            .create(uuid, data)
            .then(tag => {
                dispatch(collectionPanelActions.CREATE_COLLECTION_TAG_SUCCESS({ tag }));
                dispatch(snackbarActions.OPEN_SNACKBAR({
                    message: "Tag has been successfully added.",
                    hideDuration: 2000
                }));
            });
    };

export const deleteCollectionTag = (uuid: string) => 
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        dispatch(collectionPanelActions.DELETE_COLLECTION_TAG({ uuid }));
        return services.linkService
            .delete(uuid)
            .then(tag => {
                dispatch(collectionPanelActions.DELETE_COLLECTION_TAG_SUCCESS({ uuid: tag.uuid }));
                dispatch(snackbarActions.OPEN_SNACKBAR({
                    message: "Tag has been successfully deleted.",
                    hideDuration: 2000
                }));
            });
    };