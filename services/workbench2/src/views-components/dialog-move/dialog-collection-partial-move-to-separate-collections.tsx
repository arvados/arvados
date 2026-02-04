// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react'
import { compose, Dispatch } from 'redux'
import { DialogForm } from 'components/dialog-form/dialog-form'
import { connect } from 'react-redux'
import { withDialog } from 'store/dialog/with-dialog'
import {
	CollectionPartialMoveToSeparateCollectionsFormData,
	moveCollectionPartialToSeparateCollections,
	COLLECTION_PARTIAL_MOVE_TO_SEPARATE_COLLECTIONS,
} from 'store/collections/collection-partial-move-actions'
import { ProjectTreePickerDialogField } from 'views-components/projects-tree-picker/tree-picker-field'
import { CollectionFileSelection } from 'store/collection-panel/collection-panel-files/collection-panel-files-state'
import { WithDialogProps } from 'store/dialog/with-dialog'
import { PickerIdProp } from 'store/tree-picker/picker-id'
import { DialogTitle, DialogContent } from '@mui/material'
import { COLLECTION_PROJECT_VALIDATION } from 'validators/validators'
import { useStateWithValidation } from 'common/useStateWithValidation'

type DialogCollectionPartialMoveProps = WithDialogProps<{
	initialData: CollectionPartialMoveToSeparateCollectionsFormData
	collectionFileSelection: CollectionFileSelection
}> &
	PickerIdProp & {
		moveCollectionPartialToSeparateCollections: (
			fileSelection: CollectionFileSelection,
			formData: CollectionPartialMoveToSeparateCollectionsFormData
		) => void
	}

const mapDispatch = (dispatch: Dispatch) => ({
	moveCollectionPartialToSeparateCollections: (
		fileSelection: CollectionFileSelection,
		formData: CollectionPartialMoveToSeparateCollectionsFormData
	) => {
		dispatch<any>(moveCollectionPartialToSeparateCollections(fileSelection, formData))
	},
})

export const DialogCollectionPartialMoveToSeparateCollections = compose(
	withDialog(COLLECTION_PARTIAL_MOVE_TO_SEPARATE_COLLECTIONS),
	connect(null, mapDispatch)
)((props: DialogCollectionPartialMoveProps) => {
	const { open, data, pickerId } = props
	const { initialData, collectionFileSelection } = data

	const [projectUuid, setProjectUuid, projectUuidErrs] = useStateWithValidation(initialData?.projectUuid || '', COLLECTION_PROJECT_VALIDATION, 'Project')

	const fields = () => (
		<>
			<DialogTitle>Move to separate collections</DialogTitle>
			<DialogContent>
				<ProjectTreePickerDialogField
					pickerId={pickerId}
					setSelectedProject={setProjectUuid}
				/>
			</DialogContent>
		</>
	)

	return (
		<DialogForm
			open={open}
			fields={fields()}
			submitLabel='Create collections'
			formErrors={projectUuidErrs}
			onSubmit={(event: React.FormEvent<HTMLFormElement>) => {
				event.preventDefault()
				props.moveCollectionPartialToSeparateCollections(collectionFileSelection, {
					name: initialData?.name || '',
					projectUuid: projectUuid,
				})
			}}
			closeDialog={props.closeDialog}
			clearFormValues={() => {
				setProjectUuid('')
			}}
		/>
	)
})
