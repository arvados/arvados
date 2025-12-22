// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react'
import { compose, Dispatch } from 'redux'
import { connect } from 'react-redux'
import {
    Dialog,
    DialogTitle,
    DialogContent,
    DialogActions,
    Button,
    TextField,
} from '@mui/material'
import { withStyles, WithStyles } from '@mui/styles'
import { WithDialogProps, withDialog } from 'store/dialog/with-dialog'
import { ProjectTreePickerField } from 'views-components/projects-tree-picker/tree-picker-field'
import {
    validateTextField,
    COPY_NAME_VALIDATION,
} from 'validators/validators'
import { CopyFormDialogData } from 'store/copy-dialog/copy-dialog'
import { PickerIdProp } from 'store/tree-picker/picker-id'
import { CustomStyleRulesCallback } from 'common/custom-theme'
import { copyCollectionRunner } from 'store/workbench/workbench-actions'
import { COLLECTION_COPY_FORM_NAME, COLLECTION_MULTI_COPY_FORM_NAME } from 'store/collections/collection-copy-actions'
import { usePrevious } from 'common/usePrevious'

type CssRules = 'root' | 'paper'

const styles: CustomStyleRulesCallback<CssRules> = (theme) => ({
    root: {
        fontSize: '0.875rem',
    },
    paper: {
        width: '70vw',
        maxWidth: '70vw',
    },
})

type CopyDialogProps = WithDialogProps<CopyFormDialogData> & WithStyles<CssRules> &
    PickerIdProp & {
        selectedCollectionUuid: string | undefined
        copyCollection: (data: CopyFormDialogData) => void
    }

const mapDispatchToProps = (dispatch: Dispatch) => ({
    copyCollection: (data: CopyFormDialogData) => dispatch<any>(copyCollectionRunner(data)),
})

export const CopyCollectionDialog = compose(
    withDialog(COLLECTION_COPY_FORM_NAME),
    connect(null, mapDispatchToProps),
    withStyles(styles)
)((props: CopyDialogProps) => {
    const { open, data, classes } = props
    const [selectedProjectUuid, setSelectedProjectUuid] = React.useState<string>('')
    const [nameVal, setNameVal] = React.useState<string>(data.name || '')

	const isFormValid = nameVal.trim().length > 0 && nameVal.trim().length <= 255 && selectedProjectUuid.trim().length > 0

	// prevent stale selected project uuid when dialog is closed
	React.useEffect(() => {
		if (!open) {
			setSelectedProjectUuid('')
		}
	}, [open])

    const handleClose = () => {
        props.closeDialog()
    }

    return (
        <Dialog
            open={open}
            onClose={handleClose}
			fullWidth
			maxWidth={false}
			className={classes.root}
            PaperProps={{
                component: 'form',
                className: classes.paper,
                onSubmit: (event: React.FormEvent<HTMLFormElement>) => {
                    event.preventDefault()
                    props.copyCollection({
                        name: nameVal,
                        uuid: data.uuid,
                        ownerUuid: selectedProjectUuid,
                    })
                    handleClose()
                },
            }}
        >
            <DialogTitle>Make a copy</DialogTitle>
            <DialogContent>
                <NameField defaultName={data.name} setNameVal={setNameVal} />
            </DialogContent>
            <ProjectTreePickerField
                pickerId={props.pickerId}
                setSelectedProject={setSelectedProjectUuid}
            />
            <DialogActions>
                <Button onClick={handleClose}>Cancel</Button>
                <Button disabled={!isFormValid} type="submit">Copy</Button>
            </DialogActions>
        </Dialog>
    )
})

type NameFieldProps = {
	defaultName: string
	setNameVal: React.Dispatch<React.SetStateAction<string>>
}

const NameField = React.memo(({ defaultName, setNameVal }: NameFieldProps) => {
    const [value, setValue] = React.useState(defaultName)
    const err = validateTextField(value, COPY_NAME_VALIDATION)
	const prevErr = usePrevious(err)

	// trigger submit button enable/disable on valid/invalid input change
	React.useEffect(() => {
		if (!!prevErr !== !!err) {
			setNameVal(value.trim())
		}
		// eslint-disable-next-line react-hooks/exhaustive-deps
	}, [value])

    return (
        <TextField
            value={value}
            onChange={(e) => setValue(e.target.value)}
            autoFocus
            required
            error={!!err}
            helperText={err || ''}
            margin="dense"
            id="name"
            name="name"
            type="text"
            fullWidth
            variant="standard"
            label="Enter a new name for the copy"
            onBlur={() => setNameVal(value)}
        />
    )
})

export const CopyMultiCollectionDialog = compose(
    withDialog(COLLECTION_MULTI_COPY_FORM_NAME),
    connect(null, mapDispatchToProps),
    withStyles(styles)
)((props: CopyDialogProps) => {
    const { open, data, classes } = props
    const [selectedProjectUuid, setSelectedProjectUuid] = React.useState<string>('')

    const isFormValid = selectedProjectUuid.trim().length > 0

    // prevent stale selected project uuid when dialog is closed
    React.useEffect(() => {
        if (!open) {
            setSelectedProjectUuid('')
        }
    }, [open])

    const handleClose = () => {
        props.closeDialog()
    }

    return (
        <Dialog
            open={open}
            onClose={handleClose}
            fullWidth
            maxWidth={false}
            className={classes.root}
            PaperProps={{
                component: 'form',
                className: classes.paper,
                onSubmit: (event: React.FormEvent<HTMLFormElement>) => {
                    event.preventDefault()
                    props.copyCollection({
                        name: data.name, // Use original name for multi-copy
                        uuid: data.uuid,
                        ownerUuid: selectedProjectUuid,
                    })
                    handleClose()
                },
            }}
        >
            <DialogTitle>Make Copies</DialogTitle>
            <ProjectTreePickerField
                pickerId={props.pickerId}
                setSelectedProject={setSelectedProjectUuid}
            />
            <DialogActions>
                <Button onClick={handleClose}>Cancel</Button>
                <Button disabled={!isFormValid} type="submit">Copy</Button>
            </DialogActions>
        </Dialog>
    )
});
