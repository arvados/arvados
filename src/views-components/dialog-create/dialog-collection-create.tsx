// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { InjectedFormProps, Field } from 'redux-form';
import { WithDialogProps } from 'store/dialog/with-dialog';
import { CollectionCreateFormDialogData } from 'store/collections/collection-create-actions';
import { FormDialog } from 'components/form-dialog/form-dialog';
import {
    CollectionNameField,
    CollectionDescriptionField,
    CollectionStorageClassesField
} from 'views-components/form-fields/collection-form-fields';
import { FileUploaderField } from '../file-uploader/file-uploader';
import { ResourceParentField } from '../form-fields/resource-form-fields';
import { CreateCollectionPropertiesList } from 'views-components/collection-properties/create-collection-properties-list';
import { CreateCollectionPropertiesForm } from 'views-components/collection-properties/create-collection-properties-form';
import { FormGroup, FormLabel } from '@material-ui/core';

type DialogCollectionProps = WithDialogProps<{}> & InjectedFormProps<CollectionCreateFormDialogData>;

export const DialogCollectionCreate = (props: DialogCollectionProps) =>
    <FormDialog
        dialogTitle='New collection'
        formFields={CollectionAddFields}
        submitLabel='Create a Collection'
        {...props}
    />;

const CollectionAddFields = () => <span>
    <ResourceParentField />
    <CollectionNameField />
    <CollectionDescriptionField />
    <FormLabel>Properties</FormLabel>
    <FormGroup>
        <CreateCollectionPropertiesForm />
        <CreateCollectionPropertiesList />
    </FormGroup>
    <CollectionStorageClassesField defaultClasses={['default']} />
    <Field
        name='files'
        label='Files'
        component={FileUploaderField} />
</span>;

