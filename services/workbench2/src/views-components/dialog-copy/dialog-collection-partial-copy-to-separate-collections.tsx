// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react'
import { connect } from 'react-redux'
import { compose, Dispatch } from 'redux'
import { DialogTitle } from '@mui/material'
import { withDialog, WithDialogProps } from 'store/dialog/with-dialog'
import { ProjectTreePickerDialogField } from 'views-components/projects-tree-picker/tree-picker-field'
import { DialogForm } from 'components/dialog-form/dialog-form'
import {
	CollectionPartialCopyToSeparateCollectionsFormData,
	copyCollectionPartialToSeparateCollections,
} from 'store/collections/collection-partial-copy-actions'
import { PickerIdProp } from 'store/tree-picker/picker-id'
import { COLLECTION_PARTIAL_COPY_TO_SEPARATE_COLLECTIONS } from 'store/collections/collection-partial-copy-actions'
import { CollectionFileSelection } from 'store/collection-panel/collection-panel-files/collection-panel-files-state'
import { getFieldErrors, REQUIRED_VALIDATION } from 'validators/validators'

type DialogCollectionPartialCopyProps = WithDialogProps<CollectionFileSelection> &
	PickerIdProp & {
		copyCollectionPartialToSeparateCollections: (
			fileSelection: CollectionFileSelection,
			formData: CollectionPartialCopyToSeparateCollectionsFormData
		) => void
	}

const mapDispatch = (dispatch: Dispatch) => ({
	copyCollectionPartialToSeparateCollections: (
		fileSelection: CollectionFileSelection,
		formData: CollectionPartialCopyToSeparateCollectionsFormData
	) => {
		dispatch<any>(copyCollectionPartialToSeparateCollections(fileSelection, formData))
	},
})

export const DialogCollectionPartialCopyToSeparateCollection = compose(
	withDialog(COLLECTION_PARTIAL_COPY_TO_SEPARATE_COLLECTIONS),
	connect(null, mapDispatch),
)((props: DialogCollectionPartialCopyProps) => {
	const { open, data } = props
	const [selectedProjectUuid, setSelectedProjectUuid] = React.useState<string>('')
	const [formErrors, setFormErrors] = React.useState<string[]>([])

	const fieldErrors = getFieldErrors(selectedProjectUuid, REQUIRED_VALIDATION, 'Project')

	React.useEffect(() => {
		setFormErrors([...fieldErrors])
	}, [selectedProjectUuid])

	const fields = () => (
		<>
			<DialogTitle>Copy Selected Files to Separate Collections</DialogTitle>
			<ProjectTreePickerDialogField
				pickerId={props.pickerId}
				setSelectedProject={setSelectedProjectUuid}
			/>
		</>
	)

	return (
		<DialogForm
			open={open}
			fields={fields()}
			onSubmit={(event: React.FormEvent<HTMLFormElement>) => {
				event.preventDefault()
				props.copyCollectionPartialToSeparateCollections(data, {
					name: '',
					projectUuid: selectedProjectUuid,
				})
			}}
			formErrors={formErrors}
			closeDialog={props.closeDialog}
			clearFormValues={() => {
				setSelectedProjectUuid('')
			}}
		/>
	)
})
