// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

export type CollectionPanelFilesState = Array<CollectionPanelFile>;

export interface CollectionPanelFile {
    parentId?: string;
    id: string;
    name: string;
    size?: number;
    collapsed: boolean;
    selected: boolean;
    type: string;
}