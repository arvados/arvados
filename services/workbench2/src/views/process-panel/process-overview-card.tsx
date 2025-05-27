// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { CardContent } from '@mui/material';
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { ArvadosTheme } from 'common/custom-theme';
import { Process } from 'store/processes/process';
import { MPVPanelProps } from 'components/multi-panel-view/multi-panel-view';
import { ProcessDetailsAttributes } from './process-details-attributes';

type CssRules = 'card' | 'content';

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    card: {
        height: '100%'
    },
    content: {
        padding: theme.spacing(1),
        paddingTop: theme.spacing(0.5),
        '&:last-child': {
            paddingBottom: theme.spacing(1),
        }
    },
});

export interface ProcessOverviewCardDataProps {
    process: Process;
    cancelProcess: (uuid: string) => void;
    startProcess: (uuid: string) => void;
    resumeOnHoldWorkflow: (uuid: string) => void;
    onContextMenu: (event: React.MouseEvent<HTMLElement>) => void;
}

type ProcessDetailsCardProps = ProcessOverviewCardDataProps & WithStyles<CssRules> & MPVPanelProps;

export const ProcessOverviewCard = withStyles(styles)(
    ({ classes, process }: ProcessDetailsCardProps) => {

        return (
            <section className={classes.card}>
                <CardContent className={classes.content}>
                    <ProcessDetailsAttributes request={process.containerRequest} container={process.container} twoCol hideProcessPanelRedundantFields />
                </CardContent>
            </section>
        );
    }
);
