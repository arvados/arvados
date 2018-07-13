// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { IconTypes } from '../../icon/icon';
import Attribute from '../../attribute/attribute';
import AbstractItem from './abstract-item';
import { ProjectResource } from '../../../models/project';
import { formatDate } from '../../../common/formatters';

export default class ProjectItem extends AbstractItem<ProjectResource> {

    constructor(item: ProjectResource) {
        super(item);
        console.log('item: ', this.item);
    }

    getIcon(): IconTypes {
        return IconTypes.FOLDER;
    }

    buildDetails(): React.ReactElement<any> {
        return <div>
            <Attribute label='Type' value={this.item.groupClass} />
            {/* Missing attr */}
            <Attribute label='Size' value='---' />
            <Attribute label='Owner' value={this.item.ownerUuid} />
            <Attribute label='Last modified' value={formatDate(this.item.modifiedAt)} />
            <Attribute label='Created at' value={formatDate(this.item.createdAt)} />
            {/* Missing attr */}
            <Attribute label='File size' value='1.4 GB' />
            <Attribute label='Description' value={this.item.description} />
        </div>;
    }
}