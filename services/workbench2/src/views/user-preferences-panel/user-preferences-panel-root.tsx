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

type CssRules = 'root' | 'fullHeight' | 'mainPane' | 'actionPane' | 'inputRow' | 'label' | 'title';

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        width: '100%',
        overflow: 'auto'
    },
    fullHeight: {
        height: '100%',
    },
    mainPane: {
        flexBasis: 0,
        flexGrow: 1,
        flexShrink: 1,
    },
    actionPane: {
        flexBasis: "initial",
    },
    inputRow: {
        // Add some space below each input
        marginBottom: 20,
    },
    label: {
        fontSize: '0.8rem',
        color: theme.palette.grey['600']
    },
    title: {
        fontSize: '1.1rem',
    },
});

export interface UserPreferencesPanelRootDataProps {
    isPristine: boolean;
    isValid: boolean;
    userUuid: string;
    resources: ResourcesState;
}

type UserPreferencesPanelRootProps = InjectedFormProps<{}> & UserPreferencesPanelRootDataProps & DispatchProp & WithStyles<CssRules>;

const ProjectPanelDefaultTabOptions = Object.keys(ProjectPanelTabLabels).map((key) => ({
    key: ProjectPanelTabLabels[key],
    value: ProjectPanelTabLabels[key],
}));

export const UserPreferencesPanelRoot = withStyles(styles)(
    class extends React.Component<UserPreferencesPanelRootProps> {
        render() {
            return (
                <Paper className={this.props.classes.root}>
                    <form className={this.props.classes.fullHeight} onSubmit={this.props.handleSubmit} data-cy="preferences-form">
                        <CardContent className={this.props.classes.fullHeight}>
                            <Grid container direction="column" flexWrap="nowrap" className={this.props.classes.fullHeight}>
                                <Grid item sm={12} overflow="hidden scroll" className={this.props.classes.mainPane}>
                                    <Grid container spacing={3} flexWrap="nowrap" direction="column">
                                        <Grid item>
                                            <Typography className={this.props.classes.title}>
                                                Project Settings
                                            </Typography>
                                        </Grid>
                                        <Grid item className={this.props.classes.inputRow} data-cy="prefs.wb.default_project_tab">
                                            <InputLabel className={this.props.classes.label} htmlFor="prefs.wb.default_project_tab">Default Project Tab</InputLabel>
                                            <Field
                                                id="prefs.wb.default_project_tab"
                                                name="prefs.wb.default_project_tab"
                                                component={RadioField as any}
                                                items={ProjectPanelDefaultTabOptions}
                                                flexRowDirection
                                            />
                                        </Grid>
                                    </Grid>
                                </Grid>
                                <Grid item sm={12} className={this.props.classes.actionPane}>
                                    <Grid container direction="row" justifyContent="flex-end">
                                        <Button color="primary" onClick={this.props.reset} disabled={this.props.isPristine}>Discard changes</Button>
                                        <Button
                                            color="primary"
                                            variant="contained"
                                            type="submit"
                                            disabled={this.props.isPristine || this.props.invalid || this.props.submitting}>
                                            Save changes
                                        </Button>
                                    </Grid>
                                </Grid>
                            </Grid>
                        </CardContent>
                    </form >
                </Paper >
            );
        }
    }
);
