// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContainerRequestResource } from '../../models/container-request';
import { ContainerResource } from '../../models/container';
import { ResourcesState, getResource } from '~/store/resources/resources';

export interface Process {
    containerRequest: ContainerRequestResource;
    container?: ContainerResource;
}

export const getProcess = (uuid: string) => (resources: ResourcesState): Process | undefined => {
    const containerRequest = getResource<ContainerRequestResource>(uuid)(resources);
    if (containerRequest) {
        if (containerRequest.containerUuid) {
            const container = getResource<ContainerResource>(containerRequest.containerUuid)(resources);
            if (container) {
                return { containerRequest, container };
            }
        }
        return { containerRequest };
    }
    return;
};
