// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { ExpansionPanel, ExpansionPanelDetails, ExpansionPanelSummary } from '@material-ui/core';
import { reduxForm, Field } from 'redux-form';
import { Grid } from '@material-ui/core';
import { TextField } from '~/components/text-field/text-field';
import { ExpandIcon } from '~/components/icon/icon';

export const RUN_PROCESS_ADVANCED_FORM = 'runProcessAdvancedForm';

export interface RunProcessAdvancedFormData {
    output: string;
    runtime: string;
}

export const RunProcessAdvancedForm =
    reduxForm<RunProcessAdvancedFormData>({
        form: RUN_PROCESS_ADVANCED_FORM
    })(() =>
        <form>
            <ExpansionPanel elevation={0}>
                <ExpansionPanelSummary style={{ padding: 0 }} expandIcon={<ExpandIcon />}>
                    Advanced
                </ExpansionPanelSummary>
                <ExpansionPanelDetails style={{ padding: 0 }}>
                    <Grid container spacing={32}>
                        <Grid item xs={12} md={6}>
                            <Field
                                name='output'
                                component={TextField}
                                label="Output name" />
                        </Grid>
                        <Grid item xs={12} md={6}>
                            <Field
                                name='runtime'
                                component={TextField}
                                label="Runtime limit (hh)" />
                        </Grid>
                    </Grid>
                </ExpansionPanelDetails>
            </ExpansionPanel>
        </form >);
