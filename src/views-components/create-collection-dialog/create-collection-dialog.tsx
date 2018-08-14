// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { connect } from "react-redux";
import { Dispatch } from "redux";
import { SubmissionError } from "redux-form";

import { RootState } from "~/store/store";
import { DialogCollectionCreate } from "../dialog-create/dialog-collection-create";
import { collectionCreateActions, createCollection } from "~/store/collections/creator/collection-creator-action";
import { snackbarActions } from "~/store/snackbar/snackbar-actions";
import { UploadFile } from "~/store/collections/uploader/collection-uploader-actions";
import { projectPanelActions } from "~/store/project-panel/project-panel-action";

const mapStateToProps = (state: RootState) => ({
    open: state.collections.creator.opened
});

const mapDispatchToProps = (dispatch: Dispatch) => ({
    handleClose: () => {
        dispatch(collectionCreateActions.CLOSE_COLLECTION_CREATOR());
    },
    onSubmit: (data: { name: string, description: string }, files: UploadFile[]) => {
        return dispatch<any>(addCollection(data, files.map(f => f.file)))
            .catch((e: any) => {
                throw new SubmissionError({ name: e.errors.join("").includes("UniqueViolation") ? "Collection with this name already exists." : "" });
            });
    }
});

const addCollection = (data: { name: string, description: string }, files: File[]) =>
    (dispatch: Dispatch) => {
        return dispatch<any>(createCollection(data, files)).then(() => {
            dispatch(snackbarActions.OPEN_SNACKBAR({
                message: "Collection has been successfully created.",
                hideDuration: 2000
            }));
            dispatch(projectPanelActions.REQUEST_ITEMS());
        });
    };

export const CreateCollectionDialog = connect(mapStateToProps, mapDispatchToProps)(DialogCollectionCreate);

