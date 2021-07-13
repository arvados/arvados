// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { reduxForm, reset } from 'redux-form';
import { createCollectionTag, COLLECTION_TAG_FORM_NAME } from 'store/collection-panel/collection-panel-action';
import { ResourcePropertiesForm, ResourcePropertiesFormData } from 'views-components/resource-properties-form/resource-properties-form';
import { withStyles } from '@material-ui/core';
import { Dispatch } from 'redux';

const Form = withStyles(({ spacing }) => ({ container: { marginBottom: spacing.unit * 2 } }))(ResourcePropertiesForm);

export const CollectionTagForm = reduxForm<ResourcePropertiesFormData>({
    form: COLLECTION_TAG_FORM_NAME,
    onSubmit: (data, dispatch: Dispatch) => {
        dispatch<any>(createCollectionTag(data));
        dispatch(reset(COLLECTION_TAG_FORM_NAME));
    }
})(Form);
