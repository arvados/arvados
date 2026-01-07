// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react'
import { compose, Dispatch } from 'redux'
import { connect } from 'react-redux'
import { DialogTitle, DialogContent } from '@mui/material'
import { WithDialogProps, withDialog } from 'store/dialog/with-dialog'
import { ProjectTreePickerDialogField } from 'views-components/projects-tree-picker/tree-picker-field'
import { COPY_NAME_VALIDATION, REQUIRED_VALIDATION } from 'validators/validators'
import { CopyFormDialogData } from 'store/copy-dialog/copy-dialog'
import { PickerIdProp } from 'store/tree-picker/picker-id'
import { copyCollectionRunner } from 'store/workbench/workbench-actions'
import { COLLECTION_COPY_FORM_NAME } from 'store/collections/collection-copy-actions'
import { DialogForm } from 'components/dialog-form/dialog-form'
import { DialogTextField } from 'components/dialog-form/dialog-text-field'
import { useStateWithValidation } from 'common/useStateWithValidation'

type CopyDialogProps = WithDialogProps<CopyFormDialogData> &
	PickerIdProp & {
		selectedCollectionUuid: string | undefined
		copyCollection: (data: CopyFormDialogData) => void
	}

const mapDispatchToProps = (dispatch: Dispatch) => ({
	copyCollection: (data: CopyFormDialogData) => dispatch<any>(copyCollectionRunner(data)),
})

export const CopyCollectionDialog = compose(
	withDialog(COLLECTION_COPY_FORM_NAME),
	connect(null, mapDispatchToProps)
)((props: CopyDialogProps) => {
	const { open, data, pickerId } = props
	const [nameVal, setNameVal, nameErrs] = useStateWithValidation(data.name || '', COPY_NAME_VALIDATION, 'Name')
	const [selectedProjectUuid, setSelectedProjectUuid, selectedProjectErrs] = useStateWithValidation('', REQUIRED_VALIDATION, 'Project')
	const [formErrors, setFormErrors] = React.useState<string[]>([])

	React.useEffect(() => {
		setFormErrors([...selectedProjectErrs, ...nameErrs])
	}, [nameVal, selectedProjectUuid])

	const fields = () => (
		<>
			{data.isSingleResource ? (
				<>
					<DialogTitle>Make a copy</DialogTitle>
					<DialogContent>
						<DialogTextField
                            label="Enter a new name for the copy"
							defaultValue={data.name}
							setValue={setNameVal}
							validators={COPY_NAME_VALIDATION}
						/>
					</DialogContent>
				</>
			) : (
				<DialogTitle>Make copies</DialogTitle>
			)}
			<ProjectTreePickerDialogField
				pickerId={pickerId}
				setSelectedProject={setSelectedProjectUuid}
			/>
		</>
	)

	return (
		<DialogForm
			open={open}
			fields={fields()}
			submitLabel='Copy Collection'
			onSubmit={(event: React.FormEvent<HTMLFormElement>) => {
				event.preventDefault()
				props.copyCollection({
					name: nameVal,
					uuid: data.uuid,
					ownerUuid: selectedProjectUuid,
				})
			}}
			formErrors={formErrors}
			closeDialog={props.closeDialog}
			clearFormValues={() => {
				setSelectedProjectUuid('')
				setNameVal('')
			}}
		/>
	)
})
