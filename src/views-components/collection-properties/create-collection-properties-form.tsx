// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { reduxForm, change } from 'redux-form';
import { withStyles } from '@material-ui/core';
import {
    COLLECTION_CREATE_PROPERTIES_FORM_NAME,
    COLLECTION_CREATE_FORM_NAME
} from 'store/collections/collection-create-actions';
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

export const CreateCollectionPropertiesForm = reduxForm<ResourcePropertiesFormData>({
    form: COLLECTION_CREATE_PROPERTIES_FORM_NAME,
    onSubmit: (data, dispatch) => {
        dispatch(addPropertyToResourceForm(data, COLLECTION_CREATE_FORM_NAME));
        dispatch(change(COLLECTION_CREATE_PROPERTIES_FORM_NAME, PROPERTY_VALUE_FIELD_NAME, ''));
    }
})(Form);