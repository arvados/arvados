// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { CollectionIcon } from '~/components/icon/icon';
import { CollectionResource } from '~/models/collection';
import { DetailsData } from "./details-data";
import { CollectionDetailsAttributes } from '~/views/collection-panel/collection-panel';

export class CollectionDetails extends DetailsData<CollectionResource> {

    getIcon(className?: string) {
        return <CollectionIcon className={className} />;
    }

    getDetails() {
        return <CollectionDetailsAttributes item={this.item} />;
    }
}
