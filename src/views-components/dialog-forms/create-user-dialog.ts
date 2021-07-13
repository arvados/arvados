// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { compose } from "redux";
import { reduxForm } from 'redux-form';
import { withDialog } from "store/dialog/with-dialog";
import { USER_CREATE_FORM_NAME, createUser, UserCreateFormDialogData } from "store/users/users-actions";
import { UserRepositoryCreate } from "views-components/dialog-create/dialog-user-create";

export const CreateUserDialog = compose(
    withDialog(USER_CREATE_FORM_NAME),
    reduxForm<UserCreateFormDialogData>({
        form: USER_CREATE_FORM_NAME,
        onSubmit: (data, dispatch) => {
            dispatch(createUser(data));
        }
    })
)(UserRepositoryCreate);