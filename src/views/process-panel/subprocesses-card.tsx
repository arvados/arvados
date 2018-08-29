// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { ArvadosTheme } from '~/common/custom-theme';
import { StyleRulesCallback, withStyles, WithStyles, Card, CardHeader, CardContent, Grid, Switch } from '@material-ui/core';

type CssRules = 'root' | 'label' | 'value' | 'switch' | 'grid';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        fontSize: '0.875rem'
    },
    label: {
        color: theme.palette.grey["500"],
        marginBottom: theme.spacing.unit
    },
    value: {
        marginBottom: theme.spacing.unit
    },
    switch: {
        '& span:first-child': {
            height: '18px'
        }
    },
    grid: {
        marginLeft: '22px'
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
                    <CardHeader title="Subprocess and filters" />
                    <CardContent>
                        <Grid container direction="row" spacing={16} justify="flex-start" alignItems="stretch">
                            <Grid item>
                                <Grid container direction="column" alignItems="flex-end" spacing={8}>
                                    <Grid item className={classes.label}>Subprocesses:</Grid>
                                    <Grid item className={classes.label}>Queued:</Grid>
                                    <Grid item className={classes.label}>Active:</Grid>
                                </Grid>
                            </Grid>
                            <Grid item>
                                <Grid container direction="column" alignItems="flex-start" spacing={8}>
                                    <Grid item className={classes.value}>1</Grid>
                                    <Grid item className={classes.value}>
                                        2
                                        <Switch classes={{ root: classes.switch }}
                                            checked={this.state.queued}
                                            onChange={this.handleChange('queued')}
                                            value="queued"
                                            color="primary" />
                                    </Grid>
                                    <Grid item className={classes.value}>
                                        3
                                        <Switch classes={{ root: classes.switch }}
                                            checked={this.state.active}
                                            onChange={this.handleChange('active')}
                                            value="active"
                                            color="primary" />
                                    </Grid>
                                </Grid>
                            </Grid>
                            <Grid item className={classes.grid}>
                                <Grid container direction="column" alignItems="flex-end" spacing={8}>
                                    <Grid item className={classes.label}>&nbsp;</Grid>
                                    <Grid item className={classes.label}>Completed:</Grid>
                                    <Grid item className={classes.label}>Failed:</Grid>
                                </Grid>
                            </Grid>
                            <Grid item>
                                <Grid container direction="column" alignItems="flex-end" spacing={8}>
                                    <Grid item className={classes.value}>&nbsp;</Grid>
                                    <Grid item className={classes.value}>
                                        2
                                        <Switch classes={{ root: classes.switch }}
                                            checked={this.state.completed}
                                            onChange={this.handleChange('completed')}
                                            value="completed"
                                            color="primary" />
                                    </Grid>
                                    <Grid item className={classes.value}>
                                        1
                                        <Switch classes={{ root: classes.switch }}
                                            checked={this.state.failed}
                                            onChange={this.handleChange('failed')}
                                            value="failed"
                                            color="primary" />
                                    </Grid>
                                </Grid>
                            </Grid>
                        </Grid>
                    </CardContent>
                </Card>
            );
        }
    }
);