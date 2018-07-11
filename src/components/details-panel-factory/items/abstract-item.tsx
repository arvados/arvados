// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { DetailsPanelResource } from '../../../views-components/details-panel/details-panel';

export default abstract class AbstractItem<T extends DetailsPanelResource = DetailsPanelResource> {

    constructor(protected item: T) {}

    getTitle(): string {
        return this.item.name;
    }
  
    abstract getIcon(className?: string): React.ReactElement<any>;
    abstract buildDetails(): React.ReactElement<any>;
    
    buildActivity(): React.ReactElement<any> {
        return <div/>;
    }
}