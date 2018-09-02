// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { Grid, Typography } from '@material-ui/core';
import { FavoriteStar } from '../favorite-star/favorite-star';
import { ResourceKind, TrashResource } from '~/models/resource';
import { ProjectIcon, CollectionIcon, ProcessIcon, DefaultIcon } from '~/components/icon/icon';
import { formatDate, formatFileSize } from '~/common/formatters';
import { resourceLabel } from '~/common/labels';
import { connect } from 'react-redux';
import { RootState } from '~/store/store';
import { getResource } from '~/store/resources/resources';
import { GroupContentsResource } from '~/services/groups-service/groups-service';
import { ProcessResource } from '~/models/process';


export const renderName = (item: { name: string; uuid: string, kind: string }) =>
    <Grid container alignItems="center" wrap="nowrap" spacing={16}>
        <Grid item>
            {renderIcon(item)}
        </Grid>
        <Grid item>
            <Typography color="primary">
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
        const resource = getResource(props.uuid)(state.resources) as GroupContentsResource | undefined;
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
        default:
            return <DefaultIcon />;
    }
};

export const renderDate = (date?: string) => {
    return <Typography noWrap>{formatDate(date)}</Typography>;
};

export const ResourceLastModifiedDate = connect(
    (state: RootState, props: { uuid: string }) => {
        const resource = getResource(props.uuid)(state.resources) as GroupContentsResource | undefined;
        return { date: resource ? resource.modifiedAt : '' };
    })((props: { date: string }) => renderDate(props.date));

export const ResourceTrashDate = connect(
    (state: RootState, props: { uuid: string }) => {
        const resource = getResource(props.uuid)(state.resources) as TrashResource | undefined;
        return { date: resource ? resource.trashAt : '' };
    })((props: { date: string }) => renderDate(props.date));

export const ResourceDeleteDate = connect(
    (state: RootState, props: { uuid: string }) => {
        const resource = getResource(props.uuid)(state.resources) as TrashResource | undefined;
        return { date: resource ? resource.deleteAt : '' };
    })((props: { date: string }) => renderDate(props.date));

export const renderFileSize = (fileSize?: number) =>
    <Typography noWrap>
        {formatFileSize(fileSize)}
    </Typography>;

export const ResourceFileSize = connect(
    (state: RootState, props: { uuid: string }) => {
        const resource = getResource(props.uuid)(state.resources) as GroupContentsResource | undefined;
        return {};
    })((props: { fileSize?: number }) => renderFileSize(props.fileSize));

export const renderOwner = (owner: string) =>
    <Typography noWrap color="primary" >
        {owner}
    </Typography>;

export const ResourceOwner = connect(
    (state: RootState, props: { uuid: string }) => {
        const resource = getResource(props.uuid)(state.resources) as GroupContentsResource | undefined;
        return { owner: resource ? resource.ownerUuid : '' };
    })((props: { owner: string }) => renderOwner(props.owner));

export const renderType = (type: string) =>
    <Typography noWrap>
        {resourceLabel(type)}
    </Typography>;

export const ResourceType = connect(
    (state: RootState, props: { uuid: string }) => {
        const resource = getResource(props.uuid)(state.resources) as GroupContentsResource | undefined;
        return { type: resource ? resource.kind : '' };
    })((props: { type: string }) => renderType(props.type));

export const renderStatus = (item: { status?: string }) =>
    <Typography noWrap align="center" >
        {item.status || "-"}
    </Typography>;

export const ProcessStatus = connect(
    (state: RootState, props: { uuid: string }) => {
        const resource = getResource(props.uuid)(state.resources) as ProcessResource | undefined;
        return { status: resource ? resource.state : '-' };
    })((props: { status: string }) => renderType(props.status));
