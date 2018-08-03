// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { Grid, Typography } from '@material-ui/core';
import { FavoriteStar } from '../favorite-star/favorite-star';
import { ResourceKind } from '../../models/resource';
import { ProjectIcon, CollectionIcon, ProcessIcon, DefaultIcon } from '../../components/icon/icon';
import { formatDate, formatFileSize } from '../../common/formatters';
import { resourceLabel } from '../../common/labels';


export const renderName = (item: {name: string; uuid: string, kind: string}) =>
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


export const renderIcon = (item: {kind: string}) => {
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

export const renderDate = (date: string) => {
    return <Typography noWrap>{formatDate(date)}</Typography>;
};

export const renderFileSize = (fileSize?: number) =>
    <Typography noWrap>
        {formatFileSize(fileSize)}
    </Typography>;

export const renderOwner = (owner: string) =>
    <Typography noWrap color="primary" >
        {owner}
    </Typography>;

export const renderType = (type: string) =>
    <Typography noWrap>
        {resourceLabel(type)}
    </Typography>;

export const renderStatus = (item: {status?: string}) =>
    <Typography noWrap align="center" >
        {item.status || "-"}
    </Typography>;