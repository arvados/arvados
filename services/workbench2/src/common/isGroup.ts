// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ResourceKind } from "models/resource";
import { GroupClass } from "models/group";

export const isGroup = (resource: any) => {
    return resource && resource.kind === ResourceKind.GROUP && resource.groupClass === GroupClass.ROLE;
};

