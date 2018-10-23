// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { unionize, ofType, UnionOf } from "~/common/unionize";
import { CollectionDirectory, CollectionFile } from "~/models/collection-file";
import { Tree } from "~/models/tree";

export const resourcesDataActions = unionize({
    SET_FILES: ofType<{uuid: string, files: Tree<CollectionFile | CollectionDirectory>}>()
});

export type ResourcesDataActions = UnionOf<typeof resourcesDataActions>;
