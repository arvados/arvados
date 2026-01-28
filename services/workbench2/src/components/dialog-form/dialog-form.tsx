// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React, { useEffect } from "react"
import { Button, Dialog, DialogActions } from "@mui/material"
import withStyles, { WithStyles } from "@mui/styles/withStyles/withStyles";
import { CustomStyleRulesCallback } from "common/custom-theme";
import { CircularSuspense } from "components/loading/circular-suspense";

type CssRules = "paper" | "root" | "actions";

const styles: CustomStyleRulesCallback<CssRules> = (theme) => ({
    root: {
        fontSize: '0.875rem',
    },
    paper: {
        width: '800px',
    },
    actions: {
        paddingTop: 0,
        paddingRight: theme.spacing(2),
        paddingBottom: theme.spacing(2),
    }
})

type DialogFormProps = WithStyles<CssRules> & {
    open: boolean;
    fields: React.ReactNode;
    submitLabel?: string;
    formErrors: string[];
    isSubmitting?: boolean;
    onSubmit: (data: any) => void;
    closeDialog: () => void;
    clearFormValues: () => void;
}

export const DialogForm = withStyles(styles)((props: DialogFormProps) => {
    const { open, fields, submitLabel, classes, formErrors, isSubmitting = false, onSubmit, closeDialog, clearFormValues } = props;

    useEffect(() => {
        if (!open) {
            clearFormValues();
        }
    }, [open]);

	const handleClose = (reason?: string) => {
		if (reason === 'backdropClick' || reason === 'escapeKeyDown') {
			return
		}
		props.closeDialog()
	}

    return (
        <Dialog
            data-cy="form-dialog"
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
            {fields}
            <DialogActions className={classes.actions}>
                <Button data-cy="form-cancel-btn" onClick={closeDialog}>Cancel</Button>
                <CircularSuspense
                    showElement={!isSubmitting}
                    element={<Button data-cy="form-submit-btn" disabled={formErrors.length > 0} type="submit">
                                {submitLabel && submitLabel.length > 0 ? submitLabel : "Submit"}
                            </Button>}
                />
            </DialogActions>
        </Dialog>
    )
})