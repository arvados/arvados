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


export const getProcessStatusStyles = (status: string, theme: ArvadosTheme): React.CSSProperties => {
    let color = theme.customs.colors.grey500;
    let running = false;
    switch (status) {
        case ProcessStatus.RUNNING:
            color = theme.customs.colors.green800;
            running = true;
            break;
        case ProcessStatus.COMPLETED:
            color = theme.customs.colors.green800;
            break;
        case ProcessStatus.WARNING:
            color = theme.customs.colors.green800;
            running = true;
            break;
        case ProcessStatus.FAILING:
            color = theme.customs.colors.red900;
            running = true;
            break;
        case ProcessStatus.CANCELLED:
        case ProcessStatus.FAILED:
            color = theme.customs.colors.red900;
            break;
        case ProcessStatus.QUEUED:
            color = theme.customs.colors.grey600;
            running = true;
            break;
        default:
            color = theme.customs.colors.grey600;
            break;
    }

    // Using color and running we build the text, border, and background style properties
    return {
        // Set background color when not running, otherwise use white
        backgroundColor: running ? theme.palette.common.white : color,
        // Set text color to status color when running, else use white text for solid button
        color: running ? color : theme.palette.common.white,
        // Set border color when running, else omit the style entirely
        ...(running ? {border: `2px solid ${color}`} : {}),
    };
};

export const getProcessStatus = ({ containerRequest, container }: Process): ProcessStatus => {
    switch (true) {
        case containerRequest.containerUuid && !container:
            return ProcessStatus.UNKNOWN;

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

export const isProcessRunnable = ({ containerRequest }: Process): boolean => (
    containerRequest.state === ContainerRequestState.UNCOMMITTED
);

export const isProcessResumable = ({ containerRequest, container }: Process): boolean => (
    containerRequest.state === ContainerRequestState.COMMITTED &&
    containerRequest.priority === 0 &&
    // Don't show run button when container is present & running or cancelled
    !(container && (container.state === ContainerState.RUNNING ||
                            container.state === ContainerState.CANCELLED ||
                            container.state === ContainerState.COMPLETE))
);

export const isProcessCancelable = ({ containerRequest, container }: Process): boolean => (
    containerRequest.priority !== null &&
    containerRequest.priority > 0 &&
    container !== undefined &&
        (container.state === ContainerState.QUEUED ||
        container.state === ContainerState.LOCKED ||
        container.state === ContainerState.RUNNING)
);

const isSubprocess = (containerUuid: string) => (resource: Resource) =>
    resource.kind === ResourceKind.CONTAINER_REQUEST
    && (resource as ContainerRequestResource).requestingContainerUuid === containerUuid;
