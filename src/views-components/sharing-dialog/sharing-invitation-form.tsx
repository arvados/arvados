// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { reduxForm } from 'redux-form';
import { connect } from 'react-redux';
import { compose } from 'redux';
import SharingInvitationFormComponent from './sharing-invitation-form-component';
import { SHARING_INVITATION_FORM_NAME } from 'store/sharing-dialog/sharing-dialog-types';
import { PermissionLevel } from 'models/permission';
import { WithStyles } from '@material-ui/core/styles';

interface InvitationFormData {
    permissions: PermissionLevel;
    invitedPeople: string[];
}

interface SaveProps {
    onSave: () => void;
    saveEnabled: boolean;
}

export const SharingInvitationForm =
    reduxForm<InvitationFormData, SaveProps>({
        form: SHARING_INVITATION_FORM_NAME,
        initialValues: {
            permissions: PermissionLevel.CAN_READ,
            invitedPeople: [],
        }
    })(SharingInvitationFormComponent);
