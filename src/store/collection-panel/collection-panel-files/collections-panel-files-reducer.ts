// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { CollectionPanelFilesState, CollectionPanelFile } from "./collection-panel-files-state";
import { CollectionPanelFilesAction, collectionPanelFilesAction } from "./collection-panel-files-actions";
import { stat } from "fs";

const initialState: CollectionPanelFilesState = [{
    collapsed: true,
    id: 'Directory 1',
    name: 'Directory 1',
    selected: false,
    type: 'directory',
}, {
    parentId: 'Directory 1',
    collapsed: true,
    id: 'Directory 1.1',
    name: 'Directory 1.1',
    selected: false,
    type: 'directory',
}, {
    parentId: 'Directory 1',
    collapsed: true,
    id: 'File 1.1',
    name: 'File 1.1',
    selected: false,
    type: 'file',
}, {
    collapsed: true,
    id: 'Directory 2',
    name: 'Directory 2',
    selected: false,
    type: 'directory',
}, {
    parentId: 'Directory 2',
    collapsed: true,
    id: 'Directory 2.1',
    name: 'Directory 2.1',
    selected: false,
    type: 'directory',
}, {
    parentId: 'Directory 2.1',
    collapsed: true,
    id: 'Directory 2.1.1',
    name: 'Directory 2.1.1',
    selected: false,
    type: 'directory',
}, {
    parentId: 'Directory 2.1.1',
    collapsed: true,
    id: 'Directory 2.1.1.1',
    name: 'Directory 2.1.1.1',
    selected: false,
    type: 'directory',
}];

export const collectionPanelFilesReducer = (state: CollectionPanelFilesState = initialState, action: CollectionPanelFilesAction) => {
    return collectionPanelFilesAction.match(action, {
        SET_COLLECTION_FILES: data => data.files,
        TOGGLE_COLLECTION_FILE_COLLAPSE: data => toggleCollapsed(state, data.id),
        TOGGLE_COLLECTION_FILE_SELECTION: data => toggleSelected(state, data.id),
        default: () => state
    });
};

const toggleCollapsed = (state: CollectionPanelFilesState, id: string) =>
    state.map(file => file.id === id
        ? { ...file, collapsed: !file.collapsed }
        : file);

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

const toggleAncestors = (state: CollectionPanelFilesState, id: string): CollectionPanelFile[] => {
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

const getDescendants = (state: CollectionPanelFilesState) => ({ id }: { id: string }): CollectionPanelFile[] => {
    const root = state.find(f => f.id === id);
    if (root) {
        return [root].concat(...state.filter(f => f.parentId === id).map(getDescendants(state)));
    } else { return []; }
};

