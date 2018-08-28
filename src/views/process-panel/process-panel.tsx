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
import { RouteComponentProps } from 'react-router';
import { MoreOptionsIcon, ProcessIcon } from '~/components/icon/icon';
import { DetailsAttribute } from '~/components/details-attribute/details-attribute';
import { RootState } from '~/store/store';
import { ContextMenuKind } from '~/views-components/context-menu/context-menu';
import { openContextMenu } from '~/store/context-menu/context-menu-actions';

type CssRules = 'card' | 'iconHeader' | 'label' | 'value' | 'content' | 'chip' | 'headerText' | 'link';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    card: {
        marginBottom: theme.spacing.unit * 2,
        width: '60%'
    },
    iconHeader: {
        fontSize: '1.875rem',
        color: theme.customs.colors.green700
    },
    label: {
        fontSize: '0.875rem',
    },
    value: {
        textTransform: 'none',
        fontSize: '0.875rem'
    },
    content: {
        display: 'flex',
        paddingBottom: '0px ',
        paddingTop: '0px',
        '&:last-child': {
            paddingBottom: '0px ',
        }
    },
    link: {
        fontSize: '0.875rem',
        '&:hover': {
            color: theme.palette.primary.main,
            cursor: 'pointer'
        }
    },
    chip: {
        height: theme.spacing.unit * 2.5,
        width: theme.spacing.unit * 12,
        backgroundColor: theme.customs.colors.green700,
        color: theme.palette.common.white,
        fontSize: '0.875rem',
        borderRadius: theme.spacing.unit * 0.625
    },
    headerText: {
        fontSize: '0.875rem',
        display: 'flex',
        position: 'relative',
        justifyContent: 'flex-start',
        top: -theme.spacing.unit * 4.5,
        left: theme.spacing.unit * 3,
    }
});

interface ProcessPanelDataProps {
    item: ProcessResource;
}

interface ProcessPanelActionProps {
    onContextMenu: (event: React.MouseEvent<HTMLElement>, item: ProcessResource) => void;
}

type ProcessPanelProps = ProcessPanelDataProps & ProcessPanelActionProps & DispatchProp & WithStyles<CssRules> & RouteComponentProps<{ id: string }>;

export const ProcessPanel = withStyles(styles)(
    connect((state: RootState) => ({
        item: state.collectionPanel.item,
        tags: state.collectionPanel.tags
    }))(
        class extends React.Component<ProcessPanelProps> {
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
                                </IconButton>
                            }
                            title="Pipeline template that generates a config file from a template" />
                        <CardContent className={classes.content}>
                            <Grid container direction="column">
                                <Grid item xs={8}>
                                    <DetailsAttribute classLabel={classes.label} classValue={classes.value}
                                        label='Status' value={<Chip label="Complete" className={classes.chip} />} />
                                    <DetailsAttribute classLabel={classes.label} classValue={classes.value}
                                        label='Started at' value="1:25 PM 3/23/2018" />
                                    <DetailsAttribute classLabel={classes.label} classValue={classes.value}
                                        label='Finished at' value='1:25 PM 3/23/2018' />
                                </Grid>
                            </Grid>
                            <Grid container direction="column">
                                <Grid item xs={8}>
                                    <DetailsAttribute classLabel={classes.link} classValue={classes.value}
                                        label='Container output' />
                                    <DetailsAttribute classLabel={classes.link} classValue={classes.value}
                                        label='Show inputs' />
                                    <DetailsAttribute classLabel={classes.link} classValue={classes.value}
                                        label='Show command' />
                                </Grid>
                            </Grid>
                        </CardContent>
                        <span className={classes.headerText}>This container request was created from the workflow FastQC MultiQC</span>
                    </Card>
                </div>;
            }

            handleContextMenu = (event: React.MouseEvent<any>) => {
                // const { uuid, name, description } = this.props.item;
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