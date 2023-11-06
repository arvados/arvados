// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { reduxForm, change } from 'redux-form';
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
import { PROPERTY_VALUE_FIELD_NAME } from 'views-components/resource-properties-form/property-value-field';

const Form = withStyles(
    ({ spacing }) => (
        { container:
            {
                margin: 0,
            }
        })
    )(ResourcePropertiesForm);

export const CreateProjectPropertiesForm = reduxForm<ResourcePropertiesFormData>({
    form: PROJECT_CREATE_PROPERTIES_FORM_NAME,
    onSubmit: (data, dispatch) => {
        dispatch(addPropertyToResourceForm(data, PROJECT_CREATE_FORM_NAME));
        dispatch(change(PROJECT_CREATE_PROPERTIES_FORM_NAME, PROPERTY_VALUE_FIELD_NAME, ''));
    }
})(Form);