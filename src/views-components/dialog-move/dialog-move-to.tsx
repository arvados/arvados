// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { memoize } from 'lodash/fp';
import { InjectedFormProps, Field } from 'redux-form';
import { WithDialogProps } from 'store/dialog/with-dialog';
import { FormDialog } from 'components/form-dialog/form-dialog';
import { ProjectTreePickerField } from 'views-components/projects-tree-picker/tree-picker-field';
import { MOVE_TO_VALIDATION } from 'validators/validators';
import { MoveToFormDialogData } from 'store/move-to-dialog/move-to-dialog';
import { PickerIdProp } from "store/tree-picker/picker-id";

export const DialogMoveTo = (props: WithDialogProps<string> & InjectedFormProps<MoveToFormDialogData> & PickerIdProp) =>
    <FormDialog
        dialogTitle='Move to'
        formFields={MoveToDialogFields(props.pickerId)}
        submitLabel='Move'
        {...props}
    />;

const MoveToDialogFields = memoize(
    (pickerId: string) => () =>
        <Field
            name="ownerUuid"
            pickerId={pickerId}
            component={ProjectTreePickerField}
            validate={MOVE_TO_VALIDATION} />);

