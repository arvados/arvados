// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Tree } from 'models/tree';

export interface DataTableFilterItem {
    name: string;
}

export type DataTableFilters = Tree<DataTableFilterItem>;
