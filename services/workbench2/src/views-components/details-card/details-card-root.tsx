// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { RootState } from 'store/store';
import { connect } from 'react-redux';
import { Resource, ResourceKind } from 'models/resource';
import { UserCard } from './user-details-card';
import { ProjectCard } from './project-details-card';
import { getResource, ResourcesState } from 'store/resources/resources';

const mapStateToProps = ({ resources }: RootState) => {
    return { resources };
};

type DetailsCardProps = {
    currentItemId: string | undefined;
    resources: ResourcesState;
};

export const DetailsCardRoot = connect(mapStateToProps)(({  currentItemId, resources }: DetailsCardProps) => {
    const currentResource = currentItemId ? getResource<Resource>(currentItemId)(resources) : undefined;
    if (!currentResource) {
        return null;
    }
    switch (currentResource.kind as string) {
        case ResourceKind.USER:
            return <UserCard />;
        case ResourceKind.PROJECT:
            return <ProjectCard />;
        default:
            return null;
    }
});
