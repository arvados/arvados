// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { Field, InjectedFormProps } from "redux-form";
import { DispatchProp } from 'react-redux';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import {
    CardContent,
    Typography,
    Grid,
    Paper,
    InputLabel,
    Button,
} from '@mui/material';
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { ArvadosTheme } from 'common/custom-theme';
import { ResourcesState } from 'store/resources/resources';
import { ProjectPanelTabLabels } from 'store/project-panel/project-panel-action';
import { RadioField } from 'components/radio-field/radio-field';

type CssRules = 'root' | 'emptyRoot' | 'gridItem' | 'label' | 'readOnlyValue' | 'title' | 'actions';

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        width: '100%',
        overflow: 'auto'
    },
    emptyRoot: {
        width: '100%',
        overflow: 'auto',
        padding: theme.spacing(4),
    },
    gridItem: {
        height: 45,
        marginBottom: 20
    },
    label: {
        fontSize: '0.8rem',
        color: theme.palette.grey['600']
    },
    readOnlyValue: {
        fontSize: '0.875rem',
    },
    title: {
        fontSize: '1.1rem',
    },
    actions: {
        display: 'flex',
        justifyContent: 'flex-end'
    }
});

export interface UserPreferencesPanelRootDataProps {
    isPristine: boolean;
    isValid: boolean;
    userUuid: string;
    resources: ResourcesState;
}

type UserPreferencesPanelRootProps = InjectedFormProps<{}> & UserPreferencesPanelRootDataProps & DispatchProp & WithStyles<CssRules>;

export const UserPreferencesPanelRoot = withStyles(styles)(
    class extends React.Component<UserPreferencesPanelRootProps> {
        render() {
            return (
                <Paper className={this.props.classes.root}>
                    <CardContent>
                        <Grid container justifyContent="space-between">
                            <Grid item>
                                <Typography className={this.props.classes.title}>
                                    User Preferences
                                </Typography>
                            </Grid>
                        </Grid>
                        <form onSubmit={this.props.handleSubmit} data-cy="profile-form">
                            <Grid container spacing={3}>
                            </Grid>
                        </form >
                    </CardContent>
                </Paper >
            );
        }
    }
);
