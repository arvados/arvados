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
	CollectionPartialMoveToNewCollectionFormData,
	moveCollectionPartialToNewCollection,
	COLLECTION_PARTIAL_MOVE_TO_NEW_COLLECTION,
} from 'store/collections/collection-partial-move-actions'
import { ProjectTreePickerDialogField } from 'views-components/projects-tree-picker/tree-picker-field'
import { CollectionFileSelection } from 'store/collection-panel/collection-panel-files/collection-panel-files-state'
import { WithDialogProps } from 'store/dialog/with-dialog'
import { PickerIdProp } from 'store/tree-picker/picker-id'
import { DialogTitle, DialogContent } from '@mui/material'
import { DialogRichTextField } from 'components/dialog-form/dialog-text-field'
import { REQUIRED_VALIDATION, REQUIRED_LENGTH255_VALIDATION, MAXLENGTH_524288_VALIDATION } from 'validators/validators'
import { useStateWithValidation } from 'common/useStateWithValidation'

type DialogCollectionPartialMoveProps = WithDialogProps<{
	initialFormData: CollectionPartialMoveToNewCollectionFormData
	collectionFileSelection: CollectionFileSelection
}> &
	PickerIdProp & {
		moveCollectionPartialToNewCollection: (
			fileSelection: CollectionFileSelection,
			formData: CollectionPartialMoveToNewCollectionFormData
		) => void
	}

const mapDispatch = (dispatch: Dispatch) => ({
	moveCollectionPartialToNewCollection: (
		fileSelection: CollectionFileSelection,
		formData: CollectionPartialMoveToNewCollectionFormData
	) => {
		dispatch<any>(moveCollectionPartialToNewCollection(fileSelection, formData))
	},
})

export const DialogCollectionPartialMoveToNewCollection = compose(
	withDialog(COLLECTION_PARTIAL_MOVE_TO_NEW_COLLECTION),
	connect(null, mapDispatch)
)((props: DialogCollectionPartialMoveProps) => {
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
			<DialogTitle>Move to new collection</DialogTitle>
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
				props.moveCollectionPartialToNewCollection(collectionFileSelection, {
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
