// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import {
    StyleRulesCallback, WithStyles, withStyles, Card,
    CardHeader, IconButton, CardContent, Grid, Chip
} from '@material-ui/core';
import { ArvadosTheme } from '~/common/custom-theme';
import { ProcessResource } from '~/models/process';
import { DispatchProp, connect } from 'react-redux';
import { MoreOptionsIcon, ProcessIcon } from '~/components/icon/icon';
import { DetailsAttribute } from '~/components/details-attribute/details-attribute';
import { RootState } from '~/store/store';
import { ContextMenuKind } from '~/views-components/context-menu/context-menu';
import { openContextMenu } from '~/store/context-menu/context-menu-actions';

type CssRules = 'card' | 'iconHeader' | 'label' | 'value' | 'chip' | 'headerText' | 'link';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    card: {
        marginBottom: theme.spacing.unit * 2,
        paddingBottom: theme.spacing.unit * 3,
        position: 'relative'
    },
    iconHeader: {
        fontSize: '1.875rem',
        color: theme.customs.colors.green700
    },
    label: {
        display: 'flex',
        justifyContent: 'flex-end',
        fontSize: '0.875rem',
        marginRight: theme.spacing.unit * 3
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
        position: 'absolute',
        top: theme.spacing.unit * 2.5,
        right: theme.spacing.unit * 8,
    },
    headerText: {
        fontSize: '0.875rem',
        marginLeft: theme.spacing.unit * 3,
    }
});

interface ProcessInformationCardDataProps {
    item: ProcessResource;
}

type InformationCardProps = ProcessInformationCardDataProps & DispatchProp & WithStyles<CssRules>;

export const InformationCard = withStyles(styles)(
    connect((state: RootState) => ({
        item: state.collectionPanel.item
    }))(
        class extends React.Component<InformationCardProps> {
            render() {
                const { classes } = this.props;

                return <div>
                    <Card className={classes.card}>
                        <CardHeader
                            avatar={<ProcessIcon className={classes.iconHeader} />}
                            action={
                                <IconButton
                                    aria-label="More options"
                                    onClick={this.handleContextMenu}>
                                    <MoreOptionsIcon />
                                </IconButton>}
                            title="Pipeline template that generates a config file from a template"
                            subheader="(no description)" />
                            <Chip label="Complete" className={classes.chip} />
                        <CardContent>
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
                                    <DetailsAttribute classLabel={classes.link} classValue={classes.value}
                                        label='Outputs' />
                                    <DetailsAttribute classLabel={classes.link} classValue={classes.value}
                                        label='Inputs' />
                                </Grid>
                            </Grid>
                        </CardContent>
                    </Card>
                </div>;
            }

            handleContextMenu = (event: React.MouseEvent<any>) => {
                const resource = {
                    uuid: '',
                    name: '',
                    description: '',
                    kind: ContextMenuKind.PROCESS
                };
                this.props.dispatch<any>(openContextMenu(event, resource));
            }
        }
    )
);