// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { reduxForm, reset } from 'redux-form';
import { PROJECT_PROPERTIES_FORM_NAME, createProjectProperty } from '~/store/details-panel/details-panel-action';
import { ResourcePropertiesForm, ResourcePropertiesFormData } from '~/views-components/resource-properties-form/resource-properties-form';

export const ProjectPropertiesForm = reduxForm<ResourcePropertiesFormData>({
    form: PROJECT_PROPERTIES_FORM_NAME,
    onSubmit: (data, dispatch) => {
        dispatch<any>(createProjectProperty(data));
        dispatch(reset(PROJECT_PROPERTIES_FORM_NAME));
    }
})(ResourcePropertiesForm);
