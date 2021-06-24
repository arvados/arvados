// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { DetailsData } from "./details-data";
import { CollectionFile, CollectionDirectory, CollectionFileType } from 'models/collection-file';
import { getIcon } from 'components/file-tree/file-tree-item';
import { DetailsAttribute } from 'components/details-attribute/details-attribute';
import { formatFileSize } from 'common/formatters';
import { FileThumbnail } from 'components/file-tree/file-thumbnail';
import isImage from 'is-image';

export class FileDetails extends DetailsData<CollectionFile | CollectionDirectory> {

    getIcon(className?: string) {
        const Icon = getIcon(this.item.type);
        return <Icon className={className} />;
    }

    getDetails() {
        const { item } = this;
        return item.type === CollectionFileType.FILE
            ? <>
                <DetailsAttribute label='Size' value={formatFileSize(item.size)} />
                {
                    isImage(item.url) && <>
                        <DetailsAttribute label='Preview' />
                        <FileThumbnail file={item} />
                    </>
                }
            </>
            : <div />;
    }
}
