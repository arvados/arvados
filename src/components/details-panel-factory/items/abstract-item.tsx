// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { DetailsPanelResource } from "./../details-panel-factory";
import { IconTypes } from '../../icon/icon';

export abstract class AbstractItem {
    protected item: DetailsPanelResource;

    constructor(item: DetailsPanelResource) {
        this.item = item;
    }

    getTitle(): string {
        return this.item.name;
    }

    abstract getIcon(): IconTypes;
    abstract buildDetails(): React.ReactElement<any>;
    
    buildActivity(): React.ReactElement<any> {
        return <div/>;
    }
}