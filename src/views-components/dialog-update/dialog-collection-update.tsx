// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { InjectedFormProps } from 'redux-form';
import { WithDialogProps } from 'store/dialog/with-dialog';
import { CollectionUpdateFormDialogData } from 'store/collections/collection-update-actions';
import { FormDialog } from 'components/form-dialog/form-dialog';
import { CollectionNameField, CollectionDescriptionField } from 'views-components/form-fields/collection-form-fields';

type DialogCollectionProps = WithDialogProps<{}> & InjectedFormProps<CollectionUpdateFormDialogData>;

export const DialogCollectionUpdate = (props: DialogCollectionProps) =>
    <FormDialog
        dialogTitle='Edit Collection'
        formFields={CollectionEditFields}
        submitLabel='Save'
        {...props}
    />;

const CollectionEditFields = () => <span>
    <CollectionNameField />
    <CollectionDescriptionField />
</span>;
