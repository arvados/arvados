// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { reduxForm } from 'redux-form';
import SharingManagementFormComponent from './sharing-management-form-component';
import { SHARING_MANAGEMENT_FORM_NAME } from 'store/sharing-dialog/sharing-dialog-types';

export const SharingManagementForm = reduxForm(
    { form: SHARING_MANAGEMENT_FORM_NAME }
)(SharingManagementFormComponent);
