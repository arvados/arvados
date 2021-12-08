// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { Grid } from "@material-ui/core";
import { formatDate } from "common/formatters";
import { resourceLabel } from "common/labels";
import { DetailsAttribute } from "components/details-attribute/details-attribute";
import { ProcessResource } from "models/process";
import { ResourceKind } from "models/resource";
import { ResourceOwnerWithName } from "views-components/data-explorer/renderers";

type CssRules = 'label' | 'value';

export const ProcessDetailsAttributes = (props: { item: ProcessResource, twoCol?: boolean, classes?: Record<CssRules, string> }) => {
    const item = props.item;
    const classes = props.classes || { label: '', value: '', button: '' };
    const mdSize = props.twoCol ? 6 : 12;
    return <Grid container>
        <Grid item xs={12} md={mdSize}>
            <DetailsAttribute label='Type' value={resourceLabel(ResourceKind.PROCESS)} />
        </Grid>
        <Grid item xs={12} md={mdSize}>
            <DetailsAttribute classLabel={classes.label} classValue={classes.value}
                label='Owner' linkToUuid={item.ownerUuid}
                uuidEnhancer={(uuid: string) => <ResourceOwnerWithName uuid={uuid} />} />
        </Grid>
        <Grid item xs={12} md={12}>
            <DetailsAttribute label='Status' value={item.state} />
        </Grid>
        <Grid item xs={12} md={mdSize}>
            <DetailsAttribute label='Last modified' value={formatDate(item.modifiedAt)} />
        </Grid>
        <Grid item xs={12} md={mdSize}>
            <DetailsAttribute label='Started at' value={formatDate(item.createdAt)} />
        </Grid>
        <Grid item xs={12} md={mdSize}>
            <DetailsAttribute label='Created at' value={formatDate(item.createdAt)} />
        </Grid>
        <Grid item xs={12} md={mdSize}>
            <DetailsAttribute label='Finished at' value={formatDate(item.expiresAt)} />
        </Grid>
        <Grid item xs={12} md={mdSize}>
            <DetailsAttribute label='Outputs' value={item.outputPath} />
        </Grid>
        <Grid item xs={12} md={mdSize}>
            <DetailsAttribute label='UUID' linkToUuid={item.uuid} value={item.uuid} />
        </Grid>
        <Grid item xs={12} md={mdSize}>
            <DetailsAttribute label='Container UUID' value={item.containerUuid} />
        </Grid>
        <Grid item xs={12} md={mdSize}>
            <DetailsAttribute label='Priority' value={item.priority} />
        </Grid>
        <Grid item xs={12} md={mdSize}>
            <DetailsAttribute label='Runtime Constraints'
            value={JSON.stringify(item.runtimeConstraints)} />
        </Grid>
        <Grid item xs={12} md={mdSize}>
            <DetailsAttribute label='Docker Image locator'
            linkToUuid={item.containerImage} value={item.containerImage} />
        </Grid>
    </Grid>;
};
