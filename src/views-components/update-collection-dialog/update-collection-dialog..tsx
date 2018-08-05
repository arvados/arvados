// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { connect } from "react-redux";
import { Dispatch } from "redux";
import { SubmissionError } from "redux-form";
import { RootState } from "../../store/store";
import { snackbarActions } from "../../store/snackbar/snackbar-actions";
import { collectionUpdatorActions, updateCollection } from "../../store/collections/updater/collection-updater-action";
import { dataExplorerActions } from "../../store/data-explorer/data-explorer-action";
import { PROJECT_PANEL_ID } from "../../views/project-panel/project-panel";
import { DialogCollectionUpdate } from "../dialog-update/dialog-collection-update";

const mapStateToProps = (state: RootState) => ({
    open: state.collections.updator.opened
});

const mapDispatchToProps = (dispatch: Dispatch) => ({
    handleClose: () => {
        dispatch(collectionUpdatorActions.CLOSE_COLLECTION_UPDATER());
    },
    onSubmit: (data: { name: string, description: string }) => {
        return dispatch<any>(editCollection(data))
            .catch((e: any) => {
                if(e.errors) {
                    throw new SubmissionError({ name: e.errors.join("").includes("UniqueViolation") ? "Collection with this name already exists." : "" });
                }
            });
    }
});

const editCollection = (data: { name: string, description: string }) =>
    (dispatch: Dispatch) => {
        return dispatch<any>(updateCollection(data)).then(() => {
            dispatch(snackbarActions.OPEN_SNACKBAR({
                message: "Collection has been successfully updated.",
                hideDuration: 2000
            }));
            dispatch(dataExplorerActions.REQUEST_ITEMS({ id: PROJECT_PANEL_ID }));
        });
    };

export const UpdateCollectionDialog = connect(mapStateToProps, mapDispatchToProps)(DialogCollectionUpdate);
