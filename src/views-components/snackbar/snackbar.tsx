// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { connect } from "react-redux";
import { RootState } from "~/store/store";
import MaterialSnackbar, { SnackbarOrigin } from "@material-ui/core/Snackbar";
import { Dispatch } from "redux";
import { snackbarActions, SnackbarKind } from "~/store/snackbar/snackbar-actions";
import IconButton from '@material-ui/core/IconButton';
import SnackbarContent from '@material-ui/core/SnackbarContent';
import WarningIcon from '@material-ui/icons/Warning';
import CheckCircleIcon from '@material-ui/icons/CheckCircle';
import ErrorIcon from '@material-ui/icons/Error';
import InfoIcon from '@material-ui/icons/Info';
import CloseIcon from '@material-ui/icons/Close';
import { StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core/styles';
import { ArvadosTheme } from "~/common/custom-theme";
import { amber, green } from "@material-ui/core/colors";
import * as classNames from 'classnames';

interface SnackbarDataProps {
    anchorOrigin?: SnackbarOrigin;
    autoHideDuration?: number;
    open: boolean;
    message?: React.ReactElement<any>;
    kind: SnackbarKind;
}

interface SnackbarEventProps {
    onClose?: (event: React.SyntheticEvent<any>, reason: string) => void;
    onExited: () => void;
}

const mapStateToProps = (state: RootState): SnackbarDataProps => {
    const messages = state.snackbar.messages;
    return {
        anchorOrigin: { vertical: "bottom", horizontal: "right" },
        open: state.snackbar.open,
        message: <span>{messages.length > 0 ? messages[0].message : ""}</span>,
        autoHideDuration: messages.length > 0 ? messages[0].hideDuration : 0,
        kind: messages.length > 0 ? messages[0].kind : SnackbarKind.INFO
    };
};

const mapDispatchToProps = (dispatch: Dispatch): SnackbarEventProps => ({
    onClose: (event: any, reason: string) => {
        if (reason !== "clickaway") {
            dispatch(snackbarActions.CLOSE_SNACKBAR());
        }
    },
    onExited: () => {
        dispatch(snackbarActions.SHIFT_MESSAGES());
    }
});

type CssRules = "success" | "error" | "info" | "warning" | "icon" | "iconVariant" | "message";

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    success: {
        backgroundColor: green[600]
    },
    error: {
        backgroundColor: theme.palette.error.dark
    },
    info: {
        backgroundColor: theme.palette.primary.dark
    },
    warning: {
        backgroundColor: amber[700]
    },
    icon: {
        fontSize: 20
    },
    iconVariant: {
        opacity: 0.9,
        marginRight: theme.spacing.unit
    },
    message: {
        display: 'flex',
        alignItems: 'center'
    },
});

export const Snackbar = withStyles(styles)(connect(mapStateToProps, mapDispatchToProps)(
    (props: SnackbarDataProps & SnackbarEventProps & WithStyles<CssRules>) => {
        const { classes } = props;

        const variants = {
            [SnackbarKind.INFO]: [InfoIcon, classes.info],
            [SnackbarKind.WARNING]: [WarningIcon, classes.warning],
            [SnackbarKind.SUCCESS]: [CheckCircleIcon, classes.success],
            [SnackbarKind.ERROR]: [ErrorIcon, classes.error]
        };

        const [Icon, cssClass] = variants[props.kind];

        return (
            <MaterialSnackbar
                open={props.open}
                message={props.message}
                onClose={props.onClose}
                onExited={props.onExited}
                anchorOrigin={props.anchorOrigin}
                autoHideDuration={props.autoHideDuration}>
                <SnackbarContent
                    className={classNames(cssClass)}
                    aria-describedby="client-snackbar"
                    message={
                        <span id="client-snackbar" className={classes.message}>
                            <Icon className={classNames(classes.icon, classes.iconVariant)}/>
                            {props.message}
                        </span>
                    }
                    action={
                        <IconButton
                            key="close"
                            aria-label="Close"
                            color="inherit"
                            onClick={e => props.onClose && props.onClose(e, '')}>
                            <CloseIcon className={classes.icon}/>
                        </IconButton>
                    }
                />
            </MaterialSnackbar>
        );
    }
));
