// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { memoize } from "lodash/fp";
import { FormDialog } from 'components/form-dialog/form-dialog';
import { WithDialogProps } from 'store/dialog/with-dialog';
import { InjectedFormProps } from 'redux-form';
import { CollectionPartialCopyToSelectedCollectionFormData } from 'store/collections/collection-partial-copy-actions';
import { PickerIdProp } from "store/tree-picker/picker-id";
import { CollectionPickerField } from 'views-components/form-fields/collection-form-fields';

type DialogCollectionPartialCopyProps = WithDialogProps<string> & InjectedFormProps<CollectionPartialCopyToSelectedCollectionFormData>;

export const DialogCollectionPartialCopyToSelectedCollection = (props: DialogCollectionPartialCopyProps & PickerIdProp) =>
    <FormDialog
        dialogTitle='Choose collection'
        formFields={CollectionPartialCopyFields(props.pickerId)}
        submitLabel='Copy files'
        {...props}
    />;

export const CollectionPartialCopyFields = memoize(
    (pickerId: string) =>
        () =>
            <div>
                <CollectionPickerField {...{ pickerId }}/>
            </div>);
