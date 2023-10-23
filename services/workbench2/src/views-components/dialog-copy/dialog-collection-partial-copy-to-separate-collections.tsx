// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { memoize } from "lodash/fp";
import { FormDialog } from 'components/form-dialog/form-dialog';
import { CollectionProjectPickerField } from 'views-components/form-fields/collection-form-fields';
import { WithDialogProps } from 'store/dialog/with-dialog';
import { InjectedFormProps } from 'redux-form';
import { CollectionPartialCopyToSeparateCollectionsFormData } from 'store/collections/collection-partial-copy-actions';
import { PickerIdProp } from "store/tree-picker/picker-id";

type DialogCollectionPartialCopyProps = WithDialogProps<string> & InjectedFormProps<CollectionPartialCopyToSeparateCollectionsFormData>;

export const DialogCollectionPartialCopyToSeparateCollection = (props: DialogCollectionPartialCopyProps & PickerIdProp) =>
    <FormDialog
        dialogTitle='Copy to separate collections'
        formFields={CollectionPartialCopyFields(props.pickerId)}
        submitLabel='Create collections'
        {...props}
    />;

const CollectionPartialCopyFields = memoize(
    (pickerId: string) =>
        () =>
            <>
                <CollectionProjectPickerField {...{ pickerId }} />
            </>);
