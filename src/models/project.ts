// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { GroupClass, GroupResource } from "./group";

export interface ProjectResource extends GroupResource {
    frozenByUuid: null|string;
    groupClass: GroupClass.PROJECT | GroupClass.FILTER | GroupClass.ROLE;
}

export const getProjectUrl = (uuid: string) => {
    return `/projects/${uuid}`;
};
