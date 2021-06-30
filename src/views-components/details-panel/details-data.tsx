// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { DetailsResource } from "models/details";

export abstract class DetailsData<T extends DetailsResource = DetailsResource> {
    constructor(protected item: T) { }

    getTitle(): string {
        return this.item.name || 'Projects';
    }

    getTabLabels(): string[] {
        return ['Details'];
    }

    abstract getIcon(className?: string): React.ReactElement<any>;
    abstract getDetails(tabNr?: number): React.ReactElement<any>;
}
