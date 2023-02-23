// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { WorkflowIcon, StartIcon } from 'components/icon/icon';
import {
    WorkflowResource, parseWorkflowDefinition, getWorkflowInputs,
    getWorkflowOutputs, getWorkflow
} from 'models/workflow';
import { DetailsData } from "./details-data";
import { DetailsAttribute } from 'components/details-attribute/details-attribute';
import { ResourceWithName } from 'views-components/data-explorer/renderers';
import { formatDate } from "common/formatters";
import { Grid } from '@material-ui/core';
import { withStyles, StyleRulesCallback, WithStyles, Button } from '@material-ui/core';
import { openRunProcess } from "store/workflow-panel/workflow-panel-actions";
import { Dispatch } from 'redux';
import { connect } from 'react-redux';
import { ArvadosTheme } from 'common/custom-theme';
import { ProcessIOParameter } from 'views/process-panel/process-io-card';
import { formatInputData, formatOutputData } from 'store/process-panel/process-panel-actions';
import { AuthState } from 'store/auth/auth-reducer';
import { RootState } from 'store/store';
import { getPropertyChip } from "views-components/resource-properties-form/property-chip";

export interface WorkflowDetailsCardDataProps {
    workflow?: WorkflowResource;
}

export interface WorkflowDetailsCardActionProps {
    onClick: (wf: WorkflowResource) => () => void;
}

const mapDispatchToProps = (dispatch: Dispatch) => ({
    onClick: (wf: WorkflowResource) =>
        () => wf && dispatch<any>(openRunProcess(wf.uuid, wf.ownerUuid, wf.name)),
});

type CssRules = 'runButton' | 'propertyTag';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    runButton: {
        backgroundColor: theme.customs.colors.green700,
        '&:hover': {
            backgroundColor: theme.customs.colors.green800,
        },
        marginRight: "5px",
        boxShadow: 'none',
        padding: '2px 10px 2px 5px',
        marginLeft: 'auto'
    },
    propertyTag: {
        marginRight: theme.spacing.unit / 2,
        marginBottom: theme.spacing.unit / 2
    },
});

interface AuthStateDataProps {
    auth: AuthState;
};

export interface RegisteredWorkflowPanelDataProps {
    item: WorkflowResource;
    workflowCollection: string;
    inputParams: ProcessIOParameter[];
    outputParams: ProcessIOParameter[];
    gitprops: { [key: string]: string; };
};

export const getRegisteredWorkflowPanelData = (item: WorkflowResource, auth: AuthState): RegisteredWorkflowPanelDataProps => {
    let inputParams: ProcessIOParameter[] = [];
    let outputParams: ProcessIOParameter[] = [];
    let workflowCollection = "";
    const gitprops: { [key: string]: string; } = {};

    // parse definition
    const wfdef = parseWorkflowDefinition(item);

    const inputs = getWorkflowInputs(wfdef);
    if (inputs) {
        inputs.forEach(elm => {
            if (elm.default !== undefined && elm.default !== null) {
                elm.value = elm.default;
            }
        });
        inputParams = formatInputData(inputs, auth);
    }

    const outputs = getWorkflowOutputs(wfdef);
    if (outputs) {
        outputParams = formatOutputData(outputs, {}, undefined, auth);
    }

    const wf = getWorkflow(wfdef);
    if (wf) {
        const REGEX = /keep:([0-9a-f]{32}\+\d+)\/.*/;
        if (wf["steps"]) {
            workflowCollection = wf["steps"][0].run.match(REGEX)[1];
        }
    }

    for (const elm in wfdef) {
        if (elm.startsWith("http://arvados.org/cwl#git")) {
            gitprops[elm.substr(23)] = wfdef[elm]
        }
    }

    return { item, workflowCollection, inputParams, outputParams, gitprops };
};

const mapStateToProps = (state: RootState): AuthStateDataProps => {
    return { auth: state.auth };
};

export const WorkflowDetailsAttributes = connect(mapStateToProps, mapDispatchToProps)(
    withStyles(styles)(
        ({ workflow, onClick, auth, classes }: WorkflowDetailsCardDataProps & AuthStateDataProps & WorkflowDetailsCardActionProps & WithStyles<CssRules>) => {
            if (!workflow) {
                return <Grid />
            }

            const data = getRegisteredWorkflowPanelData(workflow, auth);
            return <Grid container>
                <Button onClick={workflow && onClick(workflow)} className={classes.runButton} variant='contained'
                    data-cy='details-panel-run-btn' color='primary' size='small'>
                    <StartIcon />
                    Run Process
                </Button>
                <Grid item xs={12} >
                    <DetailsAttribute
                        label={"Workflow UUID"}
                        linkToUuid={workflow?.uuid} />
                </Grid>
                <Grid item xs={12} >
                    <DetailsAttribute
                        label='Owner' linkToUuid={workflow?.ownerUuid}
                        uuidEnhancer={(uuid: string) => <ResourceWithName uuid={uuid} />} />
                </Grid>
                <Grid item xs={12}>
                    <DetailsAttribute label='Created at' value={formatDate(workflow?.createdAt)} />
                </Grid>
                <Grid item xs={12}>
                    <DetailsAttribute label='Last modified' value={formatDate(workflow?.modifiedAt)} />
                </Grid>
                <Grid item xs={12} >
                    <DetailsAttribute
                        label='Last modified by user' linkToUuid={workflow?.modifiedByUserUuid}
                        uuidEnhancer={(uuid: string) => <ResourceWithName uuid={uuid} />} />
                </Grid>
                <Grid item xs={12} md={12}>
                    <DetailsAttribute label='Properties' />
                    {Object.keys(data.gitprops).map(k =>
                        getPropertyChip(k, data.gitprops[k], undefined, classes.propertyTag))}
                </Grid>
            </Grid >;
        }));

export class WorkflowDetails extends DetailsData<WorkflowResource> {
    getIcon(className?: string) {
        return <WorkflowIcon className={className} />;
    }

    getDetails() {
        return <WorkflowDetailsAttributes workflow={this.item} />;
    }
}
