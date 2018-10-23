// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ResourcesDataActions, resourcesDataActions } from "~/store/resources-data/resources-data-actions";
import { getNodeDescendantsIds, TREE_ROOT_ID } from "~/models/tree";
import { CollectionFileType } from "~/models/collection-file";

export interface ResourceData {
    fileCount: number;
    fileSize: number;
}

export type ResourcesDataState = {
    [key: string]: ResourceData
};

export const resourcesDataReducer = (state: ResourcesDataState = {}, action: ResourcesDataActions) =>
    resourcesDataActions.match(action, {
        SET_FILES: ({uuid, files}) => {
            const flattenFiles = getNodeDescendantsIds(TREE_ROOT_ID)(files).map(id => files[id]);
            const [fileSize, fileCount] = flattenFiles.reduce(([size, cnt], f) =>
                f && f.value.type === CollectionFileType.FILE
                ? [size + f.value.size, cnt + 1]
                : [size, cnt]
            , [0, 0]);
            return {
                ...state,
                [uuid]: { fileCount, fileSize }
            };
        },
        default: () => state,
    });
