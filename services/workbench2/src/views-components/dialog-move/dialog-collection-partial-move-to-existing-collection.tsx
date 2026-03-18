// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react'
import { compose, Dispatch } from 'redux'
import { DialogForm } from 'components/dialog-form/dialog-form'
import { connect } from 'react-redux'
import { withDialog } from 'store/dialog/with-dialog'
import {
	CollectionPartialMoveToExistingCollectionFormData,
	moveCollectionPartialToExistingCollection,
	COLLECTION_PARTIAL_MOVE_TO_SELECTED_COLLECTION,
} from 'store/collections/collection-partial-move-actions'
import { DirectoryTreePickerDialogField } from 'views-components/projects-tree-picker/tree-picker-field'
import { CollectionFileSelection } from 'store/collection-panel/collection-panel-files/collection-panel-files-state'
import { WithDialogProps } from 'store/dialog/with-dialog'
import { PickerIdProp } from 'store/tree-picker/picker-id'
import { DialogTitle, DialogContent } from '@mui/material'
import { FILE_OPS_LOCATION_VALIDATION } from 'validators/validators'
import { useStateWithValidation } from 'common/useStateWithValidation'

type DialogCollectionPartialMoveProps = WithDialogProps<{
	initialFormData: CollectionPartialMoveToExistingCollectionFormData
	collectionFileSelection: CollectionFileSelection
}> &
	PickerIdProp & {
		moveCollectionPartialToExistingCollection: (
			fileSelection: CollectionFileSelection,
			formData: CollectionPartialMoveToExistingCollectionFormData
		) => void
	}

const mapDispatch = (dispatch: Dispatch) => ({
	moveCollectionPartialToExistingCollection: (
		fileSelection: CollectionFileSelection,
		formData: CollectionPartialMoveToExistingCollectionFormData
	) => {
		dispatch<any>(moveCollectionPartialToExistingCollection(fileSelection, formData))
	},
})

export const DialogCollectionPartialMoveToExistingCollection = compose(
	withDialog(COLLECTION_PARTIAL_MOVE_TO_SELECTED_COLLECTION),
	connect(null, mapDispatch)
)((props: DialogCollectionPartialMoveProps) => {
	const { open, data, pickerId } = props
	const { initialFormData, collectionFileSelection } = data
	const [destination, setDestination, destinationErrs] = useStateWithValidation(initialFormData?.destination || {}, FILE_OPS_LOCATION_VALIDATION, 'Destination')

	const fields = () => (
		<>
			<DialogTitle>Move to existing collection</DialogTitle>
			<DialogContent>
				<DirectoryTreePickerDialogField
					pickerId={pickerId}
					currentUuids={initialFormData?.destination.uuid ? [initialFormData.destination.uuid] : []}
					handleDirectoryChange={setDestination}
				/>
			</DialogContent>
		</>
	)

	return (
		<DialogForm
			open={open}
			fields={fields()}
            submitLabel='Move files'
			formErrors={destinationErrs}
			onSubmit={(event: React.FormEvent<HTMLFormElement>) => {
				event.preventDefault()
				props.moveCollectionPartialToExistingCollection(collectionFileSelection, {
					destination: destination,
				})
			}}
			closeDialog={props.closeDialog}
			clearFormValues={() => {
				setDestination({} as any)
			}}
		/>
	)
})
