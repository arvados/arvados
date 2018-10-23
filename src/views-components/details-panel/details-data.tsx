// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { DetailsResource } from "~/models/details";
import { ResourceData } from "~/store/resources-data/resources-data-reducer";

export abstract class DetailsData<T extends DetailsResource = DetailsResource> {
    constructor(protected item: T, protected data?: ResourceData) {}

    getTitle(): string {
        return this.item.name || 'Projects';
    }

    abstract getIcon(className?: string): React.ReactElement<any>;
    abstract getDetails(): React.ReactElement<any>;

    getActivity(): React.ReactElement<any> {
        return <div/>;
    }
}
