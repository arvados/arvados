// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { connect } from "react-redux";
import { RootState } from "~/store/store";
import MaterialSnackbar, { SnackbarProps } from "@material-ui/core/Snackbar";
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

const mapStateToProps = (state: RootState): SnackbarProps => ({
    anchorOrigin: { vertical: "bottom", horizontal: "right" },
    open: state.snackbar.open,
    message: <span>{state.snackbar.message}</span>,
    autoHideDuration: state.snackbar.hideDuration,
});

const mapDispatchToProps = (dispatch: Dispatch): Pick<SnackbarProps, "onClose"> => ({
    onClose: (event: any, reason: string) => {
        if (reason !== "clickaway") {
            dispatch(snackbarActions.CLOSE_SNACKBAR());
        }
    }
});

const ArvadosSnackbar = (props: any) => <MaterialSnackbar {...props}>
    <ArvadosSnackbarContent {...props}/>
</MaterialSnackbar>;

type CssRules = "success" | "error" | "info" | "warning" | "icon" | "iconVariant" | "message";

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    success: {
        backgroundColor: green[600],
    },
    error: {
        backgroundColor: theme.palette.error.dark,
    },
    info: {
        backgroundColor: theme.palette.primary.dark,
    },
    warning: {
        backgroundColor: amber[700],
    },
    icon: {
        fontSize: 20,
    },
    iconVariant: {
        opacity: 0.9,
        marginRight: theme.spacing.unit,
    },
    message: {
        display: 'flex',
        alignItems: 'center',
    },
});

interface ArvadosSnackbarProps {
    kind: SnackbarKind;
}

const ArvadosSnackbarContent = (props: SnackbarProps & ArvadosSnackbarProps & WithStyles<CssRules>) => {
    const { classes, className, message, onClose, kind, ...other } = props;

    let Icon = InfoIcon;
    let cssClass;
    switch (kind) {
        case SnackbarKind.INFO:
            Icon = InfoIcon;
            cssClass = classes.info;
            break;
        case SnackbarKind.WARNING:
            Icon = WarningIcon;
            cssClass = classes.warning;
            break;
        case SnackbarKind.SUCCESS:
            Icon = CheckCircleIcon;
            cssClass = classes.success;
            break;
        case SnackbarKind.ERROR:
            Icon = ErrorIcon;
            cssClass = classes.error;
            break;
    }

    return (
        <SnackbarContent
            className={classNames(cssClass, className)}
            aria-describedby="client-snackbar"
            message={
                <span id="client-snackbar" className={classes.message}>
                    <Icon className={classNames(classes.icon, classes.iconVariant)}/>
                    {message}
                </span>
            }
            action={
                <IconButton
                    key="close"
                    aria-label="Close"
                    color="inherit"
                    onClick={e => {
                        if (onClose) {
                            onClose(e, '');
                        }
                    }}>
                    <CloseIcon className={classes.icon}/>
                </IconButton>
            }
        />
    );
};

export const Snackbar = connect(mapStateToProps, mapDispatchToProps)(
    withStyles(styles)(ArvadosSnackbar)
);
