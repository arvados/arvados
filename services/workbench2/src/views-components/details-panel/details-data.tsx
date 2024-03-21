// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { DetailsResource } from "models/details";
import { ResourceKind } from 'models/resource';

interface GetDetailsParams {
  tabNr?: number
  showPreview?: boolean
}

export abstract class DetailsData<T extends DetailsResource = DetailsResource> {
    constructor(protected item: T) { }

    getTitle(): string {
        if((this.item as any).kind === ResourceKind.USER) return 'Home Projects'
        return this.item.name || 'Projects';
    }

    getTabLabels(): string[] {
        return ['Details'];
    }

    abstract getIcon(className?: string): React.ReactElement<any>;
    abstract getDetails({tabNr, showPreview}: GetDetailsParams): React.ReactElement<any>;
}
