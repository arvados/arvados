// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { reduxForm, reset } from 'redux-form';
import { withStyles } from '@material-ui/core';
import {
    PROJECT_UPDATE_PROPERTIES_FORM_NAME,
    PROJECT_UPDATE_FORM_NAME
} from 'store/projects/project-update-actions';
import {
    ResourcePropertiesForm,
    ResourcePropertiesFormData
} from 'views-components/resource-properties-form/resource-properties-form';
import { addPropertyToResourceForm } from 'store/resources/resources-actions';

const Form = withStyles(
    ({ spacing }) => (
        { container:
            {
                margin: 0,
            }
        })
    )(ResourcePropertiesForm);

export const UpdateProjectPropertiesForm = reduxForm<ResourcePropertiesFormData>({
    form: PROJECT_UPDATE_PROPERTIES_FORM_NAME,
    onSubmit: (data, dispatch) => {
        dispatch(addPropertyToResourceForm(data, PROJECT_UPDATE_FORM_NAME));
        dispatch(reset(PROJECT_UPDATE_PROPERTIES_FORM_NAME));
    }
})(Form);