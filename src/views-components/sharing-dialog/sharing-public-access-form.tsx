// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { reduxForm } from 'redux-form';
import { compose } from 'redux';
import { connect } from 'react-redux';
import SharingPublicAccessFormComponent from './sharing-public-access-form-component';
import { SHARING_PUBLIC_ACCESS_FORM_NAME, VisibilityLevel } from 'store/sharing-dialog/sharing-dialog-types';
import { RootState } from 'store/store';
import { getSharingPublicAccessFormData } from '../../store/sharing-dialog/sharing-dialog-types';

export const SharingPublicAccessForm = compose(
    reduxForm(
        { form: SHARING_PUBLIC_ACCESS_FORM_NAME }
    ),
    connect(
        (state: RootState) => {
            const { visibility } = getSharingPublicAccessFormData(state) || { visibility: VisibilityLevel.PRIVATE };
            return { visibility };
        }
    )
)(SharingPublicAccessFormComponent);

