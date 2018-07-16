// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContextMenuHOC, addMenuActionSet } from "./context-menu";
import { projectActionSet } from "./action-sets/project-action-set";
import { rootProjectActionSet } from "./action-sets/root-project-action-set";

export default ContextMenuHOC;

export enum ContextMenuKind {
    RootProject = "RootProject",
    Project = "Project"
}

addMenuActionSet(ContextMenuKind.RootProject, rootProjectActionSet);
addMenuActionSet(ContextMenuKind.Project, projectActionSet);