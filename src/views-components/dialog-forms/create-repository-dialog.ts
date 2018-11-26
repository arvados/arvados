// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { compose } from "redux";
import { reduxForm } from 'redux-form';
import { withDialog } from "~/store/dialog/with-dialog";
import { createRepository, REPOSITORY_CREATE_FORM_NAME } from "~/store/repositories/repositories-actions";
import { DialogRepositoryCreate } from "~/views-components/dialog-create/dialog-repository-create";

export const CreateRepositoryDialog = compose(
    withDialog(REPOSITORY_CREATE_FORM_NAME),
    reduxForm<any>({
        form: REPOSITORY_CREATE_FORM_NAME,
        onSubmit: (repositoryName, dispatch) => {
            dispatch(createRepository(repositoryName));
        }
    })
)(DialogRepositoryCreate);