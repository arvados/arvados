// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { reduxForm, reset } from 'redux-form';
import { RESOURCE_PROPERTIES_FORM_NAME } from 'store/details-panel/details-panel-action';
import { ResourcePropertiesForm, ResourcePropertiesFormData } from 'views-components/resource-properties-form/resource-properties-form';
import { withStyles } from '@material-ui/core';
import { Dispatch } from 'redux';
import { createResourceProperty } from 'store/resources/resources-actions';

const Form = withStyles(({ spacing }) => ({ container: { marginBottom: spacing.unit * 2 } }))(ResourcePropertiesForm);

export const ResourcePropertiesDialogForm = reduxForm<ResourcePropertiesFormData, {uuid: string}>({
    form: RESOURCE_PROPERTIES_FORM_NAME,
    onSubmit: (data, dispatch: Dispatch) => {
        dispatch<any>(createResourceProperty(data));
        dispatch(reset(RESOURCE_PROPERTIES_FORM_NAME));
    }
})(Form);
