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
import { memoize } from 'lodash';

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
    REUSED = 'Reused',
    CANCELLING = 'Cancelling',
    RESUBMITTED = 'Resubmitted',
}

export enum ProcessProperties {
    FAILED_CONTAINER_RESUBMITTED = "arv:failed_container_resubmitted",
}

/**
 * Gets a process from the store using container request uuid
 * @param uuid container request associated with process
 * @returns a Process object with containerRequest and optional container or undefined
 */

// both memoizes are needed to avoid x18 calls
export const getProcess = memoize((uuid: string) => memoize((resources: ResourcesState): Process | undefined => {
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
}));

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


export const getProcessStatusStyles = (status: string, theme: ArvadosTheme): React.CSSProperties & {'&:hover': {opacity: number, color: string}}  => {
    let primaryColor = theme.customs.colors.grey500;
    let secondaryColor = theme.palette.common.white;
    let altStyle = false;
    let lines = false;
    switch (status) {
        case ProcessStatus.RUNNING:
            primaryColor = theme.customs.colors.green800;
            altStyle = true;
            lines = true;
            break;
        case ProcessStatus.COMPLETED:
        case ProcessStatus.REUSED:
            primaryColor = theme.customs.colors.green800;
            break;
        case ProcessStatus.WARNING:
            primaryColor = theme.customs.colors.green800;
            altStyle = true;
            break;
        case ProcessStatus.RESUBMITTED:
            primaryColor = theme.customs.colors.darkOrange;
            break;
        case ProcessStatus.FAILING:
            primaryColor = theme.customs.colors.red900;
            altStyle = true;
            break;
        case ProcessStatus.CANCELLING:
            primaryColor = theme.customs.colors.red900;
            altStyle = true;
            break;
        case ProcessStatus.CANCELLED:
            primaryColor = theme.customs.colors.red900;
            altStyle = true;
            break;
        case ProcessStatus.FAILED:
            primaryColor = theme.customs.colors.red900;
            break;
        case ProcessStatus.QUEUED:
            primaryColor = theme.customs.colors.grey600;
            altStyle = true;
            lines = true;
            break;
        case ProcessStatus.ONHOLD:
            primaryColor = theme.customs.colors.grey600;
            altStyle = true;
            break;
        case ProcessStatus.DRAFT:
            primaryColor = theme.customs.colors.grey600;
            break;
        default:
            primaryColor = theme.customs.colors.black;
            break;
    }

    // Using color and altStyle we build the text, border, and background style properties
    return {
        // Set background color when not altStyle, otherwise use white
        backgroundColor: altStyle ? secondaryColor : primaryColor,
        // Set text color to status color when altStyle, else use white text for solid button
        color: altStyle ? primaryColor : secondaryColor,
        // Set background image to lines when lines, else omit the style entirely
        backgroundImage: lines ? `repeating-linear-gradient(
            310deg,
            #ccc 0px,
            #ccc 2px,
            transparent 2px,
            transparent 10px
          )` : undefined,
        '&:hover': {
                    opacity: 0.5,
                    color: primaryColor,
                },
        // Set border color when altStyle, else omit the style entirely
        border: `2px solid ${primaryColor}`,
    };
};

export const getProcessStatus = ({ containerRequest, container }: Process): ProcessStatus => {
    switch (true) {
        case containerRequest.containerUuid && !container:
            return ProcessStatus.UNKNOWN;

        case containerRequest.state === ContainerRequestState.UNCOMMITTED:
            return ProcessStatus.DRAFT;

        case containerRequest.state === ContainerRequestState.FINAL &&
                   containerRequest.properties &&
                   Boolean(containerRequest.properties[ProcessProperties.FAILED_CONTAINER_RESUBMITTED]):
                   // Failed but a new container request for the same work was resubmitted.
                   return ProcessStatus.RESUBMITTED;

        case containerRequest.state === ContainerRequestState.FINAL &&
                          container?.state === ContainerState.RUNNING:
                          // It is about to be completed but we haven't
                          // gotten the updated container record yet,
                          // if we don't catch this and show it as "Running"
                          // it will flicker "Cancelled" briefly
                          return ProcessStatus.RUNNING;

        case containerRequest.state === ContainerRequestState.FINAL &&
                                 container?.state !== ContainerState.COMPLETE:
                                 // Request was finalized before its container started (or the
                                 // container was cancelled)
            return ProcessStatus.CANCELLED;

        case container && container.state === ContainerState.COMPLETE:
            if (container?.exitCode === 0) {
                if (containerRequest && container.finishedAt) {
                    // don't compare on createdAt because the container can
                    // have a slightly earlier creation time when it is created
                    // in the same transaction as the container request.
                    // use finishedAt because most people will assume "reused" means
                    // no additional work needed to be done, it's possible
                    // to share a running container but calling it "reused" in that case
                    // is more likely to just be confusing.
                    const finishedAt = new Date(container.finishedAt).getTime();
                    const createdAt = new Date(containerRequest.createdAt).getTime();
                    if (finishedAt < createdAt) {
                        return ProcessStatus.REUSED;
                    }
                }
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
            if (container?.priority === 0) {
                return ProcessStatus.CANCELLING;
            }
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

export const isProcessRunning = ({ container }: Process): boolean => (
    container?.state === ContainerState.RUNNING
);

export const isProcessQueued = ({ container }: Process): boolean => (
    container?.state === ContainerState.QUEUED || container?.state === ContainerState.LOCKED
);

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

export const isProcessCancelable = memoize(({ containerRequest, container }: Process): boolean => (
    containerRequest.priority !== null &&
    containerRequest.priority > 0 &&
    container !== undefined &&
    (container.state === ContainerState.QUEUED ||
        container.state === ContainerState.LOCKED ||
        container.state === ContainerState.RUNNING)
));

const isSubprocess = (containerUuid: string) => (resource: Resource) =>
    resource.kind === ResourceKind.CONTAINER_REQUEST
    && (resource as ContainerRequestResource).requestingContainerUuid === containerUuid;
