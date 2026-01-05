// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React, { useEffect } from "react"
import { Button, Dialog, DialogActions } from "@mui/material"
import withStyles, { WithStyles } from "@mui/styles/withStyles/withStyles";
import { CustomStyleRulesCallback } from "common/custom-theme";

type CssRules = "paper" | "root";

const styles: CustomStyleRulesCallback<CssRules> = (theme) => ({
    root: {
        fontSize: '0.875rem',
    },
    paper: {
        width: '800px',
    },
})

type DialogFormProps = WithStyles<CssRules> & {
    open: boolean;
    fields: React.ReactNode;
    formErrors: string[];
    onSubmit: (data: any) => void;
    closeDialog: () => void;
    clearFormValues: () => void;
}

export const DialogForm = withStyles(styles)((props: DialogFormProps) => {
    const { open, fields, classes, formErrors, onSubmit, closeDialog, clearFormValues } = props;

    useEffect(() => {
        if (!open) {
            clearFormValues();
        }
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [open]);

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
                onSubmit: onSubmit,
            }}
        >
            {console.log('>>>errors:', formErrors)}
            {fields}
            <DialogActions>
                <Button onClick={closeDialog}>Cancel</Button>
                <Button disabled={formErrors.length > 0} type="submit">Copy</Button>
            </DialogActions>
        </Dialog>
    )
})