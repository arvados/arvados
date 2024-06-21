// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { RootState } from 'store/store';
import { connect } from 'react-redux';
import { getResource } from 'store/resources/resources';
import { ProjectResource } from 'models/project';
import { ResourceKind } from 'models/resource';
import { UserResource } from 'models/user';
import { UserCard } from './user-details-card';
import { ProjectCard } from './project-details-card';

const mapStateToProps = ({ resources, properties }: RootState) => {
    const currentResource = getResource(properties.currentRouteUuid)(resources);

    return {
        currentResource,
    };
};

type DetailsCardProps = {
    currentResource: ProjectResource | UserResource;
};

export const DetailsCardRoot = connect(mapStateToProps)(({ currentResource }: DetailsCardProps) => {
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
