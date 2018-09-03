// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContainerRequestResource } from '../../models/container-request';
import { ContainerResource } from '../../models/container';
import { ResourcesState, getResource } from '~/store/resources/resources';
import { filterResources } from '../resources/resources';
import { ResourceKind, Resource } from '~/models/resource';
import { getTimeDiff } from '~/common/formatters';
import { SubprocessesStatus } from '~/views/process-panel/process-subprocesses-card';
import { ArvadosTheme } from '~/common/custom-theme';

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

export const getSubprocesses = (uuid: string) => (resources: ResourcesState) => {
    const process = getProcess(uuid)(resources);
    if (process && process.container) {
        const containerRequests = filterResources(isSubprocess(process.container.uuid))(resources) as ContainerRequestResource[];
        return containerRequests.reduce((subprocesses, { uuid }) => {
            const process = getProcess(uuid)(resources);
            return process
                ? [...subprocesses, process]
                : subprocesses;
        }, []);
    }
    return [];
};

export const getProcessRuntime = ({ container }: Process) =>
    container
        ? getTimeDiff(container.finishedAt || '', container.startedAt || '')
        : 0;

export const getProcessStatusColor = (status: string, { customs }: ArvadosTheme) => {
    switch (status) {
        case SubprocessesStatus.COMPLETED:
            return customs.colors.green700;
        case SubprocessesStatus.CANCELED:
            return customs.colors.red900;
        case SubprocessesStatus.QUEUED:
            return customs.colors.grey500;
        case SubprocessesStatus.FAILED:
            return customs.colors.red900;
        case SubprocessesStatus.ACTIVE:
            return customs.colors.blue500;
        default:
            return customs.colors.grey500;
    }
};

export const getProcessStatus = (process: Process) =>
    process.container
        ? process.container.state
        : process.containerRequest.state;

const isSubprocess = (containerUuid: string) => (resource: Resource) =>
    resource.kind === ResourceKind.CONTAINER_REQUEST
    && (resource as ContainerRequestResource).requestingContainerUuid === containerUuid;

