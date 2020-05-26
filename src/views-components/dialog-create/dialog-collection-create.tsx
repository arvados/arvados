// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { InjectedFormProps, Field } from 'redux-form';
import { WithDialogProps } from '~/store/dialog/with-dialog';
import { CollectionCreateFormDialogData } from '~/store/collections/collection-create-actions';
import { FormDialog } from '~/components/form-dialog/form-dialog';
import { CollectionNameField, CollectionDescriptionField } from '~/views-components/form-fields/collection-form-fields';
import { FileUploaderField } from '../file-uploader/file-uploader';


type DialogCollectionProps = WithDialogProps<{}> & InjectedFormProps<CollectionCreateFormDialogData>;

export const DialogCollectionCreate = (props: DialogCollectionProps) =>
    <FormDialog
        dialogTitle='New collection'
        formFields={CollectionAddFields}
        submitLabel='Create a Collection'
        {...props}
    />;

const CollectionAddFields = () => <span>
    <CollectionNameField />
    <CollectionDescriptionField />
    <Field
        name='files'
        label='Files'
        component={FileUploaderField} />
</span>;

