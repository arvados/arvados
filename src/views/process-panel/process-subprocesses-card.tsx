// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import {
    StyleRulesCallback, WithStyles, withStyles, Card,
    CardHeader, IconButton, CardContent, Typography, Tooltip
} from '@material-ui/core';
import * as classnames from "classnames";
import { ArvadosTheme } from '~/common/custom-theme';
import { MoreOptionsIcon } from '~/components/icon/icon';
import { DetailsAttribute } from '~/components/details-attribute/details-attribute';
import { getBackgroundColorStatus } from '~/views/process-panel/process-panel-root';

export type CssRules = 'label' | 'value' | 'title' | 'content' | 'action' | 'options' | 'status' | 'rightSideHeader' | 'titleHeader'
    | 'header' | 'headerActive' | 'headerCompleted' | 'headerQueued' | 'headerFailed' | 'headerCanceled';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    label: {
        fontSize: '0.875rem',
    },
    value: {
        textTransform: 'none',
        fontSize: '0.875rem',
    },
    title: {
        overflow: 'hidden'
    },
    content: {
        paddingTop: theme.spacing.unit * 0.5,
        '&:last-child': {
            paddingBottom: 0
        }
    },
    action: {
        marginTop: 0
    },
    options: {
        width: theme.spacing.unit * 4,
        height: theme.spacing.unit * 4,
        color: theme.palette.common.white,
    },
    status: {
        paddingTop: theme.spacing.unit * 0.5,
        color: theme.palette.common.white,
    },
    rightSideHeader: {
        display: 'flex'
    },
    titleHeader: {
        color: theme.palette.common.white,
        fontWeight: 600
    },
    header: {
        paddingTop: 0,
        paddingBottom: 0,
    },
    headerActive: {
        backgroundColor: theme.customs.colors.blue500,
    },
    headerCompleted: {
        backgroundColor: theme.customs.colors.green700,
    },
    headerQueued: {
        backgroundColor: theme.customs.colors.grey500,
    },
    headerFailed: {
        backgroundColor: theme.customs.colors.red900,
    },
    headerCanceled: {
        backgroundColor: theme.customs.colors.red900,
    },
});

export interface ProcessSubprocessesCardDataProps {
    onContextMenu: (event: React.MouseEvent<HTMLElement>) => void;
    items: any;
}

type ProcessSubprocessesCardProps = ProcessSubprocessesCardDataProps & WithStyles<CssRules>;

export const ProcessSubprocessesCard = withStyles(styles)(
    ({ classes, onContextMenu, items }: ProcessSubprocessesCardProps) => {
        return <Card>
            <CardHeader
                className={classnames([classes.header, getBackgroundColorStatus(items.status, classes)])}
                classes={{ content: classes.title, action: classes.action }}
                action={
                    <div className={classes.rightSideHeader}>
                        <Typography noWrap variant="body2" className={classes.status}>
                            {items.status}
                        </Typography>
                        <IconButton
                            className={classes.options}
                            aria-label="More options"
                            onClick={event => onContextMenu(event)}>
                            <MoreOptionsIcon />
                        </IconButton>
                    </div>
                }
                title={
                    <Tooltip title={items.title}>
                        <Typography noWrap variant="body2" className={classes.titleHeader}>
                            {items.title}
                        </Typography>
                    </Tooltip>
                } />
            <CardContent className={classes.content}>
                <DetailsAttribute classLabel={classes.label} classValue={classes.value}
                    label='Runtime' value="0h 2m" />
            </CardContent>
        </Card>;
    });