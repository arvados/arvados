// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { compose } from "redux";
import { reduxForm, InjectedFormProps } from 'redux-form';
import { withDialog, WithDialogProps } from '~/store/dialog/with-dialog';
import { CollectionPartialCopyFields } from '../collection-form-fields/collection-form-fields';
import { FormDialog } from '~/components/form-dialog/form-dialog';
import { COLLECTION_PARTIAL_COPY, doCollectionPartialCopy, CollectionPartialCopyFormData } from '~/store/collection-panel/collection-panel-files/collection-panel-files-actions';

export const CollectionPartialCopyDialog = compose(
    withDialog(COLLECTION_PARTIAL_COPY),
    reduxForm({
        form: COLLECTION_PARTIAL_COPY,
        onSubmit: (data: CollectionPartialCopyFormData, dispatch) => {
            dispatch(doCollectionPartialCopy(data));
        }
    }))((props: WithDialogProps<string> & InjectedFormProps<CollectionPartialCopyFormData>) =>
        <FormDialog
            dialogTitle='Create a collection'
            formFields={CollectionPartialCopyFields}
            submitLabel='Create a collection'
            {...props}
        />);
