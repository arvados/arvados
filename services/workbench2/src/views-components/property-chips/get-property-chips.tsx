// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';

import { getPropertyChip } from 'views-components/resource-properties-form/property-chip';
import { ContainerRequestResource } from 'models/container-request';
import { CollectionResource } from 'models/collection';
import { ProjectResource } from 'models/project';

export const getPropertyChips = (resource: ProjectResource | CollectionResource | ContainerRequestResource | undefined, classes: any): JSX.Element | null => {
    if (!resource || !resource.properties || typeof resource.properties !== 'object') return null;

    const properties = { ...resource.properties } as Record<string, any>;

    return (
        <section data-cy='resource-properties'>
            {Object.keys(properties).map((k) =>
                Array.isArray(properties[k])
                    ? properties[k].map((v: string) => getPropertyChip(k, v, undefined, classes.tag))
                    : typeof properties[k] === 'object'
                        ? null
                        : getPropertyChip(k, properties[k], undefined, classes.tag)
            )}
        </section>
    );
};
