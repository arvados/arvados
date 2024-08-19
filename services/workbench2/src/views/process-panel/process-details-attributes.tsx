// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { Grid, StyleRulesCallback, withStyles, Typography } from "@material-ui/core";
import { Dispatch } from 'redux';
import { formatCost, formatDate } from "common/formatters";
import { resourceLabel } from "common/labels";
import { DetailsAttribute } from "components/details-attribute/details-attribute";
import { ResourceKind } from "models/resource";
import { CollectionName, ContainerRunTime, ResourceWithName } from "views-components/data-explorer/renderers";
import { getProcess, getProcessStatus } from "store/processes/process";
import { RootState } from "store/store";
import { connect } from "react-redux";
import { ProcessResource, MOUNT_PATH_CWL_WORKFLOW } from "models/process";
import { ContainerResource } from "models/container";
import { navigateToOutput, openWorkflow } from "store/process-panel/process-panel-actions";
import { ArvadosTheme } from "common/custom-theme";
import { ProcessRuntimeStatus } from "views-components/process-runtime-status/process-runtime-status";
import { getPropertyChip } from "views-components/resource-properties-form/property-chip";
import { ContainerRequestResource } from "models/container-request";
import { filterResources } from "store/resources/resources";
import { JSONMount } from 'models/mount-types';
import { getCollectionUrl } from 'models/collection';

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
    const process = getProcess(props.request.uuid)(state.resources);

    let workflowCollection = "";
    let workflowPath = "";
    let schedulingStatus = "";
    if (process?.containerRequest?.mounts && process.containerRequest.mounts[MOUNT_PATH_CWL_WORKFLOW]) {
        const wf = process.containerRequest.mounts[MOUNT_PATH_CWL_WORKFLOW] as JSONMount;

	if (process?.container &&
	    state.processPanel.containerStatus?.uuid === process?.container?.uuid)
	{
	    schedulingStatus = state.processPanel.containerStatus.schedulingStatus;
	}

        if (wf.content["$graph"] &&
	    wf.content["$graph"].length > 0 &&
	    wf.content["$graph"][0] &&
	    wf.content["$graph"][0]["steps"] &&
	    wf.content["$graph"][0]["steps"][0]) {

		const REGEX = /keep:([0-9a-f]{32}\+\d+)\/(.*)/;
            const pdh = wf.content["$graph"][0]["steps"][0].run.match(REGEX);
            if (pdh) {
                workflowCollection = pdh[1];
                workflowPath = pdh[2];
            }
        }
    }

    return {
        container: process?.container,
        workflowCollection,
        workflowPath,
	schedulingStatus,
        subprocesses: filterResources((resource: ContainerRequestResource) =>
            resource.kind === ResourceKind.CONTAINER_REQUEST &&
									    resource.requestingContainerUuid === process?.containerRequest.containerUuid
        )(state.resources),
    };
};

interface ProcessDetailsAttributesActionProps {
    navigateToOutput: (resource: ContainerRequestResource) => void;
    openWorkflow: (uuid: string) => void;
}

const mapDispatchToProps = (dispatch: Dispatch): ProcessDetailsAttributesActionProps => ({
    navigateToOutput: (resource) => dispatch<any>(navigateToOutput(resource)),
    openWorkflow: (uuid) => dispatch<any>(openWorkflow(uuid)),
});

export const ProcessDetailsAttributes = withStyles(styles, { withTheme: true })(
    connect(mapStateToProps, mapDispatchToProps)(
        (props: {
            request: ProcessResource,
	    container?: ContainerResource,
	    subprocesses: ContainerRequestResource[],
            workflowCollection,
	    workflowPath,
	    schedulingStatus,
            twoCol?: boolean,
	    hideProcessPanelRedundantFields?: boolean,
	    classes: Record<CssRules, string>
        } & ProcessDetailsAttributesActionProps) => {
            const containerRequest = props.request;
            const container = props.container;
            const subprocesses = props.subprocesses;
            const classes = props.classes;
            const mdSize = props.twoCol ? 6 : 12;
            const workflowCollection = props.workflowCollection;
            const workflowPath = props.workflowPath;
            const filteredPropertyKeys = Object.keys(containerRequest.properties)
                .filter(k => (typeof containerRequest.properties[k] !== 'object'));
            const hasTotalCost = containerRequest && containerRequest.cumulativeCost > 0;
            const totalCostNotReady = container && container.cost > 0 && container.state === "Running" && containerRequest && containerRequest.cumulativeCost === 0 && subprocesses.length > 0;
            return <Grid container>
                <Grid item xs={12}>
                    <ProcessRuntimeStatus runtimeStatus={container?.runtimeStatus} containerCount={containerRequest.containerCount} />
                </Grid>
                {!props.hideProcessPanelRedundantFields && <Grid item xs={12} md={mdSize}>
                    <DetailsAttribute label='Type' value={resourceLabel(ResourceKind.PROCESS)} />
                </Grid>}
            {props.schedulingStatus !== "" && <Grid item xs={12} md={12}>
                <Typography>{props.schedulingStatus}</Typography>
		    </Grid>}

		<Grid item xs={12} md={mdSize}>
                    <DetailsAttribute label='Container request UUID' linkToUuid={containerRequest.uuid} value={containerRequest.uuid} />
		</Grid>
		<Grid item xs={12} md={mdSize}>
                    <DetailsAttribute label='Docker image locator'
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
            {(containerRequest && containerRequest.modifiedByUserUuid) && <Grid item xs={12} md={mdSize} data-cy="process-details-attributes-modifiedby-user">
                <DetailsAttribute
                    label='Submitted by' linkToUuid={containerRequest.modifiedByUserUuid}
                    uuidEnhancer={(uuid: string) => <ResourceWithName uuid={uuid} />} />
            </Grid>}
                {(container && container.runtimeUserUuid && container.runtimeUserUuid !== containerRequest.modifiedByUserUuid) && <Grid item xs={12} md={mdSize} data-cy="process-details-attributes-runtime-user">
                    <DetailsAttribute
                        label='Run as' linkToUuid={container.runtimeUserUuid}
                        uuidEnhancer={(uuid: string) => <ResourceWithName uuid={uuid} />} />
                </Grid>}
                <Grid item xs={12} md={mdSize}>
                    <DetailsAttribute label='Requesting container UUID' value={containerRequest.requestingContainerUuid || "(none)"} />
                </Grid>
                <Grid item xs={6}>
                    <DetailsAttribute label='Output collection' />
                    {containerRequest.outputUuid && <span onClick={() => props.navigateToOutput(containerRequest!)}>
                        <CollectionName className={classes.link} uuid={containerRequest.outputUuid} />
                    </span>}
                </Grid>
                {container && <Grid item xs={12} md={mdSize}>
                    <DetailsAttribute label='Cost' value={
                        `${hasTotalCost ? formatCost(containerRequest.cumulativeCost) + ' total, ' : (totalCostNotReady ? 'total pending completion, ' : '')}${container.cost > 0 ? formatCost(container.cost) : 'not available'} for this container`
                    } />

                    {container && workflowCollection && <Grid item xs={12} md={mdSize}>
                        <DetailsAttribute label='Workflow code' link={getCollectionUrl(workflowCollection)} value={workflowPath} />
                    </Grid>}
                </Grid>}
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
