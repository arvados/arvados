// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { Grid, Typography, withStyles, Tooltip, IconButton } from '@material-ui/core';
import { FavoriteStar } from '../favorite-star/favorite-star';
import { ResourceKind, TrashableResource } from '~/models/resource';
import { ProjectIcon, CollectionIcon, ProcessIcon, DefaultIcon, WorkflowIcon, ShareIcon } from '~/components/icon/icon';
import { formatDate, formatFileSize } from '~/common/formatters';
import { resourceLabel } from '~/common/labels';
import { connect } from 'react-redux';
import { RootState } from '~/store/store';
import { getResource } from '~/store/resources/resources';
import { GroupContentsResource } from '~/services/groups-service/groups-service';
import { getProcess, Process, getProcessStatus, getProcessStatusColor } from '~/store/processes/process';
import { ArvadosTheme } from '~/common/custom-theme';
import { compose } from 'redux';
import { WorkflowResource } from '~/models/workflow';
import { ResourceStatus } from '~/views/workflow-panel/workflow-panel-view';
import { getUuidPrefix } from '~/store/workflow-panel/workflow-panel-actions';
import { CollectionResource } from "~/models/collection";
import { getResourceData } from "~/store/resources-data/resources-data";

export const renderName = (item: { name: string; uuid: string, kind: string }) =>
    <Grid container alignItems="center" wrap="nowrap" spacing={16}>
        <Grid item>
            {renderIcon(item)}
        </Grid>
        <Grid item>
            <Typography color="primary" style={{ width: '450px' }}>
                {item.name}
            </Typography>
        </Grid>
        <Grid item>
            <Typography variant="caption">
                <FavoriteStar resourceUuid={item.uuid} />
            </Typography>
        </Grid>
    </Grid>;

export const ResourceName = connect(
    (state: RootState, props: { uuid: string }) => {
        const resource = getResource<GroupContentsResource>(props.uuid)(state.resources);
        return resource || { name: '', uuid: '', kind: '' };
    })(renderName);

export const renderIcon = (item: { kind: string }) => {
    switch (item.kind) {
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

export const renderDate = (date?: string) => {
    return <Typography noWrap style={{ minWidth: '100px' }}>{formatDate(date)}</Typography>;
};

export const renderWorkflowName = (item: { name: string; uuid: string, kind: string, ownerUuid: string }) =>
    <Grid container alignItems="center" wrap="nowrap" spacing={16}>
        <Grid item>
            {renderIcon(item)}
        </Grid>
        <Grid item>
            <Typography color="primary" style={{ width: '100px' }}>
                {item.name}
            </Typography>
        </Grid>
    </Grid>;

export const RosurceWorkflowName = connect(
    (state: RootState, props: { uuid: string }) => {
        const resource = getResource<WorkflowResource>(props.uuid)(state.resources);
        return resource || { name: '', uuid: '', kind: '', ownerUuid: '' };
    })(renderWorkflowName);

const getPublicUuid = (uuidPrefix: string) => {
    return `${uuidPrefix}-tpzed-anonymouspublic`;
};

// do share onClick
export const resourceShare = (uuidPrefix: string, ownerUuid?: string) => {
    return <Tooltip title="Share">
        <IconButton onClick={() => undefined}>
            {ownerUuid === getPublicUuid(uuidPrefix) ? <ShareIcon /> : null}
        </IconButton>
    </Tooltip>;
};

export const ResourceShare = connect(
    (state: RootState, props: { uuid: string }) => {
        const resource = getResource<WorkflowResource>(props.uuid)(state.resources);
        const uuidPrefix = getUuidPrefix(state);
        return {
            ownerUuid: resource ? resource.ownerUuid : '',
            uuidPrefix
        };
    })((props: { ownerUuid?: string, uuidPrefix: string }) => resourceShare(props.uuidPrefix, props.ownerUuid));

export const renderWorkflowStatus = (uuidPrefix: string, ownerUuid?: string) => {
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
        const resource = getResourceData(props.uuid)(state.resourcesData);
        return { fileSize: resource ? resource.fileSize : 0 };
    })((props: { fileSize?: number }) => renderFileSize(props.fileSize));

export const renderOwner = (owner: string) =>
    <Typography noWrap color="primary" >
        {owner}
    </Typography>;

export const ResourceOwner = connect(
    (state: RootState, props: { uuid: string }) => {
        const resource = getResource<GroupContentsResource>(props.uuid)(state.resources);
        return { owner: resource ? resource.ownerUuid : '' };
    })((props: { owner: string }) => renderOwner(props.owner));

export const renderType = (type: string) =>
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
            align="center"
            style={{ color: getProcessStatusColor(status, props.theme) }} >
            {status}
        </Typography>;
    });
