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
    Checkbox,
    FormControlLabel,
    Typography,
} from '@mui/material';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { DialogActions } from 'components/dialog-actions/dialog-actions';
import { SharingURLsContent } from './sharing-urls';
import {
    extractUuidObjectType,
    ResourceObjectType
} from 'models/resource';
import { SharingInvitationForm } from './sharing-invitation-form';
import { SharingManagementForm } from './sharing-management-form';
import moment, { Moment } from 'moment';
import { SharingPublicAccessForm } from './sharing-public-access-form';
import { LocalizationProvider } from '@mui/x-date-pickers/LocalizationProvider';
import { AdapterMoment } from '@mui/x-date-pickers/AdapterMoment';
import { StaticDateTimePicker } from '@mui/x-date-pickers/StaticDateTimePicker';

export interface SharingDialogDataProps {
    open: boolean;
    loading: boolean;
    saveEnabled: boolean;
    sharedResourceUuid: string;
    sharingURLsNr: number;
    privateAccess: boolean;
    sharingURLsDisabled: boolean;
    permissions: any[];
}
export interface SharingDialogActionProps {
    onClose: () => void;
    onSave: () => void;
    onCreateSharingToken: (d: Date | undefined) => () => void;
    refreshPermissions: () => void;
}
enum SharingDialogTab {
    PERMISSIONS = 0,
    URLS = 1,
}
export type SharingDialogComponentProps = SharingDialogDataProps & SharingDialogActionProps;

export const SharingDialogComponent = (props: SharingDialogComponentProps) => {
    const { open, loading, saveEnabled, sharedResourceUuid,
        sharingURLsNr, privateAccess, sharingURLsDisabled,
        onClose, onSave, onCreateSharingToken, refreshPermissions } = props;
    const showTabs = !sharingURLsDisabled && extractUuidObjectType(sharedResourceUuid) === ResourceObjectType.COLLECTION;
    const [tabNr, setTabNr] = React.useState<number>(SharingDialogTab.PERMISSIONS);
    const [expDate, setExpDate] = React.useState<Moment>();
    const [withExpiration, setWithExpiration] = React.useState<boolean>(false);

    const handleChange = (newValue: moment.Moment) => setExpDate(newValue);
    const handleClose = (ev, reason) => {
        if (reason !== 'backdropClick') {
            onClose();
        }
    }

    // Sets up the dialog depending on the resource type
    if (!showTabs && tabNr !== SharingDialogTab.PERMISSIONS) {
        setTabNr(SharingDialogTab.PERMISSIONS);
    }

    React.useEffect(() => {
        if (!withExpiration) {
            setExpDate(undefined);
        } else {
            setExpDate(moment().add(2, 'hour'));
        }
    }, [withExpiration]);

    return (
        <Dialog {...{ open, onClose }} className="sharing-dialog" onClose={handleClose} fullWidth maxWidth='md' >
            <DialogTitle>
                Sharing settings
            </DialogTitle>
            {showTabs &&
                <Tabs value={tabNr}
                    onChange={(_, tb) => {
                        if (tb === SharingDialogTab.PERMISSIONS) {
                            refreshPermissions();
                        }
                        setTabNr(tb)
                    }
                    }>
                    <Tab label="With users/groups" />
                    <Tab label={`Sharing URLs ${sharingURLsNr > 0 ? '(' + sharingURLsNr + ')' : ''}`} disabled={saveEnabled} />
                </Tabs>
            }
            <DialogContent>
                {tabNr === SharingDialogTab.PERMISSIONS &&
                    <Grid container direction='column' spacing={3}>
                        <Grid item>
                            <SharingInvitationForm onSave={onSave} />
                        </Grid>
                        <Grid item>
                            <SharingManagementForm onSave={onSave} />
                        </Grid>
                        <Grid item>
                            <SharingPublicAccessForm onSave={onSave} />
                        </Grid>
                    </Grid>
                }
                {tabNr === SharingDialogTab.URLS &&
                    <SharingURLsContent uuid={sharedResourceUuid} />
                }
            </DialogContent>
            <DialogActions>
                <Grid container spacing={1} style={{ display: 'flex', width: '100%', flexDirection: 'column', alignItems: 'center'}}>
                    {tabNr === SharingDialogTab.URLS && withExpiration && 
                        <>
                            <section style={{minHeight: '42dvh', display: 'flex', flexDirection: 'column' }}>
                                <LocalizationProvider dateAdapter={AdapterMoment}>
                                    <StaticDateTimePicker 
                                        orientation="landscape" 
                                        onChange={handleChange} 
                                        value={expDate || moment().add(2, 'hour')} 
                                        disablePast
                                        minutesStep={5}
                                        ampm={false}
                                        slots={{
                                            //removes redundant action bar
                                            actionBar: () => null,
                                        }}
                                    />
                                </LocalizationProvider>
                            </section>
                            <Typography variant='caption' align='center' marginBottom='1rem'>
                                Maximum expiration date may be limited by the cluster configuration.
                            </Typography>
                        </>
                        }
                    {tabNr === SharingDialogTab.PERMISSIONS && !sharingURLsDisabled &&
                        privateAccess && sharingURLsNr > 0 &&
                        <Grid item md={12}>
                            <Typography variant='caption' align='center' color='error'>
                                Although there aren't specific permissions set, this is publicly accessible via Sharing URL(s).
                            </Typography>
                        </Grid>
                    }
                    <Grid style={{display: 'flex', justifyContent: 'end', flexDirection: 'row', width: '100%', marginBottom: '-0.5rem'}}>
                        {tabNr === SharingDialogTab.URLS && 
                            <Grid container style={{ display: 'flex', justifyContent: 'space-between'}}>
                                <Grid display='flex'>
                                    <Grid item>
                                        <FormControlLabel
                                            control={<Checkbox color="primary" checked={withExpiration}
                                                onChange={(e) => setWithExpiration(e.target.checked)} />}
                                            label="With expiration" />
                                    </Grid>
                                    <Grid item>
                                        <Button variant="contained" color="primary"
                                            disabled={expDate !== undefined && expDate.toDate() <= new Date()}
                                            onClick={onCreateSharingToken(expDate?.toDate())}>
                                            Create sharing URL
                                        </Button>
                                    </Grid>
                                </Grid>
                            </Grid>
                        }
                        <Grid>
                            <Grid style={{display: 'flex'}}>
                                <Button onClick={() => {
                                    onClose();
                                    setWithExpiration(false);
                                    }}
                                    disabled={saveEnabled}
                                    color='primary'
                                    size='small'
                                    style={{ marginLeft: '10px' }}
                                    >
                                        Close
                                </Button>
                                {tabNr !== SharingDialogTab.URLS && 
                                    <Button onClick={() => {
                                            onSave();
                                        }}
                                        data-cy="add-invited-people"
                                        disabled={!saveEnabled}
                                        color='primary'
                                        variant='contained'
                                        size='small'
                                        style={{ marginLeft: '10px' }}
                                        >
                                            Save
                                    </Button>
                                }
                            </Grid>
                        </Grid>
                    </Grid>
                </Grid>
            </DialogActions>
            {
                loading && <LoadingIndicator />
            }
        </Dialog>
    );
};

const loadingIndicatorStyles: CustomStyleRulesCallback<'root'> = theme => ({
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
