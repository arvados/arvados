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

const permissionManagementRowStyles: StyleRulesCallback<'root'> = theme => ({
    root: {
        padding: `${theme.spacing.unit}px 0`,
    }
});

const SharingInvitationFormComponent = (props: { onSave: () => void, saveEnabled: boolean }) =>
    <Grid container spacing={8} >
        <Grid data-cy="invite-people-field" item xs={8}>
            <InvitedPeopleField />
        </Grid>
        <Grid data-cy="permission-select-field" item xs={4} container wrap='nowrap'>
            <PermissionSelectField />
            <Tooltip title="Add authorization">
                <IconButton onClick={props.onSave} disabled={!props.saveEnabled} color="primary">
                    <AddIcon />
                </IconButton>
            </Tooltip>
        </Grid>
    </Grid>;

export default SharingInvitationFormComponent;

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
