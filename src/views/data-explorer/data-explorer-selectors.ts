import { TreeItem } from "../../components/tree/tree";
import { Project } from "../../models/project";
import { DataItem } from "../../views-components/data-explorer/data-item";
import { findTreeItem } from "../../store/project/project-reducer";
import { ResourceKind } from "../../models/resource";
import { Collection } from "../../models/collection";


export const projectExplorerItems = (projects: Array<TreeItem<Project>>, treeItemId: string, collections: Array<Collection>): DataItem[] => {
    const dataItems: DataItem[] = [];

    const treeItem = findTreeItem(projects, treeItemId);
    if (treeItem) {
        dataItems.push({
            name: "..",
            url: `/projects/${treeItem.data.ownerUuid}`,
            type: ResourceKind.LEVEL_UP,
            owner: treeItem.data.ownerUuid,
            uuid: treeItem.data.uuid,
            lastModified: treeItem.data.modifiedAt
        });

        if (treeItem.items) {
            treeItem.items.forEach(p => {
                const item = {
                    name: p.data.name,
                    type: ResourceKind.PROJECT,
                    url: `/projects/${treeItem.data.uuid}`,
                    owner: p.data.ownerUuid,
                    uuid: p.data.uuid,
                    lastModified: p.data.modifiedAt
                } as DataItem;

                dataItems.push(item);
            });
        }
    }

    collections.forEach(c => {
        const item = {
            name: c.name,
            type: ResourceKind.COLLECTION,
            url: `/collections/${c.uuid}`,
            owner: c.ownerUuid,
            uuid: c.uuid,
            lastModified: c.modifiedAt
        } as DataItem;

        dataItems.push(item);
    });

    return dataItems;
};

