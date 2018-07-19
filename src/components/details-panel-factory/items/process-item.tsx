// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { ProcessIcon } from '../../icon/icon';
import Attribute from '../../attribute/attribute';
import AbstractItem from './abstract-item';
import { ProcessResource } from '../../../models/process';
import { FORMAT_DATE } from '../../../common/formatters';
import { ResourceKind } from '../../../models/resource';
import { RESOURCE_LABEL } from '../../../common/labels';

export default class ProcessItem extends AbstractItem<ProcessResource> {

    getIcon(className?: string){
        return <ProcessIcon className={className} />;
    }

    buildDetails() {
        return <div>
            <Attribute label='Type' value={RESOURCE_LABEL(ResourceKind.Process)} />
            <Attribute label='Size' value='---' />
            <Attribute label='Owner' value={this.item.ownerUuid} />

            {/* Missing attr */}
            <Attribute label='Status' value={this.item.state} />
            <Attribute label='Last modified' value={FORMAT_DATE(this.item.modifiedAt)} />
            
            {/* Missing attrs */}
            <Attribute label='Started at' value={FORMAT_DATE(this.item.createdAt)} />
            <Attribute label='Finished at' value={FORMAT_DATE(this.item.expiresAt)} />

            {/* Links but we dont have view */}
            <Attribute label='Outputs' link={this.item.outputPath} value={this.item.outputPath} />
            <Attribute label='UUID' link={this.item.uuid} value={this.item.uuid} />
            <Attribute label='Container UUID' link={this.item.containerUuid} value={this.item.containerUuid} />
            
            <Attribute label='Priority' value={this.item.priority} />
            <Attribute label='Runtime Constraints' value={this.item.runtimeConstraints} />
            {/* Link but we dont have view */}
            <Attribute label='Docker Image locator' link={this.item.containerImage} value={this.item.containerImage} />
        </div>;
    }
}