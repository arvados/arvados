// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { ArvadosTheme } from '~/common/custom-theme';
import { StyleRulesCallback, withStyles, WithStyles, Card, CardHeader, CardContent, Grid, Switch } from '@material-ui/core';
import { DetailsAttribute } from '~/components/details-attribute/details-attribute';

type CssRules = 'root' | 'label' | 'value' | 'switch';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {

    },
    label: {
        paddingRight: theme.spacing.unit * 2,
        textAlign: 'right'
    },
    value: {

    },
    switch: {
        height: '18px'
    }
});

type SubprocessesProps = WithStyles<CssRules>;

export const SubprocessesCard = withStyles(styles)(
    class extends React.Component<SubprocessesProps> {

        state = {
            queued: true,
            active: true,
            completed: true,
            failed: true
        };

        handleChange = (name: string) => (event: any) => {
            this.setState({ [name]: event.target.checked });
        }

        render() {
            const { classes } = this.props;
            return (
                <Card className={classes.root}>
                    <CardHeader title="Subprocesses and filters" />
                    <CardContent>
                        <Grid container direction="row">
                            <Grid item xs={3}>
                                <DetailsAttribute classLabel={classes.label} classValue={classes.value}
                                    label='Subprocesses:' value="6" />
                            </Grid>
                        </Grid>
                        <Grid container direction="row">
                            <Grid item xs={3}>
                                <DetailsAttribute classLabel={classes.label} classValue={classes.value}
                                    label='Queued:' value='2'>
                                    <Switch classes={{ bar: classes.switch }}
                                        checked={this.state.queued}
                                        onChange={this.handleChange('queued')}
                                        value="queued"
                                        color="primary" />
                                </DetailsAttribute>
                                <DetailsAttribute classLabel={classes.label} classValue={classes.value}
                                    label='Active:' value='1'>
                                    <Switch classes={{ bar: classes.switch }}
                                        checked={this.state.active}
                                        onChange={this.handleChange('active')}
                                        value="active"
                                        color="primary" />
                                </DetailsAttribute>
                            </Grid>
                            <Grid item xs={3}>
                                <DetailsAttribute classLabel={classes.label} classValue={classes.value}
                                    label='Completed:' value='2'>
                                    <Switch classes={{ bar: classes.switch }}
                                        checked={this.state.completed}
                                        onChange={this.handleChange('completed')}
                                        value="completed"
                                        color="primary" />
                                </DetailsAttribute>
                                <DetailsAttribute classLabel={classes.label} classValue={classes.value}
                                    label='Failed:' value='1'>
                                    <Switch classes={{ bar: classes.switch }}
                                        checked={this.state.failed}
                                        onChange={this.handleChange('failed')}
                                        value="failed"
                                        color="primary" />
                                </DetailsAttribute>
                            </Grid>
                        </Grid>
                    </CardContent>
                </Card>
            );
        }
    }
);