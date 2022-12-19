// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { memoize } from "lodash/fp";
import { FormDialog } from 'components/form-dialog/form-dialog';
import { CollectionNameField, CollectionDescriptionField, CollectionProjectPickerField } from 'views-components/form-fields/collection-form-fields';
import { WithDialogProps } from 'store/dialog/with-dialog';
import { InjectedFormProps } from 'redux-form';
import { CollectionPartialCopyFormData } from 'store/collections/collection-partial-copy-actions';
import { PickerIdProp } from "store/tree-picker/picker-id";

type DialogCollectionPartialCopyProps = WithDialogProps<string> & InjectedFormProps<CollectionPartialCopyFormData>;

export const DialogCollectionPartialCopy = (props: DialogCollectionPartialCopyProps & PickerIdProp) =>
    <FormDialog
        dialogTitle='Create a collection'
        formFields={CollectionPartialCopyFields(props.pickerId)}
        submitLabel='Create a collection'
        {...props}
    />;

export const CollectionPartialCopyFields = memoize(
    (pickerId: string) =>
        () =>
            <>
                <CollectionNameField />
                <CollectionDescriptionField />
                <CollectionProjectPickerField {...{ pickerId }} />
            </>);
