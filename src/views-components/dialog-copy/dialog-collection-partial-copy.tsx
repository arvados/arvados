// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { InjectedFormProps, Field } from 'redux-form';
import { WithDialogProps } from '~/store/dialog/with-dialog';
import { CollectionPartialCopyFormData } from '~/store/collection-panel/collection-panel-files/collection-panel-files-actions';
import { FormDialog } from '~/components/form-dialog/form-dialog';
import { CollectionPartialCopyFields } from '~/views-components/form-fields/collection-form-fields';

type CopyFormDialogProps = WithDialogProps<string> & InjectedFormProps<CollectionPartialCopyFormData>;

export const DialogCollectionPartialCopy = (props: CopyFormDialogProps) =>
    <FormDialog
        dialogTitle='Create a collection'
        formFields={CollectionPartialCopyFields}
        submitLabel='Create a collection'
        {...props}
    />;