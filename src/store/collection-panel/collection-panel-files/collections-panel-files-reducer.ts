// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { CollectionPanelFilesState } from "./collection-panel-files-state";
import { CollectionPanelFilesAction, collectionPanelFilesAction } from "./collection-panel-files-actions";

export const collectionPanelFilesReducer = (state: CollectionPanelFilesState = [], action: CollectionPanelFilesAction) => {
    return collectionPanelFilesAction.match(action, {
        SET_COLLECTION_FILES: data => data.files,
        TOGGLE_COLLECTION_FILE_COLLAPSE: data => toggle(state, data.id, "collapsed"),
        TOGGLE_COLLECTION_FILE_SELECTION: data => toggle(state, data.id, "selected"),
        default: () => state
    });
};

const toggle = (state: CollectionPanelFilesState, id: string, key: "collapsed" | "selected") =>
    state.map(file => file.id === id
        ? { ...file, [key]: !file[key] }
        : file);