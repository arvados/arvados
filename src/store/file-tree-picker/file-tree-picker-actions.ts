// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { unionize, ofType, UnionOf } from "~/common/unionize";

import { TreePickerNode } from "./file-tree-picker";

export const fileTreePickerActions = unionize({
    LOAD_TREE_PICKER_NODE: ofType<{ nodeId: string, pickerId: string }>(),
    LOAD_TREE_PICKER_NODE_SUCCESS: ofType<{ nodeId: string, nodes: Array<TreePickerNode>, pickerId: string }>(),
    TOGGLE_TREE_PICKER_NODE_COLLAPSE: ofType<{ nodeId: string, pickerId: string }>(),
    TOGGLE_TREE_PICKER_NODE_SELECT: ofType<{ nodeId: string, pickerId: string }>(),
    EXPAND_TREE_PICKER_NODES: ofType<{ nodeIds: string[], pickerId: string }>(),
    RESET_TREE_PICKER: ofType<{ pickerId: string }>()
});

export type FileTreePickerAction = UnionOf<typeof fileTreePickerActions>;
