// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { ProjectIcon } from '../../icon/icon';
import { Attribute } from '../../attribute/attribute';
import { AbstractItem } from './abstract-item';
import { ProjectResource } from '../../../models/project';
import { formatDate } from '../../../common/formatters';
import { ResourceKind } from '../../../models/resource';
import { resourceLabel } from '../../../common/labels';

export class ProjectItem extends AbstractItem<ProjectResource> {

    getIcon(className?: string) {
        return <ProjectIcon className={className} />;
    }

    buildDetails() {
        return <div>
            <Attribute label='Type' value={resourceLabel(ResourceKind.Project)} />
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
