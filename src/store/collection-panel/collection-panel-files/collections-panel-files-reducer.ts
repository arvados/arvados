// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { CollectionPanelFilesState, CollectionPanelFile, CollectionPanelDirectory, CollectionPanelItem, mapManifestToItems } from "./collection-panel-files-state";
import { CollectionPanelFilesAction, collectionPanelFilesAction } from "./collection-panel-files-actions";

export const collectionPanelFilesReducer = (state: CollectionPanelFilesState = [], action: CollectionPanelFilesAction) => {
    return collectionPanelFilesAction.match(action, {
        SET_COLLECTION_FILES: ({manifest}) => mapManifestToItems(manifest),
        TOGGLE_COLLECTION_FILE_COLLAPSE: data => toggleCollapsed(state, data.id),
        TOGGLE_COLLECTION_FILE_SELECTION: data => toggleSelected(state, data.id),
        SELECT_ALL_COLLECTION_FILES: () => state.map(file => ({ ...file, selected: true })),
        UNSELECT_ALL_COLLECTION_FILES: () => state.map(file => ({ ...file, selected: false })),
        default: () => state
    });
};

const toggleCollapsed = (state: CollectionPanelFilesState, id: string) =>
    state.map((item: CollectionPanelItem) =>
        item.type === 'directory' && item.id === id
            ? { ...item, collapsed: !item.collapsed }
            : item);

const toggleSelected = (state: CollectionPanelFilesState, id: string) =>
    toggleAncestors(toggleDescendants(state, id), id);

const toggleDescendants = (state: CollectionPanelFilesState, id: string) => {
    const ids = getDescendants(state)({ id }).map(file => file.id);
    if (ids.length > 0) {
        const selected = !state.find(f => f.id === ids[0])!.selected;
        return state.map(file => ids.some(id => file.id === id) ? { ...file, selected } : file);
    }
    return state;
};

const toggleAncestors = (state: CollectionPanelFilesState, id: string): CollectionPanelItem[] => {
    const file = state.find(f => f.id === id);
    if (file) {
        const selected = state
            .filter(f => f.parentId === file.parentId)
            .every(f => f.selected);
        if (!selected) {
            const newState = state.map(f => f.id === file.parentId ? { ...f, selected } : f);
            return toggleAncestors(newState, file.parentId || "");
        }
    }
    return state;
};

const getDescendants = (state: CollectionPanelFilesState) => ({ id }: { id: string }): CollectionPanelItem[] => {
    const root = state.find(item => item.id === id);
    if (root) {
        return [root].concat(...state.filter(item => item.parentId === id).map(getDescendants(state)));
    } else { return []; }
};

