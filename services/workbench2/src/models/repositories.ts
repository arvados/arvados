// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Resource } from "models/resource";

export interface RepositoryResource extends Resource {
    name: string;
    cloneUrls: string[];
}
