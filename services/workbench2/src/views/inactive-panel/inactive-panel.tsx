// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { Dispatch } from 'redux';
import { connect } from 'react-redux';
import { Grid, Typography, Button } from '@material-ui/core';
import { StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core/styles';
import { ArvadosTheme } from 'common/custom-theme';
import { navigateToLinkAccount } from 'store/navigation/navigation-action';
import { RootState } from 'store/store';

export type CssRules = 'root' | 'ontop' | 'title';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        position: 'relative',
        backgroundColor: theme.palette.grey["200"],
        background: 'url("arvados-logo-big.png") no-repeat center center',
        backgroundBlendMode: 'soft-light',
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
    startLinking: () => void;
}

const mapDispatchToProps = (dispatch: Dispatch): InactivePanelActionProps => ({
    startLinking: () => {
        dispatch<any>(navigateToLinkAccount);
    }
});

const mapStateToProps = (state: RootState): InactivePanelStateProps => ({
    inactivePageText: state.auth.config.clusterConfig.Workbench.InactivePageHTML,
    isLoginClusterFederation: state.auth.config.clusterConfig.Login.LoginCluster !== '',
});

export interface InactivePanelStateProps {
    inactivePageText: string;
    isLoginClusterFederation: boolean;
}

type InactivePanelProps = WithStyles<CssRules> & InactivePanelActionProps & InactivePanelStateProps;

export const InactivePanelRoot = ({ classes, startLinking, inactivePageText, isLoginClusterFederation }: InactivePanelProps) =>
    <Grid container justify="center" alignItems="center" direction="column" spacing={24}
        className={classes.root}
        style={{ marginTop: 56, height: "100%" }}>
        <Grid item>
            <Typography>
                <span dangerouslySetInnerHTML={{ __html: inactivePageText }} style={{ margin: "1em" }} />
            </Typography>
        </Grid>
        { !isLoginClusterFederation
        ? <><Grid item>
            <Typography align="center">
            If you would like to use this login to access another account click "Link Account".
            </Typography>
        </Grid>
        <Grid item>
            <Button className={classes.ontop} color="primary" variant="contained" onClick={() => startLinking()}>
                Link Account
            </Button>
        </Grid></>
        : <><Grid item>
            <Typography align="center">
                If you would like to use this login to access another account, please contact your administrator.
            </Typography>
        </Grid></> }
    </Grid >;

export const InactivePanel = connect(mapStateToProps, mapDispatchToProps)(
    withStyles(styles)(InactivePanelRoot));
