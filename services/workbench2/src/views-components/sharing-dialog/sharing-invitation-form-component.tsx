// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { Field, WrappedFieldProps, FieldArray, WrappedFieldArrayProps } from 'redux-form';
import { Grid, FormControl, InputLabel, StyleRulesCallback, Divider } from '@material-ui/core';
import { PermissionSelect, parsePermissionLevel, formatPermissionLevel } from './permission-select';
import { ParticipantSelect, Participant } from './participant-select';
import { WithStyles } from '@material-ui/core/styles';
import withStyles from '@material-ui/core/styles/withStyles';
import { ArvadosTheme } from 'common/custom-theme';

type SharingStyles = 'root';

const styles: StyleRulesCallback<SharingStyles> = (theme: ArvadosTheme) => ({
    root: {
        padding: `${theme.spacing.unit}px 0`,
    },
});

const SharingInvitationFormComponent = (props: { onSave: () => void }) => <StyledSharingInvitationFormComponent onSave={props.onSave} />

export default SharingInvitationFormComponent;

const StyledSharingInvitationFormComponent = withStyles(styles)(
    ({ classes }: { onSave: () => void } & WithStyles<SharingStyles>) =>
        <Grid container spacing={8} wrap='nowrap' className={classes.root} >
            <Grid data-cy="invite-people-field" item xs={8}>
                <InvitedPeopleField />
            </Grid>
            <Grid data-cy="permission-select-field" item xs={4} container wrap='nowrap'>
                <PermissionSelectField />
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
