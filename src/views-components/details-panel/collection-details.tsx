// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { CollectionIcon } from '~/components/icon/icon';
import { CollectionResource } from '~/models/collection';
import { formatDate, formatFileSize } from '~/common/formatters';
import { resourceLabel } from '~/common/labels';
import { ResourceKind } from '~/models/resource';
import { DetailsData } from "./details-data";
import { DetailsAttribute } from "~/components/details-attribute/details-attribute";

export class CollectionDetails extends DetailsData<CollectionResource> {

    getIcon(className?: string) {
        return <CollectionIcon className={className} />;
    }

    getDetails() {
        return <div>
            <DetailsAttribute label='Type' value={resourceLabel(ResourceKind.COLLECTION)} />
            <DetailsAttribute label='Size' value='---' />
            <DetailsAttribute label='Owner' value={this.item.ownerUuid} lowercaseValue={true} />
            <DetailsAttribute label='Last modified' value={formatDate(this.item.modifiedAt)} />
            <DetailsAttribute label='Created at' value={formatDate(this.item.createdAt)} />
            {/* Links but we dont have view */}
            <DetailsAttribute label='Collection UUID' link={this.item.uuid} value={this.item.uuid} />
            <DetailsAttribute label='Content address' link={this.item.portableDataHash} value={this.item.portableDataHash} />
            {/* Missing attrs */}
            <DetailsAttribute label='Number of files' value={this.data && this.data.fileCount} />
            <DetailsAttribute label='Content size' value={formatFileSize(this.data && this.data.fileSize)} />
        </div>;
    }
}
