// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { connect } from "react-redux";
import { Dispatch } from "../../../node_modules/redux";
import { RootState } from "../../store/store";
import DialogProjectCreate from "../dialog-create/dialog-project-create";
import actions from "../../store/project/project-action";

const mapStateToProps = (state: RootState) => ({
    open: state.projects.creator.opened
});

const mapDispatchToProps = (dispatch: Dispatch) => ({
    handleClose: () => {
        dispatch(actions.CLOSE_PROJECT_CREATOR());
    }
});

export default connect(mapStateToProps, mapDispatchToProps)(DialogProjectCreate);
