// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { reduxForm } from 'redux-form';
import { connect } from 'react-redux';
import { compose } from 'redux';
import SharingPublicAccessFormComponent from './sharing-public-access-form-component';
export const SharingPublicAccessForm = compose(
    connect(() => ({
        initialValues: {
            enabled: false,
            permissions: 'Read',
        }
    })),
    reduxForm({ form: 'SHARING_PUBLIC_ACCESS_FORM' })
)(SharingPublicAccessFormComponent);