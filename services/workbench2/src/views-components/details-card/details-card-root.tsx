// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React, { useEffect } from 'react';
import { Dispatch } from 'redux';
import { RootState } from 'store/store';
import { connect } from 'react-redux';
import { ProjectResource } from 'models/project';
import { ResourceKind } from 'models/resource';
import { UserResource } from 'models/user';
import { CollectionResource } from 'models/collection';
import { WorkflowResource } from 'models/workflow';
import { ProcessResource } from 'models/process';
import { UserCard } from './user-details-card';
import { ProjectCard } from './project-details-card';
import { getResource } from 'store/resources/resources';
import { CollectionCard } from './collection-details-card';
import { WorkflowCard } from './workflow-details-card';
import { ProcessCard } from './process-details-card';
import { setSelectedResourceUuid } from 'store/selected-resource/selected-resource-actions';

const mapStateToProps = ({ resources, properties, selectedResource }: RootState) => {
    const currentResource = getResource(properties.currentRouteUuid)(resources);
    return {
        currentResource,
        selectedResourceUuid: selectedResource.selectedResourceUuid,
    };
};

const mapDispatchToProps = (dispatch: Dispatch) => ({
    setSelectedResourceUuid: (uuid: string) => dispatch<any>(setSelectedResourceUuid(uuid)),
});

type DetailsCardProps = {
    currentResource: ProjectResource | UserResource | CollectionResource | WorkflowResource | ProcessResource | undefined;
    selectedResourceUuid: string;
    setSelectedResourceUuid: (uuid: string) => void;
};

export const DetailsCardRoot = connect(mapStateToProps, mapDispatchToProps)(({ currentResource, selectedResourceUuid, setSelectedResourceUuid }: DetailsCardProps) => {
    if (!currentResource) {
        return null;
    }

    useEffect(() => {
        if (selectedResourceUuid !== currentResource.uuid) {
            setSelectedResourceUuid(currentResource.uuid);
        }
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [currentResource]);

    switch (currentResource.kind as string) {
        case ResourceKind.USER:
            return <UserCard />;
        case ResourceKind.PROJECT:
            return <ProjectCard />;
        case ResourceKind.COLLECTION:
            return <CollectionCard />;
        case ResourceKind.WORKFLOW:
            return <WorkflowCard />;
        case ResourceKind.PROCESS:
            return <ProcessCard />;
        default:
            return null;
    }
});
