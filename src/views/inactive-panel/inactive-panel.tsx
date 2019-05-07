// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { Dispatch } from 'redux';
import { connect } from 'react-redux';
import { saveAccountLinkData } from '~/store/link-account-panel/link-account-panel-actions';
import { Grid, Typography, Button } from '@material-ui/core';
import { StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core/styles';
import { ArvadosTheme } from '~/common/custom-theme';
import { LinkAccountType } from '~/models/link-account';


type CssRules = 'root' | 'ontop' | 'title';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        position: 'relative',
        backgroundColor: theme.palette.grey["200"],
        '&::after': {
            content: `''`,
            position: 'absolute',
            top: 0,
            left: 0,
            bottom: 0,
            right: 0,
            background: 'url("arvados-logo-big.png") no-repeat center center',
            opacity: 0.2,
        }
    },
    ontop: {
        zIndex: 10
    },
    title: {
        marginBottom: theme.spacing.unit * 6,
        color: theme.palette.grey["800"]
    }
});

export interface InactivePanelActionProps {
    linkAccount: () => void;
}

const mapDispatchToProps = (dispatch: Dispatch): InactivePanelActionProps => ({
    linkAccount: () => dispatch<any>(saveAccountLinkData(LinkAccountType.ACCESS_OTHER_ACCOUNT))
});

type InactivePanelProps =  WithStyles<CssRules> & InactivePanelActionProps;

export const InactivePanel = connect(null, mapDispatchToProps)(withStyles(styles)((({ classes, linkAccount }: InactivePanelProps) =>
        <Grid container justify="center" alignItems="center" direction="column" spacing={24}
            className={classes.root}
            style={{ marginTop: 56, height: "100%" }}>
            <Grid item>
                <Typography variant='h6' align="center" className={classes.title}>
                    Hi! You're logged in, but...
                </Typography>
            </Grid>
            <Grid item>
                <Typography align="center">
                    Your account is inactive. An administrator must activate your account before you can get any further.
                </Typography>
            </Grid>
            <Grid item>
                <Typography align="center">
                    If you would like to use this login to access another account click "Link Account".
                </Typography>
            </Grid>
            <Grid item>
                <Button className={classes.ontop} color="primary" variant="contained" onClick={() => linkAccount()}>
                    Link Account
                </Button>
            </Grid>
        </Grid >
    )));
