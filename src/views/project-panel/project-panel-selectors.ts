// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { TreeItem } from "../../components/tree/tree";
import { Project } from "../../models/project";
import { findTreeItem } from "../../store/project/project-reducer";
import { ResourceKind } from "../../models/resource";
import { Collection } from "../../models/collection";
import { getResourceUrl } from "../../store/navigation/navigation-action";
import { ProjectExplorerItem } from "../../views-components/project-explorer/project-explorer-item";

export const projectExplorerItems = (projects: Array<TreeItem<Project>>, treeItemId: string, collections: Array<Collection>): ProjectExplorerItem[] => {
    const dataItems: ProjectExplorerItem[] = [];

    const treeItem = findTreeItem(projects, treeItemId);
    if (treeItem) {
        dataItems.push({
            name: "..",
            url: getResourceUrl(treeItem.data),
            kind: ResourceKind.LEVEL_UP,
            owner: treeItem.data.ownerUuid,
            uuid: treeItem.data.uuid,
            lastModified: treeItem.data.modifiedAt
        });

        if (treeItem.items) {
            treeItem.items.forEach(p => {
                const item = {
                    name: p.data.name,
                    kind: ResourceKind.PROJECT,
                    url: getResourceUrl(treeItem.data),
                    owner: p.data.ownerUuid,
                    uuid: p.data.uuid,
                    lastModified: p.data.modifiedAt
                } as ProjectExplorerItem;

                dataItems.push(item);
            });
        }
    }

    collections.forEach(c => {
        const item = {
            name: c.name,
            kind: ResourceKind.COLLECTION,
            url: getResourceUrl(c),
            owner: c.ownerUuid,
            uuid: c.uuid,
            lastModified: c.modifiedAt
        } as ProjectExplorerItem;

        dataItems.push(item);
    });

    return dataItems;
};

