// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { reduxForm, reset } from 'redux-form';
import { withStyles } from '@material-ui/core';
import {
    PROJECT_UPDATE_PROPERTIES_FORM_NAME,
    addPropertyToUpdateProjectForm
} from 'store/projects/project-update-actions';
import {
    ResourcePropertiesForm,
    ResourcePropertiesFormData
} from 'views-components/resource-properties-form/resource-properties-form';

const Form = withStyles(
    ({ spacing }) => (
        { container:
            {
                paddingTop: spacing.unit,
                margin: 0,
            }
        })
    )(ResourcePropertiesForm);

export const UpdateProjectPropertiesForm = reduxForm<ResourcePropertiesFormData>({
    form: PROJECT_UPDATE_PROPERTIES_FORM_NAME,
    onSubmit: (data, dispatch) => {
        dispatch(addPropertyToUpdateProjectForm(data));
        dispatch(reset(PROJECT_UPDATE_PROPERTIES_FORM_NAME));
    }
})(Form);