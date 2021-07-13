// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import {
    Dialog, 
    DialogTitle, 
    DialogContent, 
    DialogActions, 
    Button,
    StyleRulesCallback,
    WithStyles,
    withStyles
} from "@material-ui/core";
import { ArvadosTheme } from 'common/custom-theme';
import { WithDialogProps } from "store/dialog/with-dialog";
import { withDialog } from 'store/dialog/with-dialog';
import { RICH_TEXT_EDITOR_DIALOG_NAME } from "store/rich-text-editor-dialog/rich-text-editor-dialog-actions";
import RichTextEditor from 'react-rte';

type CssRules = 'rte';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    rte: {
        fontFamily: 'Arial',
        '& a': {
            textDecoration: 'none',
            color: theme.palette.primary.main,
            '&:hover': {
                cursor: 'pointer',
                textDecoration: 'underline'
            }
        }
    },

});

export interface RichTextEditorDialogDataProps {
    title: string;
    text: string;
}

export const RichTextEditorDialog = withStyles(styles)(withDialog(RICH_TEXT_EDITOR_DIALOG_NAME)(
    (props: WithDialogProps<RichTextEditorDialogDataProps> & WithStyles<CssRules>) =>
        <Dialog open={props.open}
            onClose={props.closeDialog}
            fullWidth
            maxWidth='md'>
            <DialogTitle>{props.data.title}</DialogTitle>
            <DialogContent>
                <RichTextEditor
                    className={props.classes.rte}
                    value={props.data.text ?
                        RichTextEditor.createValueFromString(props.data.text.replace(/&lt;/g, '<').replace(/&gt;/g, '>').replace(/&amp;/g, '&'), 'html') : ''}
                    readOnly={true} />
            </DialogContent>
            <DialogActions>
                <Button
                    variant='text'
                    color='primary'
                    onClick={props.closeDialog}>
                    Close
                </Button>
            </DialogActions>
        </Dialog>)
);