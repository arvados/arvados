// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { Select, FormControl, InputLabel, MenuItem, Tooltip, Grid, withStyles, WithStyles } from '@material-ui/core';
import { WorkflowResource } from '~/models/workflow';
import { DetailsIcon } from '~/components/icon/icon';

export interface WorkflowPresetSelectProps {
    workflow: WorkflowResource;
    selectedPreset: WorkflowResource;
    presets: WorkflowResource[];
    onChange: (preset: WorkflowResource) => void;
}

export class WorkflowPresetSelect extends React.Component<WorkflowPresetSelectProps> {

    render() {

        const { selectedPreset, workflow, presets } = this.props;

        return (
            <Grid container wrap='nowrap' spacing={32}>
                <Grid item xs container>
                    <Grid item xs>
                        <FormControl fullWidth>
                            <InputLabel htmlFor="age-simple">Preset</InputLabel>
                            <Select
                                value={selectedPreset.uuid}
                                onChange={this.handleChange}>
                                <MenuItem value={workflow.uuid}>
                                    <em>Default</em>
                                </MenuItem>
                                {presets.map(
                                    ({ uuid, name }) => <MenuItem key={uuid} value={uuid}>{name}</MenuItem>
                                )}
                            </Select>
                        </FormControl>
                    </Grid>
                    <WorkflowPresetSelectInfo />
                </Grid>
            </Grid>
        );
    }

    handleChange = ({ target }: React.ChangeEvent<HTMLSelectElement>) => {

        const { workflow, presets, onChange } = this.props;

        const selectedPreset = [workflow, ...presets]
            .find(({ uuid }) => uuid === target.value);

        if (selectedPreset) {
            onChange(selectedPreset);
        }
    }
}

const WorkflowPresetSelectInfo = withStyles<'icon'>(theme => ({
    icon: {
        marginTop: 18,
        marginLeft: 8,
    },
}))(
    ({ classes }: WithStyles<'icon'>) =>
        <Tooltip title='List of already defined set of inputs to run a workflow'>
            <DetailsIcon className={classes.icon} />
        </Tooltip>
);
