// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { ProjectIcon } from '~/components/icon/icon';
import { ProjectResource } from '~/models/project';
import { formatDate } from '~/common/formatters';
import { ResourceKind } from '~/models/resource';
import { resourceLabel } from '~/common/labels';
import { DetailsData } from "./details-data";
import { DetailsAttribute } from "~/components/details-attribute/details-attribute";
import { RichTextEditorLink } from '~/components/rich-text-editor-link/rich-text-editor-link';

export class ProjectDetails extends DetailsData<ProjectResource> {

    getIcon(className?: string) {
        return <ProjectIcon className={className} />;
    }

    getDetails() {
        return <div>
            <DetailsAttribute label='Type' value={resourceLabel(ResourceKind.PROJECT)} />
            {/* Missing attr */}
            <DetailsAttribute label='Size' value='---' />
            <DetailsAttribute label='Owner' value={this.item.ownerUuid} lowercaseValue={true} />
            <DetailsAttribute label='Last modified' value={formatDate(this.item.modifiedAt)} />
            <DetailsAttribute label='Created at' value={formatDate(this.item.createdAt)} />
            {/* Missing attr */}
            {/*<DetailsAttribute label='File size' value='1.4 GB' />*/}
            <DetailsAttribute label='Description'>
                {this.item.description ?
                    <RichTextEditorLink
                        title={`Description of ${this.item.name}`}
                        content={this.item.description}
                        label='Show full description' />
                    : '---'
                }
            </DetailsAttribute>
        </div>;
    }
}
