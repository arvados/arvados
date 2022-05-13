// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import {
    Dialog,
    DialogTitle,
    Button,
    Grid,
    DialogContent,
    CircularProgress,
    Paper,
    Tabs,
    Tab,
} from '@material-ui/core';
import {
    StyleRulesCallback,
    WithStyles,
    withStyles
} from '@material-ui/core/styles';
import { DialogActions } from 'components/dialog-actions/dialog-actions';
import { SharingDialogContent } from './sharing-dialog-content';
import { SharingURLsContent } from './sharing-urls';
import {
    extractUuidObjectType,
    ResourceObjectType
} from 'models/resource';
import { SharingInvitationForm } from './sharing-invitation-form';

export interface SharingDialogDataProps {
    open: boolean;
    loading: boolean;
    saveEnabled: boolean;
    sharedResourceUuid: string;
}
export interface SharingDialogActionProps {
    onClose: () => void;
    onSave: () => void;
    onCreateSharingToken: () => void;
}
enum SharingDialogTab {
    PERMISSIONS = 0,
    URLS = 1,
}
export default (props: SharingDialogDataProps & SharingDialogActionProps) => {
    const { open, loading, saveEnabled, sharedResourceUuid,
        onClose, onSave, onCreateSharingToken } = props;
    const showTabs = extractUuidObjectType(sharedResourceUuid) === ResourceObjectType.COLLECTION;
    const [tabNr, setTabNr] = React.useState<number>(SharingDialogTab.PERMISSIONS);

    // Sets up the dialog depending on the resource type
    if (!showTabs && tabNr !== SharingDialogTab.PERMISSIONS) {
        setTabNr(SharingDialogTab.PERMISSIONS);
    }

    return <Dialog
        {...{ open, onClose }}
        className="sharing-dialog"
        fullWidth
        maxWidth='sm'
        disableBackdropClick={saveEnabled}
        disableEscapeKeyDown={saveEnabled}>
        <DialogTitle>
            Sharing settings
        </DialogTitle>
        { showTabs &&
        <Tabs value={tabNr} onChange={(_, tb) => setTabNr(tb)}>
            <Tab label="With users/groups" />
            <Tab label="Sharing URLs" disabled={saveEnabled} />
        </Tabs>
        }
        <DialogContent>
            { tabNr === SharingDialogTab.PERMISSIONS &&
            <SharingDialogContent />
            }
            { tabNr === SharingDialogTab.URLS &&
            <SharingURLsContent uuid={sharedResourceUuid} />
            }
        </DialogContent>
        <DialogActions>
            <Grid container spacing={8}>
                { tabNr === SharingDialogTab.PERMISSIONS &&
                <Grid item md={12}>
                    <SharingInvitationForm />
                </Grid> }
                { tabNr === SharingDialogTab.URLS &&
                <Grid item>
                    <Button
                        variant="contained"
                        color="primary"
                        onClick={onCreateSharingToken}>
                        Create sharing URL
                    </Button>
                </Grid>
                }
                <Grid item xs />
                { tabNr === SharingDialogTab.PERMISSIONS &&
                <Grid item>
                    <Button
                        variant='contained'
                        color='primary'
                        onClick={onSave}
                        disabled={!saveEnabled}>
                        Save
                    </Button>
                </Grid>
                }
                <Grid item>
                    <Button onClick={onClose}>
                        Close
                    </Button>
                </Grid>
            </Grid>
        </DialogActions>
        {
            loading && <LoadingIndicator />
        }
    </Dialog>;
};

const loadingIndicatorStyles: StyleRulesCallback<'root'> = theme => ({
    root: {
        position: 'absolute',
        top: 0,
        right: 0,
        bottom: 0,
        left: 0,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        backgroundColor: 'rgba(255, 255, 255, 0.8)',
    },
});

const LoadingIndicator = withStyles(loadingIndicatorStyles)(
    (props: WithStyles<'root'>) =>
        <Paper classes={props.classes}>
            <CircularProgress />
        </Paper>
);
