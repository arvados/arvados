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
} from '@material-ui/core';
import {
    StyleRulesCallback,
    WithStyles,
    withStyles
} from '@material-ui/core/styles';
import { DialogActions } from 'components/dialog-actions/dialog-actions';
import { SharingURLsContent } from './sharing-urls';
import {
    extractUuidObjectType,
    ResourceObjectType
} from 'models/resource';
import { SharingInvitationForm } from './sharing-invitation-form';
import { SharingManagementForm } from './sharing-management-form';
import {
    BasePicker,
    Calendar,
    MuiPickersUtilsProvider,
    TimePickerView
} from 'material-ui-pickers';
import DateFnsUtils from "@date-io/date-fns";
import moment from 'moment';
import { SharingPublicAccessForm } from './sharing-public-access-form';

export interface SharingDialogDataProps {
    open: boolean;
    loading: boolean;
    saveEnabled: boolean;
    sharedResourceUuid: string;
    sharingURLsNr: number;
    privateAccess: boolean;
    sharingURLsDisabled: boolean;
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

export default (props: SharingDialogComponentProps) => {
    const { open, loading, saveEnabled, sharedResourceUuid,
        sharingURLsNr, privateAccess, sharingURLsDisabled,
        onClose, onSave, onCreateSharingToken, refreshPermissions } = props;
    const showTabs = !sharingURLsDisabled && extractUuidObjectType(sharedResourceUuid) === ResourceObjectType.COLLECTION;
    const [tabNr, setTabNr] = React.useState<number>(SharingDialogTab.PERMISSIONS);
    const [expDate, setExpDate] = React.useState<Date>();
    const [withExpiration, setWithExpiration] = React.useState<boolean>(false);

    // Sets up the dialog depending on the resource type
    if (!showTabs && tabNr !== SharingDialogTab.PERMISSIONS) {
        setTabNr(SharingDialogTab.PERMISSIONS);
    }

    React.useEffect(() => {
        if (!withExpiration) {
            setExpDate(undefined);
        } else {
            setExpDate(moment().add(2, 'hour').minutes(0).seconds(0).toDate());
        }
    }, [withExpiration]);

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
        <Tabs value={tabNr}
            onChange={(_, tb) => {
                if (tb === SharingDialogTab.PERMISSIONS) {
                    refreshPermissions();
                }
                setTabNr(tb)}
            }>
            <Tab label="With users/groups" />
            <Tab label={`Sharing URLs ${sharingURLsNr > 0 ? '('+sharingURLsNr+')' : ''}`} disabled={saveEnabled} />
        </Tabs>
        }
        <DialogContent>
            { tabNr === SharingDialogTab.PERMISSIONS &&
            <Grid container direction='column' spacing={24}>
                <Grid item>
                    <SharingPublicAccessForm />
                </Grid>
                <Grid item>
                    <SharingManagementForm />
                </Grid>
            </Grid>
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
                </Grid>
                }
                { tabNr === SharingDialogTab.URLS && withExpiration && <>
                <Grid item container direction='row' md={12}>
                    <MuiPickersUtilsProvider utils={DateFnsUtils}>
                        <BasePicker autoOk value={expDate} onChange={setExpDate}>
                        {({ date, handleChange }) => (<>
                            <Grid item md={6}>
                                <Calendar date={date} minDate={new Date()} maxDate={undefined}
                                    onChange={handleChange} />
                            </Grid>
                            <Grid item md={6}>
                                <TimePickerView type="hours" date={date} ampm={false}
                                    onMinutesChange={() => {}}
                                    onSecondsChange={() => {}}
                                    onHourChange={handleChange}
                                />
                            </Grid>
                        </>)}
                        </BasePicker>
                    </MuiPickersUtilsProvider>
                </Grid>
                <Grid item md={12}>
                    <Typography variant='caption' align='center'>
                        Maximum expiration date may be limited by the cluster configuration.
                    </Typography>
                </Grid>
                </>
                }
                { tabNr === SharingDialogTab.PERMISSIONS && !sharingURLsDisabled &&
                    privateAccess && sharingURLsNr > 0 &&
                <Grid item md={12}>
                    <Typography variant='caption' align='center' color='error'>
                        Although there aren't specific permissions set, this is publicly accessible via Sharing URL(s).
                    </Typography>
                </Grid>
                }
                <Grid item xs />
                { tabNr === SharingDialogTab.URLS && <>
                <Grid item><FormControlLabel
                    control={<Checkbox color="primary" checked={withExpiration}
                        onChange={(e) => setWithExpiration(e.target.checked)} />}
                    label="With expiration" />
                </Grid>
                <Grid item>
                    <Button variant="contained" color="primary"
                        disabled={expDate !== undefined && expDate <= new Date()}
                        onClick={onCreateSharingToken(expDate)}>
                        Create sharing URL
                    </Button>
                </Grid>
                </>
                }
                { tabNr === SharingDialogTab.PERMISSIONS &&
                <Grid item>
                    <Button onClick={onSave} variant="contained" color="primary"
                        disabled={!saveEnabled}>
                        Save changes
                    </Button>
                </Grid>
                }
                <Grid item>
                    <Button onClick={() => {
                        onClose();
                        setWithExpiration(false);
                    }}>
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
