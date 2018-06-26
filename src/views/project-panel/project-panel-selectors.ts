// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { TreeItem } from "../../components/tree/tree";
import { Project } from "../../models/project";
import { findTreeItem } from "../../store/project/project-reducer";
import { ResourceKind } from "../../models/resource";
import { Collection } from "../../models/collection";
import { getResourceUrl } from "../../store/navigation/navigation-action";
import { ProjectPanelItem } from "./project-panel-item";

export const projectPanelItems = (projects: Array<TreeItem<Project>>, treeItemId: string, collections: Array<Collection>): ProjectPanelItem[] => {
    const dataItems: ProjectPanelItem[] = [];

    const treeItem = findTreeItem(projects, treeItemId);
    if (treeItem) {
        dataItems.push({
            name: "..",
            url: getResourceUrl(treeItem.data),
            kind: ResourceKind.LEVEL_UP,
            owner: "",
            uuid: treeItem.data.uuid,
            lastModified: ""
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
                } as ProjectPanelItem;

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
        } as ProjectPanelItem;

        dataItems.push(item);
    });

    return dataItems;
};

