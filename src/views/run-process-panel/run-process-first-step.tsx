// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { StyleRulesCallback, withStyles, Grid, Button, WithStyles, List, ListItem, ListItemText, ListItemIcon, Tabs, Tab } from '@material-ui/core';
import { ArvadosTheme } from '~/common/custom-theme';
import { WorkflowResource } from '~/models/workflow';
import { WorkflowIcon } from '~/components/icon/icon';
import { WorkflowDetailsCard } from '../workflow-panel/workflow-description-card';

type CssRules = 'rightGrid' | 'list' | 'listItem' | 'itemSelected' | 'listItemText' | 'listItemIcon';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    rightGrid: {
        borderLeft: `1px solid ${theme.palette.grey["300"]}`
    },
    list: {
        maxHeight: 300,
        position: 'relative',
        overflow: 'auto'
    },
    listItem: {
        padding: theme.spacing.unit,
    },
    itemSelected: {
        backgroundColor: 'rgba(3, 190, 171, 0.3) !important'
    },
    listItemText: {
        fontSize: '0.875rem'
    },
    listItemIcon: {
        color: theme.customs.colors.red900
    }
});

export interface RunProcessFirstStepDataProps {
    workflows: WorkflowResource[];
    selectedWorkflow: WorkflowResource | undefined;
}

export interface RunProcessFirstStepActionProps {
    onSetStep: (step: number) => void;
    onSetWorkflow: (workflow: WorkflowResource) => void;
}

type RunProcessFirstStepProps = RunProcessFirstStepDataProps & RunProcessFirstStepActionProps & WithStyles<CssRules>;

export const RunProcessFirstStep = withStyles(styles)(
    ({ onSetStep, onSetWorkflow, workflows, selectedWorkflow, classes }: RunProcessFirstStepProps) =>
        <Grid container spacing={16}>
            <Grid container item xs={6}>
                <Grid item xs={12}>
                    {/* TODO: add filters */}
                </Grid>
                <Grid item xs={12}>
                    <List className={classes.list}>
                        {workflows.map(workflow => (
                            <ListItem key={workflow.uuid} button
                                classes={{ root: classes.listItem, selected: classes.itemSelected}}
                                selected={selectedWorkflow && (selectedWorkflow.uuid === workflow.uuid)}
                                onClick={() => onSetWorkflow(workflow)}>
                                <ListItemIcon>
                                    <WorkflowIcon className={classes.listItemIcon}/>
                                </ListItemIcon>
                                <ListItemText className={classes.listItemText} primary={workflow.name} disableTypography={true} />
                            </ListItem>
                        ))}
                    </List>
                </Grid>
            </Grid>
            <Grid item xs={6} className={classes.rightGrid}>
                <WorkflowDetailsCard workflow={selectedWorkflow}/>
            </Grid>
            <Grid item xs={12}>
                <Button variant="contained" color="primary" 
                    disabled={!(!!selectedWorkflow)}
                    onClick={() => onSetStep(1)}>
                    Next
                </Button>
            </Grid>
        </Grid>
);
