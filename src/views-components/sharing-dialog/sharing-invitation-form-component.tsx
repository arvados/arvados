// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { Field, WrappedFieldProps, FieldArray, WrappedFieldArrayProps } from 'redux-form';
import { Grid, FormControl, InputLabel, Tooltip, IconButton, StyleRulesCallback } from '@material-ui/core';
import { PermissionSelect, parsePermissionLevel, formatPermissionLevel } from './permission-select';
import { ParticipantSelect, Participant } from './participant-select';
import { AddIcon } from 'components/icon/icon';
import { WithStyles } from '@material-ui/core/styles';
import withStyles from '@material-ui/core/styles/withStyles';
import { ArvadosTheme } from 'common/custom-theme';

type SharingStyles = 'root' | 'addButtonRoot' | 'addButtonPrimary' | 'addButtonDisabled';

const styles: StyleRulesCallback<SharingStyles> = (theme: ArvadosTheme) => ({
    root: {
        padding: `${theme.spacing.unit}px 0`,
    },
    addButtonRoot: {
        height: "36px",
        width: "36px",
        marginRight: "6px",
        marginLeft: "6px",
        marginTop: "12px",
    },
    addButtonPrimary: {
        color: theme.palette.primary.contrastText,
        background: theme.palette.primary.main,
        "&:hover": {
            background: theme.palette.primary.dark,
        }
    },
    addButtonDisabled: {
        background: 'none',
    }
});

const SharingInvitationFormComponent = (props: { onSave: () => void, saveEnabled: boolean }) => <StyledSharingInvitationFormComponent onSave={props.onSave} saveEnabled={props.saveEnabled} />

export default SharingInvitationFormComponent;

const StyledSharingInvitationFormComponent = withStyles(styles)(
    ({ onSave, saveEnabled, classes }: { onSave: () => void, saveEnabled: boolean } & WithStyles<SharingStyles>) =>
        <Grid container spacing={8} wrap='nowrap' className={classes.root} >
            <Grid data-cy="invite-people-field" item xs={8}>
                <InvitedPeopleField />
            </Grid>
            <Grid data-cy="permission-select-field" item xs={4} container wrap='nowrap'>
                <PermissionSelectField />
                <IconButton onClick={onSave} disabled={!saveEnabled} color="primary" classes={{
                    root: classes.addButtonRoot,
                    colorPrimary: classes.addButtonPrimary,
                    disabled: classes.addButtonDisabled
                }}>
                    <Tooltip title="Add authorization">
                        <AddIcon />
                    </Tooltip>
                </IconButton>
            </Grid>
        </Grid >);

const InvitedPeopleField = () =>
    <FieldArray
        name='invitedPeople'
        component={InvitedPeopleFieldComponent as any} />;


const InvitedPeopleFieldComponent = ({ fields }: WrappedFieldArrayProps<Participant>) =>
    <ParticipantSelect
        items={fields.getAll() || []}
        onSelect={fields.push}
        onDelete={fields.remove} />;

const PermissionSelectField = () =>
    <Field
        name='permissions'
        component={PermissionSelectComponent}
        format={formatPermissionLevel}
        parse={parsePermissionLevel} />;

const PermissionSelectComponent = ({ input }: WrappedFieldProps) =>
    <FormControl fullWidth>
        <InputLabel>Authorization</InputLabel>
        <PermissionSelect {...input} />
    </FormControl>;
