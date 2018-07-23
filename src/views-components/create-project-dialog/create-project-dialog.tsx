// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { connect } from "react-redux";
import { Dispatch } from "redux";
import { SubmissionError } from "redux-form";

import { RootState } from "../../store/store";
import  DialogProjectCreate from "../dialog-create/dialog-project-create";
import { projectActions, createProject, getProjectList } from "../../store/project/project-action";
import { dataExplorerActions } from "../../store/data-explorer/data-explorer-action";
import { PROJECT_PANEL_ID } from "../../views/project-panel/project-panel";

const mapStateToProps = (state: RootState) => ({
    open: state.projects.creator.opened
});

const addProject = (data: { name: string, description: string }) =>
    (dispatch: Dispatch, getState: () => RootState) => {
        const { ownerUuid } = getState().projects.creator;
        return dispatch<any>(createProject(data)).then(() => {
            dispatch(dataExplorerActions.REQUEST_ITEMS({ id: PROJECT_PANEL_ID }));
            dispatch<any>(getProjectList(ownerUuid));
        });
    };

const mapDispatchToProps = (dispatch: Dispatch) => ({
    handleClose: () => {
        dispatch(projectActions.CLOSE_PROJECT_CREATOR());
    },
    onSubmit: (data: { name: string, description: string }) => {
        return dispatch<any>(addProject(data))
            .catch((e: any) => {
                throw new SubmissionError({ name: e.errors.join("").includes("UniqueViolation") ? "Project with this name already exists." : "" });
            });
    }
});

export const CreateProjectDialog = connect(mapStateToProps, mapDispatchToProps)(DialogProjectCreate);
