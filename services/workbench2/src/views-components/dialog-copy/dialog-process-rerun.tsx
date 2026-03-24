// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { Dispatch, compose } from 'redux';
import { connect } from 'react-redux';
import { WithDialogProps, withDialog } from 'store/dialog/with-dialog';
import { DialogForm } from 'components/dialog-form/dialog-form';
import { ProjectTreePickerDialogField } from 'views-components/projects-tree-picker/tree-picker-field';
import { COPY_NAME_VALIDATION, COPY_FILE_VALIDATION } from 'validators/validators';
import { CopyFormDialogData } from 'store/copy-dialog/copy-dialog';
import { PickerIdProp } from 'store/tree-picker/picker-id';
import { copyProcessRunner } from 'store/workbench/workbench-actions';
import { DialogTitle, DialogContent } from '@mui/material';
import { DialogTextField } from 'components/dialog-form/dialog-text-field';
import { useStateWithValidation } from 'common/useStateWithValidation';
import { PROCESS_COPY_FORM_NAME } from 'store/processes/process-copy-actions';

type ProcessRerunFormDialogProps = WithDialogProps<CopyFormDialogData> & PickerIdProp & {
    copyProcess: (data: CopyFormDialogData) => void
};

const mapDispatch = (dispatch: Dispatch) => ({
    copyProcess: (data: CopyFormDialogData) => {
        dispatch<any>(copyProcessRunner(data));
    }
});

export const DialogProcessRerun = compose(
    withDialog(PROCESS_COPY_FORM_NAME),
    connect(null, mapDispatch))
    ((props: ProcessRerunFormDialogProps) => {
        const { open, data, pickerId, copyProcess } = props;
        const [name, setName, nameErrs] = useStateWithValidation(data.name || '', COPY_NAME_VALIDATION, 'Name');
        const [ownerUuid, setOwnerUuid, ownerUuidErrs] = useStateWithValidation(data.ownerUuid || '', COPY_FILE_VALIDATION, 'Project');
        const [formErrors, setFormErrors] = React.useState<string[]>([]);

        React.useEffect(() => {
            setFormErrors([...nameErrs, ...ownerUuidErrs]);
        }, [nameErrs, ownerUuidErrs]);

        const fields = () => (
            <>
                <DialogTitle>Choose location for re-run</DialogTitle>
                <DialogContent>
                    <DialogTextField
                        label="Enter a new name for the copy"
                        defaultValue={data.name || ''}
                        setValue={setName}
                        validators={COPY_NAME_VALIDATION}
                    />
                    <ProjectTreePickerDialogField
                        pickerId={pickerId}
                        setSelectedProject={setOwnerUuid}
                    />
                </DialogContent>
            </>
        );

        return (
            <DialogForm
                open={open}
                fields={fields()}
                submitLabel="Copy"
                formErrors={formErrors}
                onSubmit={(event: React.FormEvent<HTMLFormElement>) => {
                    event.preventDefault();
                    copyProcess({
                        name,
                        ownerUuid,
                        uuid: data?.uuid || ''});
                }}
                closeDialog={props.closeDialog}
                clearFormValues={() => {
                    setName('');
                    setOwnerUuid('');
                }}
            />
        );
    }
);
