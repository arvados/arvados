// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ActionType, createStandardAction } from "typesafe-actions";
import { Project } from "../models/project";

export const actions = {
    createProject: createStandardAction('@@project/create')<Project>(),
    removeProject: createStandardAction('@@project/remove')<string>()
};

export type ProjectAction = ActionType<typeof actions>;
