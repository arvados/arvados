// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { Select, FormControl, InputLabel, MenuItem, Tooltip, withStyles, WithStyles } from '@material-ui/core';
import { WorkflowResource } from 'models/workflow';
import { DetailsIcon } from 'components/icon/icon';

export interface WorkflowPresetSelectProps {
    workflow: WorkflowResource;
    selectedPreset: WorkflowResource;
    presets: WorkflowResource[];
    onChange: (preset: WorkflowResource) => void;
}

type CssRules = 'root' | 'icon';

export const WorkflowPresetSelect = withStyles<CssRules>(theme => ({
    root: {
        display: 'flex',
    },
    icon: {
        color: theme.palette.text.hint,
        marginTop: 18,
        marginLeft: 8,
    },
}))(
    class extends React.Component<WorkflowPresetSelectProps & WithStyles<CssRules>> {

        render() {

            const { selectedPreset, workflow, presets, classes } = this.props;

            return (
                <div className={classes.root}>
                    <FormControl fullWidth>
                        <InputLabel>Preset</InputLabel>
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
                    <Tooltip title='List of already defined set of inputs to run a workflow'>
                        <DetailsIcon className={classes.icon} />
                    </Tooltip>
                </div >
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
    });
