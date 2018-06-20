// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0
import { default as unionize, ofType, UnionOf } from "unionize";

import { Project } from "../../models/project";

const actions = unionize({
    CREATE_PROJECT: ofType<Project>(),
    REMOVE_PROJECT: ofType<string>(),
    PROJECTS_REQUEST: ofType<any>(),
    PROJECTS_SUCCESS: ofType<{ projects: Project[], parentItemId?: string }>(),
    TOGGLE_PROJECT_TREE_ITEM_OPEN: ofType<string>(),
    TOGGLE_PROJECT_TREE_ITEM_ACTIVE: ofType<string>(),
}, {
    tag: 'type',
    value: 'payload'
});

export type ProjectAction = UnionOf<typeof actions>;
export default actions;
