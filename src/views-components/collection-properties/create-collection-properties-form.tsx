// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { reduxForm, reset } from 'redux-form';
import { withStyles } from '@material-ui/core';
import {
    COLLECTION_CREATE_PROPERTIES_FORM_NAME,
    addPropertyToCreateCollectionForm
} from 'store/collections/collection-create-actions';
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

export const CreateCollectionPropertiesForm = reduxForm<ResourcePropertiesFormData>({
    form: COLLECTION_CREATE_PROPERTIES_FORM_NAME,
    onSubmit: (data, dispatch) => {
        dispatch(addPropertyToCreateCollectionForm(data));
        dispatch(reset(COLLECTION_CREATE_PROPERTIES_FORM_NAME));
    }
})(Form);