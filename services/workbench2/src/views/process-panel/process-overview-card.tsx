// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import {  Grid, Typography } from '@mui/material';
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { ArvadosTheme } from 'common/custom-theme';
import { Process } from 'store/processes/process';
import { MPVPanelProps } from 'components/multi-panel-view/multi-panel-view';
import { OverviewPanel } from 'components/overview-panel/overview-panel';
import WarningIcon from '@mui/icons-material/Warning';
import { Link } from "react-router-dom";
import { ProcessProperties } from "store/processes/process";
import { getResourceUrl } from "routes/routes";
import { ProcessAttributes } from './process-attributes';

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
        const containerRequest = process.containerRequest;
        const resubmittedUrl = containerRequest && getResourceUrl(containerRequest.properties[ProcessProperties.FAILED_CONTAINER_RESUBMITTED]);

        return (
            <>
                {resubmittedUrl && <Grid item xs={12}>
                    <Typography>
                        <WarningIcon />
                        This process failed but was automatically resubmitted.  <Link to={resubmittedUrl}> Click here to go to the resubmitted process.</Link>
                    </Typography>
                </Grid>}
                <OverviewPanel detailsElement={<ProcessAttributes request={process.containerRequest} container={process.container} hideProcessPanelRedundantFields />} />
            </>
        );
    }
);
