// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { ProcessIcon } from 'components/icon/icon';
import { ProcessResource } from 'models/process';
import { DetailsData } from "./details-data";
import { ProcessDetailsAttributes } from 'views/process-panel/process-details-attributes';

export class ProcessDetails extends DetailsData<ProcessResource> {

    getIcon(className?: string) {
        return <ProcessIcon className={className} />;
    }

    getDetails() {
        return <ProcessDetailsAttributes item={this.item} />;
    }
}
