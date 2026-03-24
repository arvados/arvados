// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { compose, Dispatch } from "redux";
import { connect } from "react-redux";
import { withDialog } from 'store/dialog/with-dialog';
import { DialogForm } from "components/dialog-form/dialog-form";
import { WithDialogProps } from 'store/dialog/with-dialog';
import { DialogTitle } from "@mui/material";
import { CollectionFileSelection } from 'store/collection-panel/collection-panel-files/collection-panel-files-state';
import { CollectionPartialCopyToExistingCollectionFormData, copyCollectionPartialToExistingCollection, COLLECTION_PARTIAL_COPY_TO_SELECTED_COLLECTION } from 'store/collections/collection-partial-copy-actions';
import { PickerIdProp } from "store/tree-picker/picker-id";
import { DirectoryTreePickerDialogField } from "views-components/projects-tree-picker/tree-picker-field";
import { useStateWithValidation } from "common/useStateWithValidation";
import { FILE_OPS_LOCATION_VALIDATION } from "validators/validators";
import { FileOperationLocation } from "store/tree-picker/tree-picker-actions";

type DialogCollectionPartialCopyProps = WithDialogProps<{ initialFormData: CollectionPartialCopyToExistingCollectionFormData, collectionFileSelection: CollectionFileSelection }> & {
} & PickerIdProp & {
    copyCollectionPartialToExistingCollection: (
        fileSelection: CollectionFileSelection,
        formData: CollectionPartialCopyToExistingCollectionFormData
    ) => void
}

const mapDispatchToProps = (dispatch: Dispatch) => ({
    copyCollectionPartialToExistingCollection: (
        fileSelection: CollectionFileSelection,
        formData: CollectionPartialCopyToExistingCollectionFormData
    ) => {
        dispatch<any>(copyCollectionPartialToExistingCollection(fileSelection, formData));
    },
});

export const DialogCollectionPartialCopyToExistingCollection = compose(
    withDialog(COLLECTION_PARTIAL_COPY_TO_SELECTED_COLLECTION),
    connect(null, mapDispatchToProps)
)((props: DialogCollectionPartialCopyProps & PickerIdProp) => {
    const { open, data, copyCollectionPartialToExistingCollection } = props;
    const { collectionFileSelection, initialFormData } = data;
    const [selectedDestination, setSelectedDestination, errs] = useStateWithValidation<FileOperationLocation | null>(null, FILE_OPS_LOCATION_VALIDATION, 'Collection');

    const handleDirectoryChange = (res: FileOperationLocation) => {
        setSelectedDestination(res);
    }

    const fields = () => (
        <>
            <DialogTitle>Copy Selected Files to Existing Collection</DialogTitle>
            <DirectoryTreePickerDialogField
                currentUuids={initialFormData?.destination.uuid ? [initialFormData.destination.uuid] : []}
                pickerId={props.pickerId}
                handleDirectoryChange={handleDirectoryChange}
            />
        </>
    );


    return <DialogForm
                open={open}
                fields={fields()}
                submitLabel="Copy Files"
                onSubmit={(ev)=>{
                    ev.preventDefault();
                    if (!!selectedDestination) {
                        copyCollectionPartialToExistingCollection(collectionFileSelection, { destination: selectedDestination });
                    }
                }}
                formErrors={errs}
                closeDialog={props.closeDialog}
                clearFormValues={()=> {
                    setSelectedDestination(null);
                }}
            />;
});