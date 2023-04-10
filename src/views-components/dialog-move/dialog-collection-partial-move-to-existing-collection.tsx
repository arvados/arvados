// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { memoize } from "lodash/fp";
import { FormDialog } from 'components/form-dialog/form-dialog';
import { WithDialogProps } from 'store/dialog/with-dialog';
import { InjectedFormProps } from 'redux-form';
import { CollectionPartialMoveToExistingCollectionFormData } from "store/collections/collection-partial-move-actions";
import { PickerIdProp } from "store/tree-picker/picker-id";
import { CollectionPickerField } from 'views-components/form-fields/collection-form-fields';

type DialogCollectionPartialMoveProps = WithDialogProps<string> & InjectedFormProps<CollectionPartialMoveToExistingCollectionFormData>;

export const DialogCollectionPartialMoveToExistingCollection = (props: DialogCollectionPartialMoveProps & PickerIdProp) =>
    <FormDialog
        dialogTitle='Move to existing collection'
        formFields={CollectionPartialMoveFields(props.pickerId)}
        submitLabel='Move files'
        {...props}
    />;

const CollectionPartialMoveFields = memoize(
    (pickerId: string) =>
        () =>
            <>
                <CollectionPickerField {...{ pickerId }}/>
            </>);
