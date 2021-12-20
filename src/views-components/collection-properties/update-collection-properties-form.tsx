// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { reduxForm, reset } from 'redux-form';
import { withStyles } from '@material-ui/core';
import {
    COLLECTION_UPDATE_FORM_NAME,
    COLLECTION_UPDATE_PROPERTIES_FORM_NAME
} from 'store/collections/collection-update-actions';
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

export const UpdateCollectionPropertiesForm = reduxForm<ResourcePropertiesFormData>({
    form: COLLECTION_UPDATE_PROPERTIES_FORM_NAME,
    onSubmit: (data, dispatch) => {
        dispatch(addPropertyToResourceForm(data, COLLECTION_UPDATE_FORM_NAME));
        dispatch(reset(COLLECTION_UPDATE_PROPERTIES_FORM_NAME));
    }
})(Form);