// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react'
import { compose, Dispatch } from 'redux'
import { connect } from 'react-redux'
import { withDialog } from 'store/dialog/with-dialog'
import { WithDialogProps } from 'store/dialog/with-dialog'
import { DialogForm } from 'components/dialog-form/dialog-form'
import { ProjectTreePickerDialogField } from 'views-components/projects-tree-picker/tree-picker-field'
import { MOVE_TO_VALIDATION } from 'validators/validators'
import { MoveToFormDialogData } from 'store/move-to-dialog/move-to-dialog'
import { PickerIdProp } from 'store/tree-picker/picker-id'
import { DialogTitle, DialogContent } from '@mui/material'
import { useStateWithValidation } from 'common/useStateWithValidation'
import { moveProject, PROJECT_MOVE_FORM_NAME } from 'store/projects/project-move-actions'

type DialogMoveProjectProps = WithDialogProps<MoveToFormDialogData> & PickerIdProp & {
    moveProject: (data: MoveToFormDialogData) => void
}

const mapDispatch = (dispatch: Dispatch) => ({
    moveProject: (data: MoveToFormDialogData) => {
        dispatch<any>(moveProject(data))
    },
})

export const DialogMoveProject = compose(
    withDialog(PROJECT_MOVE_FORM_NAME),
    connect(null, mapDispatch)
)((props: DialogMoveProjectProps) => {
    const { open, data, pickerId } = props
    const initialData = data || { ownerUuid: '' }

    const [ownerUuid, setOwnerUuid, ownerUuidErrs] = useStateWithValidation('', MOVE_TO_VALIDATION, 'Owner')

    const fields = () => (
        <>
            <DialogTitle>Move to</DialogTitle>
            <DialogContent>
                <ProjectTreePickerDialogField
                    pickerId={pickerId}
                    setSelectedProject={setOwnerUuid}
                />
            </DialogContent>
        </>
    )

    return (
        <DialogForm
            open={open}
            fields={fields()}
            submitLabel='Move'
            formErrors={ownerUuidErrs}
            onSubmit={(event: React.FormEvent<HTMLFormElement>) => {
                event.preventDefault()
                props.moveProject({
                    ownerUuid: ownerUuid,
                    uuid: initialData.uuid || '',
                    name: initialData.name || '',
                })
            }}
            closeDialog={props.closeDialog}
            clearFormValues={() => {
                setOwnerUuid('')
            }}
        />
    )
})

