import { TreeItem } from "../../components/tree/tree";
import { Project } from "../../models/project";
import { DataItem } from "../../views-components/data-explorer/data-item";

export const mapProjectTreeItem = (item: TreeItem<Project>): DataItem => ({
    name: item.data.name,
    type: item.data.kind,
    owner: item.data.ownerUuid,
    lastModified: item.data.modifiedAt,
    uuid: item.data.uuid
});
