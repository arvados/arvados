// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import {
    StyleRulesCallback,
    WithStyles,
    withStyles,
    Card,
    CardHeader,
    IconButton,
    CardContent,
    Tooltip,
} from '@material-ui/core';
import { ArvadosTheme } from 'common/custom-theme';
import { CloseIcon } from 'components/icon/icon';
import { Process } from 'store/processes/process';
import { MPVPanelProps } from 'components/multi-panel-view/multi-panel-view';
import { ProcessDetailsAttributes } from './process-details-attributes';

type CssRules = 'card' | 'content' | 'title' | 'header';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    card: {
        height: '100%'
    },
    header: {
        paddingTop: theme.spacing.unit,
        paddingBottom: theme.spacing.unit,
    },
    content: {
        '&:last-child': {
            paddingBottom: theme.spacing.unit * 2,
        }
    },
    title: {
        overflow: 'hidden',
        paddingTop: theme.spacing.unit * 0.5
    },
});

export interface ProcessDetailsCardDataProps {
    process: Process;
}

type ProcessDetailsCardProps = ProcessDetailsCardDataProps & WithStyles<CssRules> & MPVPanelProps;

export const ProcessDetailsCard = withStyles(styles)(
    ({ classes, process, doHidePanel, panelName }: ProcessDetailsCardProps) => {
        return <Card className={classes.card}>
            <CardHeader
                className={classes.header}
                classes={{
                    content: classes.title,
                }}
                title='Details'
                action={ doHidePanel &&
                        <Tooltip title={`Close ${panelName || 'panel'}`} disableFocusListener>
                            <IconButton onClick={doHidePanel}><CloseIcon /></IconButton>
                        </Tooltip> } />
            <CardContent className={classes.content}>
                <ProcessDetailsAttributes item={process.containerRequest} twoCol />
            </CardContent>
        </Card>;
    }
);

