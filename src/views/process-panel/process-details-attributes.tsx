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
import { CollectionName, ContainerRunTime, ResourceWithName } from "views-components/data-explorer/renderers";
import { getProcess, getProcessStatus } from "store/processes/process";
import { RootState } from "store/store";
import { connect } from "react-redux";
import { ProcessResource } from "models/process";
import { ContainerResource } from "models/container";
import { navigateToOutput, openWorkflow } from "store/process-panel/process-panel-actions";
import { ArvadosTheme } from "common/custom-theme";
import { ProcessRuntimeStatus } from "views-components/process-runtime-status/process-runtime-status";
import { getPropertyChip } from "views-components/resource-properties-form/property-chip";

type CssRules = 'link' | 'propertyTag';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    link: {
        fontSize: '0.875rem',
        color: theme.palette.primary.main,
        '&:hover': {
            cursor: 'pointer'
        }
    },
    propertyTag: {
        marginRight: theme.spacing.unit / 2,
        marginBottom: theme.spacing.unit / 2
    },
});

const mapStateToProps = (state: RootState, props: { request: ProcessResource }) => {
    return {
        container: getProcess(props.request.uuid)(state.resources)?.container,
    };
};

interface ProcessDetailsAttributesActionProps {
    navigateToOutput: (uuid: string) => void;
    openWorkflow: (uuid: string) => void;
}

const mapDispatchToProps = (dispatch: Dispatch): ProcessDetailsAttributesActionProps => ({
    navigateToOutput: (uuid) => dispatch<any>(navigateToOutput(uuid)),
    openWorkflow: (uuid) => dispatch<any>(openWorkflow(uuid)),
});

export const ProcessDetailsAttributes = withStyles(styles, { withTheme: true })(
    connect(mapStateToProps, mapDispatchToProps)(
        (props: { request: ProcessResource, container?: ContainerResource, twoCol?: boolean, hideProcessPanelRedundantFields?: boolean, classes: Record<CssRules, string> } & ProcessDetailsAttributesActionProps) => {
            const containerRequest = props.request;
            const container = props.container;
            const classes = props.classes;
            const mdSize = props.twoCol ? 6 : 12;
            const filteredPropertyKeys = Object.keys(containerRequest.properties)
                                            .filter(k => (typeof containerRequest.properties[k] !== 'object'));
            return <Grid container>
                <Grid item xs={12}>
                    <ProcessRuntimeStatus runtimeStatus={container?.runtimeStatus} containerCount={containerRequest.containerCount} />
                </Grid>
                {!props.hideProcessPanelRedundantFields && <Grid item xs={12} md={mdSize}>
                    <DetailsAttribute label='Type' value={resourceLabel(ResourceKind.PROCESS)} />
                </Grid>}
                <Grid item xs={12} md={mdSize}>
                    <DetailsAttribute label='Container Request UUID' linkToUuid={containerRequest.uuid} value={containerRequest.uuid} />
                </Grid>
                <Grid item xs={12} md={mdSize}>
                    <DetailsAttribute label='Docker Image locator'
                        linkToUuid={containerRequest.containerImage} value={containerRequest.containerImage} />
                </Grid>
                <Grid item xs={12} md={mdSize}>
                    <DetailsAttribute
                        label='Owner' linkToUuid={containerRequest.ownerUuid}
                        uuidEnhancer={(uuid: string) => <ResourceWithName uuid={uuid} />} />
                </Grid>
                <Grid item xs={12} md={mdSize}>
                    <DetailsAttribute label='Container UUID' value={containerRequest.containerUuid} />
                </Grid>
                {!props.hideProcessPanelRedundantFields && <Grid item xs={12} md={mdSize}>
                    <DetailsAttribute label='Status' value={getProcessStatus({ containerRequest, container })} />
                </Grid>}
                <Grid item xs={12} md={mdSize}>
                    <DetailsAttribute label='Created at' value={formatDate(containerRequest.createdAt)} />
                </Grid>
                <Grid item xs={12} md={mdSize}>
                    <DetailsAttribute label='Started at' value={container ? formatDate(container.startedAt) : "(none)"} />
                </Grid>
                <Grid item xs={12} md={mdSize}>
                    <DetailsAttribute label='Finished at' value={container ? formatDate(container.finishedAt) : "(none)"} />
                </Grid>
                <Grid item xs={12} md={mdSize}>
                    <DetailsAttribute label='Container run time'>
                        <ContainerRunTime uuid={containerRequest.uuid} />
                    </DetailsAttribute>
                </Grid>
                <Grid item xs={12} md={mdSize}>
                    <DetailsAttribute label='Requesting Container UUID' value={containerRequest.requestingContainerUuid || "(none)"} />
                </Grid>
                <Grid item xs={6}>
                    <DetailsAttribute label='Output Collection' />
                    {containerRequest.outputUuid && <span onClick={() => props.navigateToOutput(containerRequest.outputUuid!)}>
                        <CollectionName className={classes.link} uuid={containerRequest.outputUuid} />
                    </span>}
                </Grid>
                {containerRequest.properties.template_uuid &&
                    <Grid item xs={12} md={mdSize}>
                        <span onClick={() => props.openWorkflow(containerRequest.properties.template_uuid)}>
                            <DetailsAttribute classValue={classes.link}
                                label='Workflow' value={containerRequest.properties.workflowName} />
                        </span>
                    </Grid>}
                <Grid item xs={12} md={mdSize}>
                    <DetailsAttribute label='Priority' value={containerRequest.priority} />
                </Grid>
                {/*
                    NOTE: The property list should be kept at the bottom, because it spans
                    the entire available width, without regards of the twoCol prop.
                */}
                <Grid item xs={12} md={12}>
                    <DetailsAttribute label='Properties' />
                    {filteredPropertyKeys.length > 0
                        ? filteredPropertyKeys.map(k =>
                            Array.isArray(containerRequest.properties[k])
                                ? containerRequest.properties[k].map((v: string) =>
                                    getPropertyChip(k, v, undefined, classes.propertyTag))
                                : getPropertyChip(k, containerRequest.properties[k], undefined, classes.propertyTag))
                        : <div>No properties</div>}
                </Grid>
            </Grid>;
        }
    )
);
