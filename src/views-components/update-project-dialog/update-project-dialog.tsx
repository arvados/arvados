// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { connect } from "react-redux";
import { Dispatch } from "redux";
import { SubmissionError } from "redux-form";
import { RootState } from "~/store/store";
import { snackbarActions } from "~/store/snackbar/snackbar-actions";
import { DialogProjectUpdate } from "../dialog-update/dialog-project-update";
import { projectActions, updateProject } from "~/store/project/project-action";

const mapStateToProps = (state: RootState) => ({
    open: state.projects.updater.opened
});

const mapDispatchToProps = (dispatch: Dispatch) => ({
    handleClose: () => {
        dispatch(projectActions.CLOSE_PROJECT_UPDATER());
    },
    onSubmit: (data: { name: string, description: string }) => {
        return dispatch<any>(editProject(data))
            .catch((e: any) => {
                if (e.errors) {
                    throw new SubmissionError({ name: e.errors.join("").includes("UniqueViolation") ? "CProject with this name already exists." : "" });
                }
            });
    }
});

const editProject = (data: { name: string, description: string }) =>
    (dispatch: Dispatch, getState: () => RootState) => {
        const { uuid } = getState().projects.updater;
        return dispatch<any>(updateProject(data)).then(() => {
            dispatch(snackbarActions.OPEN_SNACKBAR({
                message: "Project has been successfully updated.",
                hideDuration: 2000
            }));
        });
    };

export const UpdateProjectDialog = connect(mapStateToProps, mapDispatchToProps)(DialogProjectUpdate);
