// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { memoize } from "lodash/fp";
import { FormDialog } from 'components/form-dialog/form-dialog';
import { WithDialogProps } from 'store/dialog/with-dialog';
import { InjectedFormProps } from 'redux-form';
import { CollectionPartialCopyToExistingCollectionFormData } from 'store/collections/collection-partial-copy-actions';
import { PickerIdProp } from "store/tree-picker/picker-id";
import { DirectoryPickerField } from 'views-components/form-fields/collection-form-fields';

type DialogCollectionPartialCopyProps = WithDialogProps<string> & InjectedFormProps<CollectionPartialCopyToExistingCollectionFormData>;

export const DialogCollectionPartialCopyToExistingCollection = (props: DialogCollectionPartialCopyProps & PickerIdProp) =>
    <FormDialog
        dialogTitle='Copy to existing collection'
        formFields={CollectionPartialCopyFields(props.pickerId)}
        submitLabel='Copy files'
        enableWhenPristine
        {...props}
    />;

const CollectionPartialCopyFields = memoize(
    (pickerId: string) =>
        () =>
            <>
                <DirectoryPickerField {...{ pickerId }}/>
            </>);
