// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import {
    StyleRulesCallback, WithStyles, withStyles, Card,
    CardHeader, IconButton, CardContent, Grid
} from '@material-ui/core';
import { ArvadosTheme } from '~/common/custom-theme';
import { ProcessResource } from '~/models/process';
import { DispatchProp, connect } from 'react-redux';
import { RouteComponentProps } from 'react-router';
import { MoreOptionsIcon, ProcessIcon } from '~/components/icon/icon';
import { DetailsAttribute } from '~/components/details-attribute/details-attribute';
import { RootState } from '~/store/store';

type CssRules = 'card' | 'iconHeader' | 'label' | 'value';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    card: {
        marginBottom: theme.spacing.unit * 2
    },
    iconHeader: {
        fontSize: '1.875rem',
        color: theme.customs.colors.green700
    },
    label: {
        fontSize: '0.875rem'
    },
    value: {
        textTransform: 'none',
        fontSize: '0.875rem'
    }
});

interface ProcessPanelDataProps {
    item: ProcessResource;
}

interface ProcessPanelActionProps {
    onItemRouteChange: (processId: string) => void;
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
                const { classes, onContextMenu, item } = this.props;

                return <div>
                    <Card className={classes.card}>
                        <CardHeader
                            avatar={<ProcessIcon className={classes.iconHeader} />}
                            action={
                                <IconButton
                                    aria-label="More options"
                                    onClick={event => onContextMenu(event, item)}>
                                    <MoreOptionsIcon />
                                </IconButton>
                            }
                            title="Pipeline template that generates a config file from a template"
                            subheader="(no description)"
                        />
                        <CardContent>
                            <Grid container direction="column">
                                <Grid item xs={6}>
                                    <DetailsAttribute classLabel={classes.label} classValue={classes.value}
                                        label='Collection UUID' value="uuid" />
                                    <DetailsAttribute classLabel={classes.label} classValue={classes.value}
                                        label='Number of files' value='14' />
                                    <DetailsAttribute classLabel={classes.label} classValue={classes.value}
                                        label='Content size' value='54 MB' />
                                    <DetailsAttribute classLabel={classes.label} classValue={classes.value}
                                        label='Owner' value="ownerUuid" />
                                </Grid>
                            </Grid>
                        </CardContent>
                    </Card>
                </div>;
            }
            componentWillReceiveProps({ match, item, onItemRouteChange }: ProcessPanelProps) {
                if (!item || match.params.id !== item.uuid) {
                    onItemRouteChange(match.params.id);
                }
            }
        }
    )
);