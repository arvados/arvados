// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import {
    StyleRulesCallback, WithStyles, withStyles, Card,
    CardHeader, IconButton, CardContent, Grid, Chip, Typography, Tooltip
} from '@material-ui/core';
import { ArvadosTheme } from '~/common/custom-theme';
import { MoreOptionsIcon, ProcessIcon } from '~/components/icon/icon';
import { DetailsAttribute } from '~/components/details-attribute/details-attribute';

type CssRules = 'card' | 'iconHeader' | 'label' | 'value' | 'chip' | 'headerText' | 'link' | 'content' | 'title' | 'avatar';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    card: {
        marginBottom: theme.spacing.unit * 2
    },
    iconHeader: {
        fontSize: '1.875rem',
        color: theme.customs.colors.green700,
    },
    avatar: {
        alignSelf: 'flex-start'
    },
    label: {
        display: 'flex',
        justifyContent: 'flex-end',
        fontSize: '0.875rem',
        marginRight: theme.spacing.unit * 3,
        paddingRight: theme.spacing.unit
    },
    value: {
        textTransform: 'none',
        fontSize: '0.875rem',
    },
    link: {
        fontSize: '0.875rem',
        color: theme.palette.primary.main,
        '&:hover': {
            cursor: 'pointer'
        }
    },
    chip: {
        height: theme.spacing.unit * 3,
        width: theme.spacing.unit * 12,
        backgroundColor: theme.customs.colors.green700,
        color: theme.palette.common.white,
        fontSize: '0.875rem',
        borderRadius: theme.spacing.unit * 0.625,
    },
    headerText: {
        fontSize: '0.875rem',
        marginLeft: theme.spacing.unit * 3,
    },
    content: {
        '&:last-child': {
            paddingBottom: theme.spacing.unit * 2,
            paddingTop: '0px'
        }
    },
    title: {
        overflow: 'hidden'
    }
});

export interface ProcessInformationCardDataProps {
    item: any;
    onContextMenu: (event: React.MouseEvent<HTMLElement>) => void;
}

type ProcessInformationCardProps = ProcessInformationCardDataProps & WithStyles<CssRules>;

export const ProcessInformationCard = withStyles(styles)(
    ({ classes, onContextMenu }: ProcessInformationCardProps) =>
        <Card className={classes.card}>
            <CardHeader
                classes={{
                    content: classes.title,
                    avatar: classes.avatar
                }}
                avatar={<ProcessIcon className={classes.iconHeader} />}
                action={
                    <div>
                        <Chip label="Complete" className={classes.chip} />
                        <IconButton
                            aria-label="More options"
                            onClick={event => onContextMenu(event)}>
                            <MoreOptionsIcon />
                        </IconButton>
                    </div>
                }
                title={
                    <Tooltip title="Pipeline template that generates a config file from a template">
                        <Typography noWrap variant="title">
                            Pipeline template that generates a config file from a template
                        </Typography>
                    </Tooltip>
                }
                subheader="(no-description)" />
            <CardContent className={classes.content}>
                <Grid container>
                    <Grid item xs={6}>
                        <DetailsAttribute classLabel={classes.label} classValue={classes.value}
                            label='From' value="1:25 PM 3/23/2018" />
                        <DetailsAttribute classLabel={classes.label} classValue={classes.value}
                            label='To' value='1:25 PM 3/23/2018' />
                        <DetailsAttribute classLabel={classes.label} classValue={classes.link}
                            label='Workflow' value='FastQC MultiQC' />
                    </Grid>
                    <Grid item xs={6}>
                        <DetailsAttribute classLabel={classes.link} label='Outputs' />
                        <DetailsAttribute classLabel={classes.link} label='Inputs' />
                    </Grid>
                </Grid>
            </CardContent>
        </Card>
);