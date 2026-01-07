// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react'
import { compose, Dispatch } from 'redux'
import { DialogForm } from 'components/dialog-form/dialog-form'
import { connect } from 'react-redux'
import { withDialog } from 'store/dialog/with-dialog'
import { DialogCollectionNameField } from 'views-components/form-fields/collection-form-fields'
import {
	CollectionPartialCopyToNewCollectionFormData,
	copyCollectionPartialToNewCollection,
	COLLECTION_PARTIAL_COPY_FORM_NAME,
} from 'store/collections/collection-partial-copy-actions'
import { ProjectTreePickerDialogField } from 'views-components/projects-tree-picker/tree-picker-field'
import { CollectionFileSelection } from 'store/collection-panel/collection-panel-files/collection-panel-files-state'
import { WithDialogProps } from 'store/dialog/with-dialog'
import { PickerIdProp } from 'store/tree-picker/picker-id'
import { DialogTitle, DialogContent } from '@mui/material'
import { DialogRichTextField } from 'components/dialog-form/dialog-text-field'
import { REQUIRED_VALIDATION, REQUIRED_LENGTH255_VALIDATION, MAXLENGTH_524288_VALIDATION } from 'validators/validators'
import { useStateWithValidation } from 'common/useStateWithValidation'

type DialogCollectionPartialCopyProps = WithDialogProps<{
	initialFormData: CollectionPartialCopyToNewCollectionFormData
	collectionFileSelection: CollectionFileSelection
}> &
	PickerIdProp & {
		copyCollectionPartialToNewCollection: (
			fileSelection: CollectionFileSelection,
			formData: CollectionPartialCopyToNewCollectionFormData
		) => void
	}

const mapDispatch = (dispatch: Dispatch) => ({
	copyCollectionPartialToNewCollection: (
		fileSelection: CollectionFileSelection,
		formData: CollectionPartialCopyToNewCollectionFormData
	) => {
		dispatch<any>(copyCollectionPartialToNewCollection(fileSelection, formData))
	},
})

export const DialogCollectionPartialCopyToNewCollection = compose(
	withDialog(COLLECTION_PARTIAL_COPY_FORM_NAME),
	connect(null, mapDispatch)
)((props: DialogCollectionPartialCopyProps) => {
	const { open, data, pickerId } = props
	const { initialFormData, collectionFileSelection } = data
	const { name, description } = initialFormData || {}

	const [thisName, setThisName, nameErrs] = useStateWithValidation(name, REQUIRED_LENGTH255_VALIDATION, 'Name')
    const [thisDescription, setThisDescription, descriptionErrs] = useStateWithValidation(description, MAXLENGTH_524288_VALIDATION, 'Description')
    const [thisOwnerUuid, setThisOwnerUuid, ownerUuidErrs] = useStateWithValidation('', REQUIRED_VALIDATION, 'Project')

    const [formErrors, setFormErrors] = React.useState<string[]>([])

    React.useEffect(() => {
        setFormErrors([...nameErrs, ...descriptionErrs, ...ownerUuidErrs])
    }, [nameErrs, descriptionErrs, ownerUuidErrs])

	const fields = () => (
		<>
			<DialogTitle>Copy to new collection</DialogTitle>
			<DialogContent>
				<DialogCollectionNameField defaultValue={name} setValue={setThisName} />
				<DialogRichTextField
					label="Description"
					defaultValue={description}
					setValue={setThisDescription}
					validators={MAXLENGTH_524288_VALIDATION}
				/>
				<ProjectTreePickerDialogField
					pickerId={pickerId}
					setSelectedProject={setThisOwnerUuid}
				/>
			</DialogContent>
		</>
	)

	return (
		<DialogForm
			open={open}
			fields={fields()}
            submitLabel='Create Collection'
			formErrors={formErrors}
			onSubmit={(event: React.FormEvent<HTMLFormElement>) => {
				event.preventDefault()
				props.copyCollectionPartialToNewCollection(collectionFileSelection, {
					name: thisName,
					description: thisDescription,
					projectUuid: thisOwnerUuid,
				})
			}}
			closeDialog={props.closeDialog}
			clearFormValues={() => {
				setThisName('')
				setThisDescription('')
				setThisOwnerUuid('')
			}}
		/>
	)
})
