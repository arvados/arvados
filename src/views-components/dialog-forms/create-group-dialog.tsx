// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { compose } from "redux";
import { reduxForm, InjectedFormProps } from 'redux-form';
import { withDialog, WithDialogProps } from "~/store/dialog/with-dialog";
import { FormDialog } from '~/components/form-dialog/form-dialog';
import { CREATE_GROUP_DIALOG, CREATE_GROUP_FORM } from '~/store/groups-panel/groups-panel-actions';

export const CreateGroupDialog = compose(
    withDialog(CREATE_GROUP_DIALOG),
    reduxForm<{}>({
        form: CREATE_GROUP_FORM,
        onSubmit: (data, dispatch) => { return; }
    })
)(
    (props: CreateGroupDialogComponentProps) =>
        <FormDialog
            dialogTitle='Create a group'
            formFields={CreateGroupFormFields}
            submitLabel='Create'
            {...props}
        />
);

type CreateGroupDialogComponentProps = WithDialogProps<{}> & InjectedFormProps<{}>;

const CreateGroupFormFields = (props: CreateGroupDialogComponentProps) => <span>
    CreateGroupFormFields
</span>;
