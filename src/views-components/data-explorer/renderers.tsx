// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { Grid, Typography, withStyles, Tooltip, IconButton, Checkbox } from '@material-ui/core';
import { FavoriteStar, PublicFavoriteStar } from '../favorite-star/favorite-star';
import { ResourceKind, TrashableResource } from '~/models/resource';
import { ProjectIcon, CollectionIcon, ProcessIcon, DefaultIcon, WorkflowIcon, ShareIcon } from '~/components/icon/icon';
import { formatDate, formatFileSize } from '~/common/formatters';
import { resourceLabel } from '~/common/labels';
import { connect, DispatchProp } from 'react-redux';
import { RootState } from '~/store/store';
import { getResource } from '~/store/resources/resources';
import { GroupContentsResource } from '~/services/groups-service/groups-service';
import { getProcess, Process, getProcessStatus, getProcessStatusColor } from '~/store/processes/process';
import { ArvadosTheme } from '~/common/custom-theme';
import { compose, Dispatch } from 'redux';
import { WorkflowResource } from '~/models/workflow';
import { ResourceStatus } from '~/views/workflow-panel/workflow-panel-view';
import { getUuidPrefix, openRunProcess } from '~/store/workflow-panel/workflow-panel-actions';
import { openSharingDialog } from '~/store/sharing-dialog/sharing-dialog-actions';
import { UserResource } from '~/models/user';
import { toggleIsActive, toggleIsAdmin } from '~/store/users/users-actions';
import { LinkResource } from '~/models/link';
import { navigateTo } from '~/store/navigation/navigation-action';
import { withResourceData } from '~/views-components/data-explorer/with-resources';
import { CollectionResource } from '~/models/collection';
import { IllegalNamingWarning } from '~/components/warning/warning';

const renderName = (dispatch: Dispatch, item: { name: string; uuid: string, kind: string }) =>
    <Grid container alignItems="center" wrap="nowrap" spacing={16}>
        <Grid item>
            {renderIcon(item.kind)}
        </Grid>
        <Grid item>
            <Typography color="primary" style={{ width: 'auto', cursor: 'pointer' }} onClick={() => dispatch<any>(navigateTo(item.uuid))}>
                { item.kind === ResourceKind.PROJECT || item.kind === ResourceKind.COLLECTION
                    ? <IllegalNamingWarning name={item.name} />
                    : null }
                {item.name}
            </Typography>
        </Grid>
        <Grid item>
            <Typography variant="caption">
                <FavoriteStar resourceUuid={item.uuid} />
                <PublicFavoriteStar resourceUuid={item.uuid} />
            </Typography>
        </Grid>
    </Grid>;

export const ResourceName = connect(
    (state: RootState, props: { uuid: string }) => {
        const resource = getResource<GroupContentsResource>(props.uuid)(state.resources);
        return resource || { name: '', uuid: '', kind: '' };
    })((resource: { name: string; uuid: string, kind: string } & DispatchProp<any>) => renderName(resource.dispatch, resource));

const renderIcon = (kind: string) => {
    switch (kind) {
        case ResourceKind.PROJECT:
            return <ProjectIcon />;
        case ResourceKind.COLLECTION:
            return <CollectionIcon />;
        case ResourceKind.PROCESS:
            return <ProcessIcon />;
        case ResourceKind.WORKFLOW:
            return <WorkflowIcon />;
        default:
            return <DefaultIcon />;
    }
};

const renderDate = (date?: string) => {
    return <Typography noWrap style={{ minWidth: '100px' }}>{formatDate(date)}</Typography>;
};

const renderWorkflowName = (item: { name: string; uuid: string, kind: string, ownerUuid: string }) =>
    <Grid container alignItems="center" wrap="nowrap" spacing={16}>
        <Grid item>
            {renderIcon(item.kind)}
        </Grid>
        <Grid item>
            <Typography color="primary" style={{ width: '100px' }}>
                {item.name}
            </Typography>
        </Grid>
    </Grid>;

export const ResourceWorkflowName = connect(
    (state: RootState, props: { uuid: string }) => {
        const resource = getResource<WorkflowResource>(props.uuid)(state.resources);
        return resource || { name: '', uuid: '', kind: '', ownerUuid: '' };
    })(renderWorkflowName);

const getPublicUuid = (uuidPrefix: string) => {
    return `${uuidPrefix}-tpzed-anonymouspublic`;
};

const resourceShare = (dispatch: Dispatch, uuidPrefix: string, ownerUuid?: string, uuid?: string) => {
    const isPublic = ownerUuid === getPublicUuid(uuidPrefix);
    return (
        <div>
            {!isPublic && uuid &&
                <Tooltip title="Share">
                    <IconButton onClick={() => dispatch<any>(openSharingDialog(uuid))}>
                        <ShareIcon />
                    </IconButton>
                </Tooltip>
            }
        </div>
    );
};

export const ResourceShare = connect(
    (state: RootState, props: { uuid: string }) => {
        const resource = getResource<WorkflowResource>(props.uuid)(state.resources);
        const uuidPrefix = getUuidPrefix(state);
        return {
            uuid: resource ? resource.uuid : '',
            ownerUuid: resource ? resource.ownerUuid : '',
            uuidPrefix
        };
    })((props: { ownerUuid?: string, uuidPrefix: string, uuid?: string } & DispatchProp<any>) =>
        resourceShare(props.dispatch, props.uuidPrefix, props.ownerUuid, props.uuid));

const renderFirstName = (item: { firstName: string }) => {
    return <Typography noWrap>{item.firstName}</Typography>;
};

// User Resources
export const ResourceFirstName = connect(
    (state: RootState, props: { uuid: string }) => {
        const resource = getResource<UserResource>(props.uuid)(state.resources);
        return resource || { firstName: '' };
    })(renderFirstName);

const renderLastName = (item: { lastName: string }) =>
    <Typography noWrap>{item.lastName}</Typography>;

export const ResourceLastName = connect(
    (state: RootState, props: { uuid: string }) => {
        const resource = getResource<UserResource>(props.uuid)(state.resources);
        return resource || { lastName: '' };
    })(renderLastName);

const renderUuid = (item: { uuid: string }) =>
    <Typography noWrap>{item.uuid}</Typography>;

export const ResourceUuid = connect(
    (state: RootState, props: { uuid: string }) => {
        const resource = getResource<UserResource>(props.uuid)(state.resources);
        return resource || { uuid: '' };
    })(renderUuid);

const renderEmail = (item: { email: string }) =>
    <Typography noWrap>{item.email}</Typography>;

export const ResourceEmail = connect(
    (state: RootState, props: { uuid: string }) => {
        const resource = getResource<UserResource>(props.uuid)(state.resources);
        return resource || { email: '' };
    })(renderEmail);

const renderIsActive = (props: { uuid: string, isActive: boolean, toggleIsActive: (uuid: string) => void }) =>
    <Checkbox
        color="primary"
        checked={props.isActive}
        onClick={() => props.toggleIsActive(props.uuid)} />;

export const ResourceIsActive = connect(
    (state: RootState, props: { uuid: string }) => {
        const resource = getResource<UserResource>(props.uuid)(state.resources);
        return resource || { isActive: false };
    }, { toggleIsActive }
)(renderIsActive);

const renderIsAdmin = (props: { uuid: string, isAdmin: boolean, toggleIsAdmin: (uuid: string) => void }) =>
    <Checkbox
        color="primary"
        checked={props.isAdmin}
        onClick={() => props.toggleIsAdmin(props.uuid)} />;

export const ResourceIsAdmin = connect(
    (state: RootState, props: { uuid: string }) => {
        const resource = getResource<UserResource>(props.uuid)(state.resources);
        return resource || { isAdmin: false };
    }, { toggleIsAdmin }
)(renderIsAdmin);

const renderUsername = (item: { username: string }) =>
    <Typography noWrap>{item.username}</Typography>;

export const ResourceUsername = connect(
    (state: RootState, props: { uuid: string }) => {
        const resource = getResource<UserResource>(props.uuid)(state.resources);
        return resource || { username: '' };
    })(renderUsername);

// Common methods
const renderCommonData = (data: string) =>
    <Typography noWrap>{data}</Typography>;

const renderCommonDate = (date: string) =>
    <Typography noWrap>{formatDate(date)}</Typography>;

export const CommonUuid = withResourceData('uuid', renderCommonData);

// Api Client Authorizations
export const TokenApiClientId = withResourceData('apiClientId', renderCommonData);

export const TokenApiToken = withResourceData('apiToken', renderCommonData);

export const TokenCreatedByIpAddress = withResourceData('createdByIpAddress', renderCommonDate);

export const TokenDefaultOwnerUuid = withResourceData('defaultOwnerUuid', renderCommonData);

export const TokenExpiresAt = withResourceData('expiresAt', renderCommonDate);

export const TokenLastUsedAt = withResourceData('lastUsedAt', renderCommonDate);

export const TokenLastUsedByIpAddress = withResourceData('lastUsedByIpAddress', renderCommonData);

export const TokenScopes = withResourceData('scopes', renderCommonData);

export const TokenUserId = withResourceData('userId', renderCommonData);

// Compute Node Resources
const renderNodeInfo = (data: string) => {
    return <Typography>{JSON.stringify(data, null, 4)}</Typography>;
};

const clusterColors = [
    ['#f44336', '#fff'],
    ['#2196f3', '#fff'],
    ['#009688', '#fff'],
    ['#cddc39', '#fff'],
    ['#ff9800', '#fff']
];

export const ResourceCluster = (props: { uuid: string }) => {
    const CLUSTER_ID_LENGTH = 5;
    const pos = props.uuid.length > CLUSTER_ID_LENGTH ? props.uuid.indexOf('-') : 5;
    const clusterId = pos >= CLUSTER_ID_LENGTH ? props.uuid.substr(0, pos) : '';
    const ci = pos >= CLUSTER_ID_LENGTH ? (((((
        (props.uuid.charCodeAt(0) * props.uuid.charCodeAt(1))
        + props.uuid.charCodeAt(2))
        * props.uuid.charCodeAt(3))
        + props.uuid.charCodeAt(4))) % clusterColors.length) : 0;
    return <span style={{
        backgroundColor: clusterColors[ci][0],
        color: clusterColors[ci][1],
        padding: "2px 7px",
        borderRadius: 3
    }}>{clusterId}</span>;
};

export const ComputeNodeInfo = withResourceData('info', renderNodeInfo);

export const ComputeNodeDomain = withResourceData('domain', renderCommonData);

export const ComputeNodeFirstPingAt = withResourceData('firstPingAt', renderCommonDate);

export const ComputeNodeHostname = withResourceData('hostname', renderCommonData);

export const ComputeNodeIpAddress = withResourceData('ipAddress', renderCommonData);

export const ComputeNodeJobUuid = withResourceData('jobUuid', renderCommonData);

export const ComputeNodeLastPingAt = withResourceData('lastPingAt', renderCommonDate);

// Links Resources
const renderLinkName = (item: { name: string }) =>
    <Typography noWrap>{item.name || '(none)'}</Typography>;

export const ResourceLinkName = connect(
    (state: RootState, props: { uuid: string }) => {
        const resource = getResource<LinkResource>(props.uuid)(state.resources);
        return resource || { name: '' };
    })(renderLinkName);

const renderLinkClass = (item: { linkClass: string }) =>
    <Typography noWrap>{item.linkClass}</Typography>;

export const ResourceLinkClass = connect(
    (state: RootState, props: { uuid: string }) => {
        const resource = getResource<LinkResource>(props.uuid)(state.resources);
        return resource || { linkClass: '' };
    })(renderLinkClass);

const renderLinkTail = (dispatch: Dispatch, item: { uuid: string, tailUuid: string, tailKind: string }) => {
    const currentLabel = resourceLabel(item.tailKind);
    const isUnknow = currentLabel === "Unknown";
    return (<div>
        {!isUnknow ? (
            renderLink(dispatch, item.tailUuid, currentLabel)
        ) : (
                <Typography noWrap color="default">
                    {item.tailUuid}
                </Typography>
            )}
    </div>);
};

const renderLink = (dispatch: Dispatch, uuid: string, label: string) =>
    <Typography noWrap color="primary" style={{ 'cursor': 'pointer' }} onClick={() => dispatch<any>(navigateTo(uuid))}>
        {label}: {uuid}
    </Typography>;

export const ResourceLinkTail = connect(
    (state: RootState, props: { uuid: string }) => {
        const resource = getResource<LinkResource>(props.uuid)(state.resources);
        return {
            item: resource || { uuid: '', tailUuid: '', tailKind: ResourceKind.NONE }
        };
    })((props: { item: any } & DispatchProp<any>) =>
        renderLinkTail(props.dispatch, props.item));

const renderLinkHead = (dispatch: Dispatch, item: { uuid: string, headUuid: string, headKind: ResourceKind }) =>
    renderLink(dispatch, item.headUuid, resourceLabel(item.headKind));

export const ResourceLinkHead = connect(
    (state: RootState, props: { uuid: string }) => {
        const resource = getResource<LinkResource>(props.uuid)(state.resources);
        return {
            item: resource || { uuid: '', headUuid: '', headKind: ResourceKind.NONE }
        };
    })((props: { item: any } & DispatchProp<any>) =>
        renderLinkHead(props.dispatch, props.item));

export const ResourceLinkUuid = connect(
    (state: RootState, props: { uuid: string }) => {
        const resource = getResource<LinkResource>(props.uuid)(state.resources);
        return resource || { uuid: '' };
    })(renderUuid);

// Process Resources
const resourceRunProcess = (dispatch: Dispatch, uuid: string) => {
    return (
        <div>
            {uuid &&
                <Tooltip title="Run process">
                    <IconButton onClick={() => dispatch<any>(openRunProcess(uuid))}>
                        <ProcessIcon />
                    </IconButton>
                </Tooltip>}
        </div>
    );
};

export const ResourceRunProcess = connect(
    (state: RootState, props: { uuid: string }) => {
        const resource = getResource<WorkflowResource>(props.uuid)(state.resources);
        return {
            uuid: resource ? resource.uuid : ''
        };
    })((props: { uuid: string } & DispatchProp<any>) =>
        resourceRunProcess(props.dispatch, props.uuid));

const renderWorkflowStatus = (uuidPrefix: string, ownerUuid?: string) => {
    if (ownerUuid === getPublicUuid(uuidPrefix)) {
        return renderStatus(ResourceStatus.PUBLIC);
    } else {
        return renderStatus(ResourceStatus.PRIVATE);
    }
};

const renderStatus = (status: string) =>
    <Typography noWrap style={{ width: '60px' }}>{status}</Typography>;

export const ResourceWorkflowStatus = connect(
    (state: RootState, props: { uuid: string }) => {
        const resource = getResource<WorkflowResource>(props.uuid)(state.resources);
        const uuidPrefix = getUuidPrefix(state);
        return {
            ownerUuid: resource ? resource.ownerUuid : '',
            uuidPrefix
        };
    })((props: { ownerUuid?: string, uuidPrefix: string }) => renderWorkflowStatus(props.uuidPrefix, props.ownerUuid));

export const ResourceLastModifiedDate = connect(
    (state: RootState, props: { uuid: string }) => {
        const resource = getResource<GroupContentsResource>(props.uuid)(state.resources);
        return { date: resource ? resource.modifiedAt : '' };
    })((props: { date: string }) => renderDate(props.date));

export const ResourceTrashDate = connect(
    (state: RootState, props: { uuid: string }) => {
        const resource = getResource<TrashableResource>(props.uuid)(state.resources);
        return { date: resource ? resource.trashAt : '' };
    })((props: { date: string }) => renderDate(props.date));

export const ResourceDeleteDate = connect(
    (state: RootState, props: { uuid: string }) => {
        const resource = getResource<TrashableResource>(props.uuid)(state.resources);
        return { date: resource ? resource.deleteAt : '' };
    })((props: { date: string }) => renderDate(props.date));

export const renderFileSize = (fileSize?: number) =>
    <Typography noWrap style={{ minWidth: '45px' }}>
        {formatFileSize(fileSize)}
    </Typography>;

export const ResourceFileSize = connect(
    (state: RootState, props: { uuid: string }) => {
        const resource = getResource<CollectionResource>(props.uuid)(state.resources);
        return { fileSize: resource ? resource.fileSizeTotal : 0 };
    })((props: { fileSize?: number }) => renderFileSize(props.fileSize));

const renderOwner = (owner: string) =>
    <Typography noWrap>
        {owner}
    </Typography>;

export const ResourceOwner = connect(
    (state: RootState, props: { uuid: string }) => {
        const resource = getResource<GroupContentsResource>(props.uuid)(state.resources);
        return { owner: resource ? resource.ownerUuid : '' };
    })((props: { owner: string }) => renderOwner(props.owner));

export const ResourceOwnerName = connect(
    (state: RootState, props: { uuid: string }) => {
        const resource = getResource<GroupContentsResource>(props.uuid)(state.resources);
        const ownerNameState = state.ownerName;
        const ownerName = ownerNameState.find(it => it.uuid === resource!.ownerUuid);
        return { owner: ownerName ? ownerName!.name : resource!.ownerUuid };
    })((props: { owner: string }) => renderOwner(props.owner));

const renderType = (type: string) =>
    <Typography noWrap>
        {resourceLabel(type)}
    </Typography>;

export const ResourceType = connect(
    (state: RootState, props: { uuid: string }) => {
        const resource = getResource<GroupContentsResource>(props.uuid)(state.resources);
        return { type: resource ? resource.kind : '' };
    })((props: { type: string }) => renderType(props.type));

export const ProcessStatus = compose(
    connect((state: RootState, props: { uuid: string }) => {
        return { process: getProcess(props.uuid)(state.resources) };
    }),
    withStyles({}, { withTheme: true }))
    ((props: { process?: Process, theme: ArvadosTheme }) => {
        const status = props.process ? getProcessStatus(props.process) : "-";
        return <Typography
            noWrap
            style={{ color: getProcessStatusColor(status, props.theme) }} >
            {status}
        </Typography>;
    });
