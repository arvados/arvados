// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { Grid, Typography } from "@mui/material";
import withStyles from '@mui/styles/withStyles';
import { Dispatch } from 'redux';
import { formatCost, formatDate } from "common/formatters";
import { resourceLabel } from "common/labels";
import { DetailsAttribute } from "components/details-attribute/details-attribute";
import { ResourceKind } from "models/resource";
import { CollectionName, ContainerRunTime, ResourceWithName } from "views-components/data-explorer/renderers";
import { getProcess, getProcessStatus, ProcessProperties } from "store/processes/process";
import { RootState } from "store/store";
import { connect } from "react-redux";
import { ProcessResource, MOUNT_PATH_CWL_WORKFLOW } from "models/process";
import { ContainerResource } from "models/container";
import { navigateToOutput, openWorkflow } from "store/process-panel/process-panel-actions";
import { ArvadosTheme } from "common/custom-theme";
import { ProcessRuntimeStatus } from "views-components/process-runtime-status/process-runtime-status";
import { ContainerRequestResource } from "models/container-request";
import { filterResources } from "store/resources/resources";
import { JSONMount, MountType } from 'models/mount-types';
import { getCollectionUrl } from 'models/collection';
import { Link } from "react-router-dom";
import { getResourceUrl } from "routes/routes";
import WarningIcon from '@mui/icons-material/Warning';
import { ResourcesState } from "store/resources/resources";
import { getPropertyChips } from "views-components/property-chips/get-property-chips";

type CssRules = 'link' | 'propertyTag';

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    link: {
        fontSize: '0.875rem',
        color: theme.palette.primary.main,
        '&:hover': {
            cursor: 'pointer'
        }
    },
    propertyTag: {
        marginRight: theme.spacing(0.5),
        marginBottom: theme.spacing(0.5)
    },
});

const mapStateToProps = (state: RootState, props: { request: ProcessResource, container?: ContainerResource }) => {
    return {
        requestUuid: props.request.uuid,
        resources: state.resources,
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

type ProcessDetailsDataProps = {
    request: ProcessResource,
    container?: ContainerResource,
    twoCol?: boolean,
    hideProcessPanelRedundantFields?: boolean,
    classes: Record<CssRules, string>
    requestUuid: string;
    resources: ResourcesState;
}

export const ProcessDetailsAttributes = withStyles(styles, { withTheme: true })(
    connect(mapStateToProps, mapDispatchToProps)(
        (props: ProcessDetailsDataProps & ProcessDetailsAttributesActionProps) => {
            const process = getProcess(props.request.uuid)(props.resources);
            const subprocesses = filterResources((resource: ContainerRequestResource) =>
                (resource.kind === ResourceKind.CONTAINER_REQUEST &&
                    resource.requestingContainerUuid === process?.containerRequest.containerUuid)
            )(props.resources)
            const mounts = process?.containerRequest?.mounts;
            const containerRequest = process?.containerRequest;
            const container = props.container;
            const classes = props.classes;
            const mdSize = props.twoCol ? 6 : 12;
            const { workflowCollection, workflowPath } = parseMounts(mounts);
            const hasTotalCost = containerRequest && containerRequest.cumulativeCost > 0;
            const totalCostNotReady = container && container.cost > 0 && container.state === "Running" && containerRequest && containerRequest.cumulativeCost === 0 && subprocesses.length > 0;
            const resubmittedUrl = containerRequest && getResourceUrl(containerRequest.properties[ProcessProperties.FAILED_CONTAINER_RESUBMITTED]);
            const hasDescription = containerRequest?.description && containerRequest.description.length > 0;

            function parseMounts(mounts: { [path: string]: MountType } | undefined) {
                if (!mounts || !mounts[MOUNT_PATH_CWL_WORKFLOW]) {
                    return { workflowCollection: "", workflowPath: "" };
                }
                const wf = mounts[MOUNT_PATH_CWL_WORKFLOW] as JSONMount;
                let workflowCollection = "";
                let workflowPath = "";
                if (wf.content) {
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
                return { workflowCollection, workflowPath };
            }

            if (!containerRequest) return <></>;

            return <Grid container>
            <Grid item xs={12}>
                <ProcessRuntimeStatus runtimeStatus={container?.runtimeStatus} containerCount={containerRequest.containerCount} />
            </Grid>
            {!props.hideProcessPanelRedundantFields && <Grid item xs={12} md={mdSize}>
                <DetailsAttribute label='Type' value={resourceLabel(ResourceKind.PROCESS)} />
            </Grid>}
            {resubmittedUrl && <Grid item xs={12}>
                <Typography>
                    <WarningIcon />
                    This process failed but was automatically resubmitted.  <Link to={resubmittedUrl}> Click here to go to the resubmitted process.</Link>
                </Typography>
            </Grid>}
            <Grid item xs={12} md={12}>
                <DetailsAttribute label={'Description'}>
                    {hasDescription
                        ? <Typography>{containerRequest.description}</Typography>
                        : <Typography>No description available</Typography>}
                </DetailsAttribute>
            </Grid>
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
            </Grid>}
            {container && workflowCollection && <Grid item xs={12} md={mdSize}>
                <DetailsAttribute label='Workflow code' link={getCollectionUrl(workflowCollection)} value={workflowPath} />
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
                {getPropertyChips(containerRequest, classes)}
            </Grid>
            </Grid>;
        }
    )
);
