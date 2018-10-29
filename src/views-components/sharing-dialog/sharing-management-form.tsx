// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { reduxForm } from 'redux-form';
import { connect } from 'react-redux';
import { compose } from 'redux';
import SharingManagementFormComponent from './sharing-management-form-component';
import { SHARING_MANAGEMENT_FORM_NAME } from '~/store/sharing-dialog/sharing-dialog-types';
import { PermissionLevel } from '~/models/permission';

export const SharingManagementForm = compose(
    connect(() => ({
        initialValues: {
            permissions: [
                {
                    email: 'chrystian.klingenberg@contractors.roche.com',
                    permissions: PermissionLevel.CAN_MANAGE,
                },
                {
                    email: 'artur.janicki@contractors.roche.com',
                    permissions: PermissionLevel.CAN_WRITE,
                },
            ],
        }
    })),
    reduxForm({ form: SHARING_MANAGEMENT_FORM_NAME })
)(SharingManagementFormComponent);