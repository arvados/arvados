// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { Dispatch } from "redux";
import { connect } from "react-redux";
import { RootState } from 'store/store';
import { withDialog, WithDialogProps } from "store/dialog/with-dialog";
import { NOT_FOUND_DIALOG_NAME } from 'store/not-found-panel/not-found-panel-action';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { Dialog, DialogContent, DialogActions, Button } from '@mui/material';
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { ArvadosTheme } from 'common/custom-theme';
import { NotFoundPanel } from "views/not-found-panel/not-found-panel";

type CssRules = 'tag';

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    tag: {
        marginRight: theme.spacing(1),
        marginBottom: theme.spacing(1)
    }
});

interface NotFoundDialogDataProps {

}

interface NotFoundDialogActionProps {

}

const mapStateToProps = (state: RootState): NotFoundDialogDataProps => ({

});

const mapDispatchToProps = (dispatch: Dispatch): NotFoundDialogActionProps => ({

});

type NotFoundDialogProps =  NotFoundDialogDataProps & NotFoundDialogActionProps & WithDialogProps<{}> & WithStyles<CssRules>;

export const NotFoundDialog = connect(mapStateToProps, mapDispatchToProps)(
    withStyles(styles)(
    withDialog(NOT_FOUND_DIALOG_NAME)(
        ({ open, closeDialog }: NotFoundDialogProps) =>
            <Dialog
                open={open}
                onClose={closeDialog}
                fullWidth
                maxWidth='md'
                disableEscapeKeyDown>
                <DialogContent>
                    <NotFoundPanel notWrapped />
                </DialogContent>
                <DialogActions>
                    <Button
                        variant='text'
                        color='primary'
                        onClick={closeDialog}>
                        Close
                    </Button>
                </DialogActions>
            </Dialog>
    )
));