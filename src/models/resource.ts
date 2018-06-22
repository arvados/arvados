export interface Resource {
    name: string;
    createdAt: string;
    modifiedAt: string;
    uuid: string;
    ownerUuid: string;
    href: string;
    kind: string;
}

export enum ResourceKind {
    PROJECT = "project",
    COLLECTION = "collection",
    PIPELINE = "pipeline",
    LEVEL_UP = "levelup",
    UNKNOWN = "unknown"
}

export function getResourceKind(itemKind: string) {
    switch (itemKind) {
        case "arvados#project": return ResourceKind.PROJECT;
        case "arvados#collection": return ResourceKind.COLLECTION;
        case "arvados#pipeline": return ResourceKind.PIPELINE;
        default:
            return ResourceKind.UNKNOWN;
    }
}
