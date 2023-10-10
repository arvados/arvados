// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { memoize } from "lodash/fp";
import { FormDialog } from 'components/form-dialog/form-dialog';
import { CollectionProjectPickerField } from 'views-components/form-fields/collection-form-fields';
import { WithDialogProps } from 'store/dialog/with-dialog';
import { InjectedFormProps } from 'redux-form';
import { CollectionPartialMoveToSeparateCollectionsFormData } from "store/collections/collection-partial-move-actions";
import { PickerIdProp } from "store/tree-picker/picker-id";

type DialogCollectionPartialMoveProps = WithDialogProps<string> & InjectedFormProps<CollectionPartialMoveToSeparateCollectionsFormData>;

export const DialogCollectionPartialMoveToSeparateCollections = (props: DialogCollectionPartialMoveProps & PickerIdProp) =>
    <FormDialog
        dialogTitle='Move to separate collections'
        formFields={CollectionPartialMoveFields(props.pickerId)}
        submitLabel='Create collections'
        {...props}
    />;

const CollectionPartialMoveFields = memoize(
    (pickerId: string) =>
        () =>
            <>
                <CollectionProjectPickerField {...{ pickerId }} />
            </>);
