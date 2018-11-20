// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { Dialog, DialogTitle, DialogContent, DialogActions, Button } from "@material-ui/core";
import { WithDialogProps } from "~/store/dialog/with-dialog";
import { withDialog } from '~/store/dialog/with-dialog';
import { RICH_TEXT_EDITOR_DIALOG_NAME } from "~/store/rich-text-editor-dialog/rich-text-editor-dialog-actions";
import RichTextEditor from 'react-rte';

export interface RichTextEditorDialogDataProps {
    title: string;
    text: string;
}

export const RichTextEditorDialog = withDialog(RICH_TEXT_EDITOR_DIALOG_NAME)(
    (props: WithDialogProps<RichTextEditorDialogDataProps>) =>
        <Dialog open={props.open}
            onClose={props.closeDialog}
            fullWidth
            maxWidth='sm'>
            <DialogTitle>{props.data.title}</DialogTitle>
            <DialogContent>
                <RichTextEditor 
                    value={RichTextEditor.createValueFromString(props.data.text, 'html')}
                    readOnly={true} />
            </DialogContent>
            <DialogActions>
                <Button
                    variant='flat'
                    color='primary'
                    onClick={props.closeDialog}>
                    Close
                </Button>
            </DialogActions>
        </Dialog>
);