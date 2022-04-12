// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ContainerRequestResource, ContainerRequestState } from '../../models/container-request';
import { ContainerResource, ContainerState } from '../../models/container';
import { ResourcesState, getResource } from 'store/resources/resources';
import { filterResources } from '../resources/resources';
import { ResourceKind, Resource, extractUuidKind } from 'models/resource';
import { getTimeDiff } from 'common/formatters';
import { ArvadosTheme } from 'common/custom-theme';

export interface Process {
    containerRequest: ContainerRequestResource;
    container?: ContainerResource;
}

export enum ProcessStatus {
    CANCELLED = 'Cancelled',
    COMPLETED = 'Completed',
    DRAFT = 'Draft',
    FAILING = 'Failing',
    FAILED = 'Failed',
    ONHOLD = 'On hold',
    QUEUED = 'Queued',
    RUNNING = 'Running',
    WARNING = 'Warning',
    UNKNOWN = 'Unknown',
}

export const getProcess = (uuid: string) => (resources: ResourcesState): Process | undefined => {
    if (extractUuidKind(uuid) === ResourceKind.CONTAINER_REQUEST) {
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

export const getProcessRuntime = ({ container }: Process) => {
    if (container) {
        if (container.startedAt === null) {
            return 0;
        }
        if (container.finishedAt === null) {
            // Count it from now
            return new Date().getTime() - new Date(container.startedAt).getTime();
        }
        return getTimeDiff(container.finishedAt, container.startedAt);
    } else {
        return 0;
    }
};

export const getProcessStatusColor = (status: string, { customs }: ArvadosTheme) => {
    switch (status) {
        case ProcessStatus.RUNNING:
            return customs.colors.blue500;
        case ProcessStatus.COMPLETED:
            return customs.colors.green700;
        case ProcessStatus.WARNING:
            return customs.colors.yellow700;
        case ProcessStatus.FAILING:
        case ProcessStatus.CANCELLED:
        case ProcessStatus.FAILED:
            return customs.colors.red900;
        default:
            return customs.colors.grey500;
    }
};

export const getProcessStatus = ({ containerRequest, container }: Process): ProcessStatus => {
    switch (true) {
        case containerRequest.state === ContainerRequestState.FINAL &&
            container?.state !== ContainerState.COMPLETE:
            // Request was finalized before its container started (or the
            // container was cancelled)
            return ProcessStatus.CANCELLED;

        case containerRequest.state === ContainerRequestState.UNCOMMITTED:
            return ProcessStatus.DRAFT;

        case container?.state === ContainerState.COMPLETE:
            if (container?.exitCode === 0) {
                return ProcessStatus.COMPLETED;
            }
            return ProcessStatus.FAILED;

        case container?.state === ContainerState.CANCELLED:
            return ProcessStatus.CANCELLED;

        case container?.state === ContainerState.QUEUED ||
            container?.state === ContainerState.LOCKED:
            if (containerRequest.priority === 0) {
                return ProcessStatus.ONHOLD;
            }
            return ProcessStatus.QUEUED;

        case container?.state === ContainerState.RUNNING:
            if (!!container?.runtimeStatus.error) {
                return ProcessStatus.FAILING;
            }
            if (!!container?.runtimeStatus.warning) {
                return ProcessStatus.WARNING;
            }
            return ProcessStatus.RUNNING;

        default:
            return ProcessStatus.UNKNOWN;
    }
};

const isSubprocess = (containerUuid: string) => (resource: Resource) =>
    resource.kind === ResourceKind.CONTAINER_REQUEST
    && (resource as ContainerRequestResource).requestingContainerUuid === containerUuid;
