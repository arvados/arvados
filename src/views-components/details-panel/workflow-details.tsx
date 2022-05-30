// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { WorkflowIcon } from 'components/icon/icon';
import { WorkflowResource } from 'models/workflow';
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

type CssRules = 'runButton';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    runButton: {
        boxShadow: 'none',
        padding: '2px 10px 2px 5px',
        fontSize: '0.75rem'
    },
});

export const WorkflowDetailsAttributes = connect(null, mapDispatchToProps)(
    withStyles(styles)(
        ({ workflow, onClick, classes }: WorkflowDetailsCardDataProps & WorkflowDetailsCardActionProps & WithStyles<CssRules>) => {
            return <Grid container>
                <Button onClick={workflow && onClick(workflow)} className={classes.runButton} variant='contained'
                    data-cy='details-panel-run-btn' color='primary' size='small'>
                    Run
                </Button>
                {workflow && workflow.description !== "" && <Grid item xs={12} >
                    <DetailsAttribute
                        label={"Description"}
                        value={workflow?.description} />
                </Grid>}
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
