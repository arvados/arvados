// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { reduxForm, reset } from 'redux-form';
import { PROJECT_PROPERTIES_FORM_NAME, createProjectProperty } from 'store/details-panel/details-panel-action';
import { ResourcePropertiesForm, ResourcePropertiesFormData } from 'views-components/resource-properties-form/resource-properties-form';
import { withStyles } from '@material-ui/core';
import { Dispatch } from 'redux';

const Form = withStyles(({ spacing }) => ({ container: { marginBottom: spacing.unit * 2 } }))(ResourcePropertiesForm);

export const ProjectPropertiesForm = reduxForm<ResourcePropertiesFormData>({
    form: PROJECT_PROPERTIES_FORM_NAME,
    onSubmit: (data, dispatch: Dispatch) => {
        dispatch<any>(createProjectProperty(data));
        dispatch(reset(PROJECT_PROPERTIES_FORM_NAME));
    }
})(Form);
