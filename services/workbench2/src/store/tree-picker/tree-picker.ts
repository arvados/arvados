// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Tree } from "models/tree";
import { TreeNodeStatus } from 'models/tree';

export type TreePicker = { [key: string]: Tree<any> };

export const getTreePicker = <Value = {}>(id: string) => (state: TreePicker): Tree<Value> | undefined => state[id];

export const createTreePickerNode = (data: { nodeId: string, value: any }) => ({
    ...data,
    selected: false,
    collapsed: true,
    status: TreeNodeStatus.INITIAL
});

