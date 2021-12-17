// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { reduxForm, reset } from 'redux-form';
import { withStyles } from '@material-ui/core';
import {
    PROJECT_CREATE_PROPERTIES_FORM_NAME,
    PROJECT_CREATE_FORM_NAME
} from 'store/projects/project-create-actions';
import {
    ResourcePropertiesForm,
    ResourcePropertiesFormData
} from 'views-components/resource-properties-form/resource-properties-form';
import { addPropertyToResourceForm } from 'store/resources/resources-actions';

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
        dispatch(addPropertyToResourceForm(data, PROJECT_CREATE_FORM_NAME));
        dispatch(reset(PROJECT_CREATE_PROPERTIES_FORM_NAME));
    }
})(Form);