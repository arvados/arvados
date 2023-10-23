// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Tree } from "models/tree";
import { TreeItemStatus } from 'components/tree/tree';
export type TreePicker = { [key: string]: Tree<any> };

export const getTreePicker = <Value = {}>(id: string) => (state: TreePicker): Tree<Value> | undefined => state[id];

export const createTreePickerNode = (data: { nodeId: string, value: any }) => ({
    ...data,
    selected: false,
    collapsed: true,
    status: TreeItemStatus.INITIAL
});
