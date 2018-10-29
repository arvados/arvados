// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { reduxForm } from 'redux-form';
import { connect } from 'react-redux';
import { compose } from 'redux';
import SharingPublicAccessFormComponent from './sharing-public-access-form-component';
import { SHARING_PUBLIC_ACCESS_FORM_NAME } from '~/store/sharing-dialog/sharing-dialog-types';
import { PermissionLevel } from '~/models/permission';
export const SharingPublicAccessForm = compose(
    connect(() => ({
        initialValues: {
            enabled: false,
            permissions: PermissionLevel.CAN_READ,
        }
    })),
    reduxForm({ form: SHARING_PUBLIC_ACCESS_FORM_NAME })
)(SharingPublicAccessFormComponent);