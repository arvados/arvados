// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { ProcessIcon } from '../../components/icon/icon';
import { ProcessResource } from '../../models/process';
import { formatDate } from '../../common/formatters';
import { ResourceKind } from '../../models/resource';
import { resourceLabel } from '../../common/labels';
import { DetailsData } from "./details-data";
import { DetailsAttribute } from "../../components/details-attribute/details-attribute";

export class ProcessDetails extends DetailsData<ProcessResource> {

    getIcon(className?: string){
        return <ProcessIcon className={className} />;
    }

    getDetails() {
        return <div>
            <DetailsAttribute label='Type' value={resourceLabel(ResourceKind.Process)} />
            <DetailsAttribute label='Size' value='---' />
            <DetailsAttribute label='Owner' value={this.item.ownerUuid} />

            {/* Missing attr */}
            <DetailsAttribute label='Status' value={this.item.state} />
            <DetailsAttribute label='Last modified' value={formatDate(this.item.modifiedAt)} />

            {/* Missing attrs */}
            <DetailsAttribute label='Started at' value={formatDate(this.item.createdAt)} />
            <DetailsAttribute label='Finished at' value={formatDate(this.item.expiresAt)} />

            {/* Links but we dont have view */}
            <DetailsAttribute label='Outputs' link={this.item.outputPath} value={this.item.outputPath} />
            <DetailsAttribute label='UUID' link={this.item.uuid} value={this.item.uuid} />
            <DetailsAttribute label='Container UUID' link={this.item.containerUuid} value={this.item.containerUuid} />

            <DetailsAttribute label='Priority' value={this.item.priority} />
            <DetailsAttribute label='Runtime Constraints' value={this.item.runtimeConstraints} />
            {/* Link but we dont have view */}
            <DetailsAttribute label='Docker Image locator' link={this.item.containerImage} value={this.item.containerImage} />
        </div>;
    }
}
