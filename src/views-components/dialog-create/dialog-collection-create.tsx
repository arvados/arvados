// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { InjectedFormProps } from 'redux-form';
import { WithDialogProps } from '~/store/dialog/with-dialog';
import { CollectionCreateFormDialogData } from '~/store/collections/collection-create-actions';
import { collectionUploaderActions, UploadFile } from "~/store/collections/uploader/collection-uploader-actions";
import { FormDialog } from '~/components/form-dialog/form-dialog';
import { CollectionNameField, CollectionDescriptionField } from '~/views-components/form-fields/collection-form-fields';
import { FileUpload } from '~/components/file-upload/file-upload';

// interface DialogCollectionDataProps {
//     open: boolean;
//     handleSubmit: any;
//     submitting: boolean;
//     invalid: boolean;
//     pristine: boolean;
//     files: UploadFile[];
// }

type DialogCollectionProps = WithDialogProps<{}> & InjectedFormProps<CollectionCreateFormDialogData>;

export const DialogCollectionCreate = (props: DialogCollectionProps) =>
    <FormDialog
        dialogTitle='Create a collection'
        formFields={CollectionAddFields}
        submitLabel='Create a Collection'
        {...props}
    />;

const CollectionAddFields = () => <span>
    <CollectionNameField />
    <CollectionDescriptionField />
    {/* <FileUpload
        files={this.props.files}
        disabled={busy}
        onDrop={files => this.props.dispatch(collectionUploaderActions.SET_UPLOAD_FILES(files))} /> */}
</span>;