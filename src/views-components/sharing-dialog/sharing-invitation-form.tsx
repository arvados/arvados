// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { reduxForm } from 'redux-form';
import { connect } from 'react-redux';
import { compose } from 'redux';
import SharingInvitationFormComponent from './sharing-invitation-form-component';
import { PermissionSelectValue } from './permission-select';

export const SharingInvitationForm = compose(
    connect(() => ({
        initialValues: {
            permission: PermissionSelectValue.READ
        }
    })),
    reduxForm({ form: 'SIMPLE_SHARING_FORM' })
)(SharingInvitationFormComponent);