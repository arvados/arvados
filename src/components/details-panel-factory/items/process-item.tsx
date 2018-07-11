// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import IconBase, { IconTypes } from '../../icon/icon';
import Attribute from '../../attribute/attribute';
import AbstractItem from './abstract-item';
import { ProcessResource } from '../../../models/process';
import { formatDate } from '../../../common/formatters';
import { ResourceKind } from '../../../models/resource';

export default class ProcessItem extends AbstractItem<ProcessResource> {

    getIcon(): IconTypes {
        return IconTypes.BUBBLE_CHART;
    }

    buildDetails(): React.ReactElement<any> {
        return <div>
            <Attribute label='Type' value={ResourceKind.Process} />
            <Attribute label='Size' value='---' />
            <Attribute label='Owner' value={this.item.ownerUuid} />

            {/* Missing attr */}
            <Attribute label='Status' value={this.item.state} />
            <Attribute label='Last modified' value={formatDate(this.item.modifiedAt)} />
            
            {/* Missing attrs */}
            <Attribute label='Started at' value={formatDate(this.item.createdAt)} />
            <Attribute label='Finished at' value={formatDate(this.item.expiresAt)} />

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