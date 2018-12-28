// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { DetailsData } from "./details-data";
import { CollectionFile, CollectionDirectory } from '~/models/collection-file';
import { getIcon } from '~/components/file-tree/file-tree-item';

export class FileDetails extends DetailsData<CollectionFile | CollectionDirectory> {

    getIcon(className?: string) {
        const Icon = getIcon(this.item.type);
        return <Icon className={className} />;
    }

    getDetails() {
        return <div>File details</div>;
    }
}
