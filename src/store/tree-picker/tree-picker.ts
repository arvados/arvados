// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Tree } from "~/models/tree";
import { TreeItemStatus } from "~/components/tree/tree";

export type TreePicker = Tree<TreePickerNode>;

export interface TreePickerNode {
    id: string;
    value: any;
    selected: boolean;
    collapsed: boolean;
    status: TreeItemStatus;
}

export const createTreePickerNode = (data: {id: string, value: any}) => ({
    ...data,
    selected: false,
    collapsed: true,
    status: TreeItemStatus.INITIAL
});
