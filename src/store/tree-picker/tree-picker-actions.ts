// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { default as unionize, ofType, UnionOf } from "unionize";

import { TreePickerNode } from "./tree-picker";

export const treePickerActions = unionize({
    LOAD_TREE_PICKER_NODE: ofType<{ nodeId: string, pickerId: string }>(),
    LOAD_TREE_PICKER_NODE_SUCCESS: ofType<{ nodeId: string, nodes: Array<TreePickerNode>, pickerId: string }>(),
    TOGGLE_TREE_PICKER_NODE_COLLAPSE: ofType<{ nodeId: string, pickerId: string }>(),
    TOGGLE_TREE_PICKER_NODE_SELECT: ofType<{ nodeId: string, pickerId: string }>()
}, {
        tag: 'type',
        value: 'payload'
    });

export type TreePickerAction = UnionOf<typeof treePickerActions>;
