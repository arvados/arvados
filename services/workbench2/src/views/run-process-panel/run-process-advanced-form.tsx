// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { Accordion, AccordionDetails, AccordionSummary } from '@mui/material';
import { reduxForm, Field } from 'redux-form';
import { Grid } from '@mui/material';
import { TextField } from 'components/text-field/text-field';
import { ExpandIcon } from 'components/icon/icon';
import * as IntInput from './inputs/int-input';
import { min } from 'validators/min';
import { optional } from 'validators/optional';
import { RUN_PROCESS_ADVANCED_FORM,
         OUTPUT_FIELD,
         RUNTIME_FIELD,
         RAM_FIELD,
         VCPUS_FIELD,
         KEEP_CACHE_RAM_FIELD,
         RUNNER_IMAGE_FIELD,
         RunProcessAdvancedFormData
} from 'store/run-process-panel/run-process-panel-actions';

export const RunProcessAdvancedForm =
    reduxForm<RunProcessAdvancedFormData>({
        form: RUN_PROCESS_ADVANCED_FORM,
    })(() =>
        <form>
            <Accordion elevation={0}>
                <AccordionSummary style={{ padding: 0 }} expandIcon={<ExpandIcon />}>
                    Advanced
                </AccordionSummary>
                <AccordionDetails style={{ padding: 0 }}>
                    <Grid container spacing={4}>
                        <Grid item xs={12} md={6}>
                            <Field
                                name={OUTPUT_FIELD}
                                component={TextField as any}
                                label="Output name" />
                        </Grid>
                        <Grid item xs={12} md={6}>
                            <Field
                                name={RUNTIME_FIELD}
                                component={TextField as any}
                                helperText="Maximum running time (in seconds) that this container will be allowed to run before being cancelled."
                                label="Runtime limit"
                                parse={IntInput.parse}
                                format={IntInput.format}
                                type='number'
                                validate={runtimeValidation} />
                        </Grid>
                        <Grid item xs={12} md={6}>
                            <Field
                                name={RAM_FIELD}
                                component={TextField as any}
                                label="RAM"
                                helperText="Number of ram bytes to be used to run this process."
                                parse={IntInput.parse}
                                format={IntInput.format}
                                type='number'
                                required
                                validate={ramValidation} />
                        </Grid>
                        <Grid item xs={12} md={6}>
                            <Field
                                name={VCPUS_FIELD}
                                component={TextField as any}
                                label="VCPUs"
                                helperText="Number of cores to be used to run this process."
                                parse={IntInput.parse}
                                format={IntInput.format}
                                type='number'
                                required
                                validate={vcpusValidation} />
                        </Grid>
                        <Grid item xs={12} md={6}>
                            <Field
                                name={KEEP_CACHE_RAM_FIELD}
                                component={TextField as any}
                                label="Keep cache RAM"
                                helperText="Number of keep cache bytes to be used to run this process."
                                parse={IntInput.parse}
                                format={IntInput.format}
                                type='number'
                                validate={keepCacheRamValidation} />
                        </Grid>
                        <Grid item xs={12} md={6}>
                            <Field
                                name={RUNNER_IMAGE_FIELD}
                                component={TextField as any}
                                label='Runner'
                                required
                                helperText='The container image with arvados-cwl-runner that will execute this workflow.' />
                        </Grid>
                    </Grid>
                </AccordionDetails>
            </Accordion>
        </form >);

const ramValidation = [min(0)];
const vcpusValidation = [min(1)];
const keepCacheRamValidation = [optional(min(0))];
const runtimeValidation = [optional(min(1))];
