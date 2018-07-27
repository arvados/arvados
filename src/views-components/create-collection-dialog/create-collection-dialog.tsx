// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { connect } from "react-redux";
import { Dispatch } from "redux";
import { SubmissionError } from "redux-form";

import { RootState } from "../../store/store";
import { DialogCollectionCreate } from "../dialog-create/dialog-collection-create";
import { collectionCreateActions, createCollection } from "../../store/collections/creator/collection-creator-action";
import { dataExplorerActions } from "../../store/data-explorer/data-explorer-action";
import { PROJECT_PANEL_ID } from "../../views/project-panel/project-panel";

const mapStateToProps = (state: RootState) => ({
    open: state.collectionCreation.creator.opened
});

const mapDispatchToProps = (dispatch: Dispatch) => ({
    handleClose: () => {
        dispatch(collectionCreateActions.CLOSE_COLLECTION_CREATOR());
    },
    onSubmit: (data: { name: string, description: string }) => {
        return dispatch<any>(addCollection(data))
            .catch((e: any) => {
                throw new SubmissionError({ name: e.errors.join("").includes("UniqueViolation") ? "Collection with this name already exists." : "" });
            });
    }
});

const addCollection = (data: { name: string, description: string }) =>
    (dispatch: Dispatch) => {
        return dispatch<any>(createCollection(data)).then(() => {
            dispatch(dataExplorerActions.REQUEST_ITEMS({ id: PROJECT_PANEL_ID }));
        });
    };

export const CreateCollectionDialog = connect(mapStateToProps, mapDispatchToProps)(DialogCollectionCreate);

