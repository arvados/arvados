// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react'
import { connect } from 'react-redux'
import { compose, Dispatch } from 'redux'
import { Dialog, DialogTitle, DialogActions, Button } from '@mui/material'
import { WithStyles, withStyles } from '@mui/styles'
import { withDialog } from 'store/dialog/with-dialog'
import { ProjectTreePickerDialogField } from 'views-components/projects-tree-picker/tree-picker-field'
import { WithDialogProps } from 'store/dialog/with-dialog'
import {
	CollectionPartialCopyToSeparateCollectionsFormData,
	copyCollectionPartialToSeparateCollections,
} from 'store/collections/collection-partial-copy-actions'
import { PickerIdProp } from 'store/tree-picker/picker-id'
import { CustomStyleRulesCallback } from 'common/custom-theme'
import { COLLECTION_PARTIAL_COPY_TO_SEPARATE_COLLECTIONS } from 'store/collections/collection-partial-copy-actions'
import { CollectionFileSelection } from 'store/collection-panel/collection-panel-files/collection-panel-files-state'
import { getFieldErrors, REQUIRED_VALIDATION } from 'validators/validators'

type CssRules = 'root' | 'paper'

const styles: CustomStyleRulesCallback<CssRules> = (theme) => ({
	root: {
		fontSize: '0.875rem',
	},
	paper: {
        padding: theme.spacing(1),
		width: '800px',
	},
})

type DialogCollectionPartialCopyProps = WithDialogProps<CollectionFileSelection> &
	PickerIdProp &
	WithStyles<CssRules> & {
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
	withStyles(styles)
)((props: DialogCollectionPartialCopyProps) => {
	const { open, data, classes } = props
	const [selectedProjectUuid, setSelectedProjectUuid] = React.useState<string>('')

	const formErrors = getFieldErrors(selectedProjectUuid, REQUIRED_VALIDATION, 'Project')

	const handleClose = (reason?: string) => {
		if (reason === 'backdropClick' || reason === 'escapeKeyDown') {
			return
		}
		props.closeDialog()
	}

	return (
		<Dialog
			open={open}
			onClose={(_, reason) => handleClose(reason)}
            fullWidth
            maxWidth={false}
            className={classes.root}
			PaperProps={{
				component: 'form',
				className: classes.paper,
				onSubmit: (event: React.FormEvent<HTMLFormElement>) => {
					event.preventDefault()
					props.copyCollectionPartialToSeparateCollections(data, {
						name: '',
						projectUuid: selectedProjectUuid,
					})
					handleClose()
				},
			}}
		>
            <DialogTitle>Copy Selected Files to Separate Collections</DialogTitle>
			<ProjectTreePickerDialogField
				pickerId={props.pickerId}
				setSelectedProject={setSelectedProjectUuid}
			/>
			<DialogActions>
				<Button onClick={props.closeDialog}>Cancel</Button>
				<Button disabled={formErrors.length > 0} type="submit">
					Copy
				</Button>
			</DialogActions>
		</Dialog>
	)
})
