// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { InjectedFormProps, Field } from 'redux-form';
import { WithDialogProps } from '~/store/dialog/with-dialog';
import { CollectionUpdateFormDialogData } from '~/store/collections/collection-updater-actions';
import { COLLECTION_NAME_VALIDATION, COLLECTION_DESCRIPTION_VALIDATION } from '~/validators/validators';
import { TextField } from '~/components/text-field/text-field';
import { FormDialog } from '~/components/form-dialog/form-dialog';

type DialogCollectionProps = WithDialogProps<{}> & InjectedFormProps<CollectionUpdateFormDialogData>;

export const DialogCollectionUpdate = (props: DialogCollectionProps) =>
    <FormDialog
        dialogTitle='Edit Collection'
        formFields={CollectionEditFields}
        submitLabel='Save'
        {...props}
    />;

const CollectionEditFields = () => <span>
    <Field
        name='name'
        component={TextField}
        validate={COLLECTION_NAME_VALIDATION}
        label="Collection Name" />
    <Field
        name="description"
        component={TextField}
        validate={COLLECTION_DESCRIPTION_VALIDATION} 
        label="Description - optional" />
</span>;
