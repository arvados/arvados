// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { Select, FormControl, InputLabel, MenuItem, Tooltip } from '@mui/material';
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { WorkflowResource } from 'models/workflow';
import { DetailsIcon } from 'components/icon/icon';

export interface WorkflowPresetSelectProps {
    workflow: WorkflowResource;
    selectedPreset: WorkflowResource;
    presets: WorkflowResource[];
    onChange: (preset: WorkflowResource) => void;
}

type CssRules = 'root' | 'icon';

const styles: CustomStyleRulesCallback<CssRules> = theme => ({
    root: {
        display: 'flex',
    },
    icon: {
        color: 'rgba(0, 0, 0, 0.38)',
        marginTop: 18,
        marginLeft: 8,
    },
});

export const WorkflowPresetSelect = withStyles(styles)(
    class extends React.Component<WorkflowPresetSelectProps & WithStyles<CssRules>> {

        render() {

            const { selectedPreset, workflow, presets, classes } = this.props;

            return (
                <div className={classes.root}>
                    <FormControl fullWidth>
                        <InputLabel>Preset</InputLabel>
                        <Select
                            value={selectedPreset.uuid}
                            onChange={(event: any)=>this.handleChange(event)}>
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
