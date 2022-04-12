// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { Grid, StyleRulesCallback, withStyles } from "@material-ui/core";
import { Dispatch } from 'redux';
import { formatDate } from "common/formatters";
import { resourceLabel } from "common/labels";
import { DetailsAttribute } from "components/details-attribute/details-attribute";
import { ResourceKind } from "models/resource";
import { ResourceOwnerWithName } from "views-components/data-explorer/renderers";
import { getProcess } from "store/processes/process";
import { RootState } from "store/store";
import { connect } from "react-redux";
import { ProcessResource } from "models/process";
import { ContainerResource } from "models/container";
import { openProcessInputDialog } from "store/processes/process-input-actions";
import { navigateToOutput, openWorkflow } from "store/process-panel/process-panel-actions";
import { ArvadosTheme } from "common/custom-theme";

type CssRules = 'link';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    link: {
        fontSize: '0.875rem',
        color: theme.palette.primary.main,
        '&:hover': {
            cursor: 'pointer'
        }
    },
});

const mapStateToProps = (state: RootState, props: { request: ProcessResource }) => {
    return {
        container: getProcess(props.request.uuid)(state.resources)?.container,
    };
};

interface ProcessDetailsAttributesActionProps {
    openProcessInputDialog: (uuid: string) => void;
    navigateToOutput: (uuid: string) => void;
    openWorkflow: (uuid: string) => void;
}

const mapDispatchToProps = (dispatch: Dispatch): ProcessDetailsAttributesActionProps => ({
    openProcessInputDialog: (uuid) => dispatch<any>(openProcessInputDialog(uuid)),
    navigateToOutput: (uuid) => dispatch<any>(navigateToOutput(uuid)),
    openWorkflow: (uuid) => dispatch<any>(openWorkflow(uuid)),
});

export const ProcessDetailsAttributes = withStyles(styles, { withTheme: true })(
    connect(mapStateToProps, mapDispatchToProps)(
        (props: { request: ProcessResource, container?: ContainerResource, twoCol?: boolean, classes?: Record<CssRules, string> } & ProcessDetailsAttributesActionProps) => {
            const containerRequest = props.request;
            const container = props.container;
            const classes = props.classes || { label: '', value: '', button: '', link: '' };
            const mdSize = props.twoCol ? 6 : 12;
            return <Grid container>
                <Grid item xs={12} md={mdSize}>
                    <DetailsAttribute label='Type' value={resourceLabel(ResourceKind.PROCESS)} />
                </Grid>
                <Grid item xs={12} md={mdSize}>
                    <DetailsAttribute
                        label='Owner' linkToUuid={containerRequest.ownerUuid}
                        uuidEnhancer={(uuid: string) => <ResourceOwnerWithName uuid={uuid} />} />
                </Grid>
                <Grid item xs={12} md={12}>
                    <DetailsAttribute label='Status' value={containerRequest.state} />
                </Grid>
                <Grid item xs={12} md={mdSize}>
                    <DetailsAttribute label='Last modified' value={formatDate(containerRequest.modifiedAt)} />
                </Grid>
                <Grid item xs={12} md={mdSize}>
                    <DetailsAttribute label='Created at' value={formatDate(containerRequest.createdAt)} />
                </Grid>
                <Grid item xs={12} md={mdSize}>
                    <DetailsAttribute label='Started at' value={container ? formatDate(container.startedAt) : "N/A"} />
                </Grid>
                <Grid item xs={12} md={mdSize}>
                    <DetailsAttribute label='Finished at' value={container ? formatDate(container.finishedAt) : "N/A"} />
                </Grid>
                <Grid item xs={12} md={mdSize}>
                    <DetailsAttribute label='Expires at' value={formatDate(containerRequest.expiresAt)} />
                </Grid>
                <Grid item xs={12} md={mdSize}>
                    <DetailsAttribute label='Outputs' value={containerRequest.outputPath} />
                </Grid>
                <Grid item xs={12} md={mdSize}>
                    <DetailsAttribute label='UUID' linkToUuid={containerRequest.uuid} value={containerRequest.uuid} />
                </Grid>
                <Grid item xs={12} md={mdSize}>
                    <DetailsAttribute label='Container UUID' value={containerRequest.containerUuid} />
                </Grid>
                <Grid item xs={6}>
                    <span onClick={() => props.navigateToOutput(containerRequest.outputUuid!)}>
                        <DetailsAttribute classLabel={classes.link} label='Outputs' />
                    </span>
                    <span onClick={() => props.openProcessInputDialog(containerRequest.uuid)}>
                        <DetailsAttribute classLabel={classes.link} label='Inputs' />
                    </span>
                </Grid>
                {containerRequest.properties.workflowUuid &&
                <Grid item xs={12} md={mdSize}>
                    <span onClick={() => props.openWorkflow(containerRequest.properties.workflowUuid)}>
                        <DetailsAttribute classValue={classes.link}
                            label='Workflow' value={containerRequest.properties.workflowName} />
                    </span>
                </Grid>}
                <Grid item xs={12} md={mdSize}>
                    <DetailsAttribute label='Priority' value={containerRequest.priority} />
                </Grid>
                <Grid item xs={12} md={mdSize}>
                    <DetailsAttribute label='Runtime Constraints'
                    value={JSON.stringify(containerRequest.runtimeConstraints)} />
                </Grid>
                <Grid item xs={12} md={mdSize}>
                    <DetailsAttribute label='Docker Image locator'
                    linkToUuid={containerRequest.containerImage} value={containerRequest.containerImage} />
                </Grid>
            </Grid>;
        }
    )
);
