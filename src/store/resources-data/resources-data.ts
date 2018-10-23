// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ResourceData, ResourcesDataState } from "~/store/resources-data/resources-data-reducer";

export const getResourceData = (id: string) =>
    (state: ResourcesDataState): ResourceData | undefined => state[id];
