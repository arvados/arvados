// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { Field, WrappedFieldProps, FieldArray, WrappedFieldArrayProps } from 'redux-form';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { Grid, FormControl, InputLabel } from '@mui/material';
import { PermissionSelect, parsePermissionLevel, formatPermissionLevel } from './permission-select';
import { ParticipantSelect, Participant } from './participant-select';
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { ArvadosTheme } from 'common/custom-theme';
import { AutocompleteCat } from 'components/autocomplete/autocomplete';

type SharingStyles = 'root';

const styles: CustomStyleRulesCallback<SharingStyles> = (theme: ArvadosTheme) => ({
    root: {
        padding: `${theme.spacing(1)} 0`,
    },
});

const SharingInvitationFormComponent = (props: { onSave: () => void }) => <StyledSharingInvitationFormComponent onSave={props.onSave} />

export default SharingInvitationFormComponent;

const StyledSharingInvitationFormComponent = withStyles(styles)(
    ({ classes }: { onSave: () => void } & WithStyles<SharingStyles>) =>
        <Grid container spacing={1} wrap='nowrap' className={classes.root} >
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
        onDelete={fields.remove}
        category={AutocompleteCat.SHARING} />;

const PermissionSelectField = () =>
    <Field
        name='permissions'
        component={PermissionSelectComponent}
        format={formatPermissionLevel}
        parse={parsePermissionLevel} />;

const PermissionSelectComponent = ({ input }: WrappedFieldProps) =>
    <FormControl variant="standard" fullWidth>
        <InputLabel>Authorization</InputLabel>
        <PermissionSelect {...input} />
    </FormControl>;
