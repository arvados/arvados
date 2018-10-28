// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { reduxForm } from 'redux-form';
import { connect } from 'react-redux';
import { compose } from 'redux';
import SharingManagementFormComponent from './sharing-management-form-component';

export const SharingManagementForm = compose(
    connect(() => ({
        initialValues: {
            permissions: [
                { email: 'chrystian.klingenberg@contractors.roche.com', permissions: 'Read' },
                { email: 'artur.janicki@contractors.roche.com', permissions: 'Write' },
            ],
        }
    })),
    reduxForm({ form: 'SHARING_MANAGEMENT_FORM' })
)(SharingManagementFormComponent);