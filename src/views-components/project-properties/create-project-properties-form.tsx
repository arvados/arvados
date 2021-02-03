// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { reduxForm, reset } from 'redux-form';
import { withStyles } from '@material-ui/core';
import {
    PROJECT_CREATE_PROPERTIES_FORM_NAME,
    addPropertyToCreateProjectForm
} from '~/store/projects/project-create-actions';
import {
    ResourcePropertiesForm,
    ResourcePropertiesFormData
} from '~/views-components/resource-properties-form/resource-properties-form';

const Form = withStyles(
    ({ spacing }) => (
        { container:
            {
                paddingTop: spacing.unit,
                margin: 0,
            }
        })
    )(ResourcePropertiesForm);

export const CreateProjectPropertiesForm = reduxForm<ResourcePropertiesFormData>({
    form: PROJECT_CREATE_PROPERTIES_FORM_NAME,
    onSubmit: (data, dispatch) => {
        dispatch(addPropertyToCreateProjectForm(data));
        dispatch(reset(PROJECT_CREATE_PROPERTIES_FORM_NAME));
    }
})(Form);