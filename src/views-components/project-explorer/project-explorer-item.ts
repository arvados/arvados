// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { TreeItem } from "../../components/tree/tree";
import { Project } from "../../models/project";

export interface ProjectExplorerItem {
    uuid: string;
    name: string;
    type: string;
    owner: string;
    lastModified: string;
    fileSize?: number;
    status?: string;
}

export const mapProjectTreeItem = (item: TreeItem<Project>): ProjectExplorerItem => ({
    name: item.data.name,
    type: item.data.kind,
    owner: item.data.ownerUuid,
    lastModified: item.data.modifiedAt,
    uuid: item.data.uuid
});