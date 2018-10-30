// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { reduxForm } from 'redux-form';
import SharingPublicAccessFormComponent from './sharing-public-access-form-component';
import { SHARING_PUBLIC_ACCESS_FORM_NAME } from '~/store/sharing-dialog/sharing-dialog-types';

export const SharingPublicAccessForm = reduxForm(
    { form: SHARING_PUBLIC_ACCESS_FORM_NAME }
)(SharingPublicAccessFormComponent);
