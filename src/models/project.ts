// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { GroupClass, GroupResource } from "./group";

export interface ProjectResource extends GroupResource {
    groupClass: GroupClass.PROJECT | GroupClass.FILTER;
}

export const getProjectUrl = (uuid: string) => {
    return `/projects/${uuid}`;
};
