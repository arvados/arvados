// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0


import { reduxForm } from 'redux-form';

import SharingInvitationFormComponent from './sharing-invitation-form-component';

export const SharingSimpleForm = reduxForm({form: 'SIMPLE_SHARING_FORM'})(SharingInvitationFormComponent);