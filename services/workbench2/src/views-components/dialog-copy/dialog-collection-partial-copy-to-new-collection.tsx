// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { memoize } from "lodash/fp";
import { FormDialog } from 'components/form-dialog/form-dialog';
import { CollectionNameField, CollectionDescriptionField, CollectionProjectPickerField } from 'views-components/form-fields/collection-form-fields';
import { WithDialogProps } from 'store/dialog/with-dialog';
import { InjectedFormProps } from 'redux-form';
import { CollectionPartialCopyToNewCollectionFormData } from 'store/collections/collection-partial-copy-actions';
import { PickerIdProp } from "store/tree-picker/picker-id";

type DialogCollectionPartialCopyProps = WithDialogProps<string> & InjectedFormProps<CollectionPartialCopyToNewCollectionFormData>;

export const DialogCollectionPartialCopyToNewCollection = (props: DialogCollectionPartialCopyProps & PickerIdProp) =>
    <FormDialog
        dialogTitle='Copy to new collection'
        formFields={CollectionPartialCopyFields(props.pickerId)}
        submitLabel='Create collection'
        {...props}
    />;

const CollectionPartialCopyFields = memoize(
    (pickerId: string) =>
        () =>
            <>
                <CollectionNameField />
                <CollectionDescriptionField />
                <CollectionProjectPickerField {...{ pickerId }} />
            </>);
