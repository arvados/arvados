// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { Grid } from '@material-ui/core';
import { ProcessInformationCard } from '~/views/process-panel/information-card';
import { SubprocessesCard } from '~/views/process-panel/subprocesses-card';
import { SubprocessFilterDataProps } from '~/components/subprocess-filter/subprocess-filter';

export class ProcessPanel extends React.Component {
    state = {
        filters: [
            {
                key: 'queued',
                value: 1,
                label: 'Queued',
                checked: true
            }, {
                key: 'active',
                value: 2,
                label: 'Active',
                checked: true
            },
            {
                key: 'completed',
                value: 2,
                label: 'Completed',
                checked: true
            },
            {
                key: 'failed',
                value: 2,
                label: 'Failed',
                checked: true
            }
        ]
    };

    onToggle = (filter: SubprocessFilterDataProps) => {
        this.setState((prev: { filters: any[] }) => {
            return {
                filters: prev.filters.map((f: SubprocessFilterDataProps) => {
                    if(f.key === filter.key) {
                        return {
                            ...filter,
                            checked: !filter.checked
                        };
                    }
                    return f;
                })
            };
        });
    }

    render() {
        return <Grid container spacing={16}>
            <Grid item xs={7}>
                <ProcessInformationCard />
            </Grid>
            <Grid item xs={5}>
                <SubprocessesCard 
                    subprocesses={4}
                    filters={this.state.filters}
                    onToggle={this.onToggle}
                />
            </Grid>
        </Grid>;
    }
}
