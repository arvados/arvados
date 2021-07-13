// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { Dispatch } from "redux";
import { connect } from "react-redux";
import { RootState } from "store/store";
import { Button, IconButton, StyleRulesCallback, WithStyles, withStyles, SnackbarContent } from '@material-ui/core';
import MaterialSnackbar, { SnackbarOrigin } from "@material-ui/core/Snackbar";
import { snackbarActions, SnackbarKind } from "store/snackbar/snackbar-actions";
import { navigateTo } from 'store/navigation/navigation-action';
import WarningIcon from '@material-ui/icons/Warning';
import CheckCircleIcon from '@material-ui/icons/CheckCircle';
import ErrorIcon from '@material-ui/icons/Error';
import InfoIcon from '@material-ui/icons/Info';
import CloseIcon from '@material-ui/icons/Close';
import { ArvadosTheme } from "common/custom-theme";
import { amber, green } from "@material-ui/core/colors";
import classNames from 'classnames';

interface SnackbarDataProps {
    anchorOrigin?: SnackbarOrigin;
    autoHideDuration?: number;
    open: boolean;
    message?: React.ReactElement<any>;
    kind: SnackbarKind;
    link?: string;
}

interface SnackbarEventProps {
    onClose?: (event: React.SyntheticEvent<any>, reason: string) => void;
    onExited: () => void;
    onClick: (uuid: string) => void;
}

const mapStateToProps = (state: RootState): SnackbarDataProps => {
    const messages = state.snackbar.messages;
    return {
        anchorOrigin: { vertical: "bottom", horizontal: "right" },
        open: state.snackbar.open,
        message: <span>{messages.length > 0 ? messages[0].message : ""}</span>,
        autoHideDuration: messages.length > 0 ? messages[0].hideDuration : 0,
        kind: messages.length > 0 ? messages[0].kind : SnackbarKind.INFO,
        link: messages.length > 0 ? messages[0].link : ''
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
    },
    onClick: (uuid: string) => {
        dispatch<any>(navigateTo(uuid));
    }
});

type CssRules = "success" | "error" | "info" | "warning" | "icon" | "iconVariant" | "message" | "linkButton";

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    success: {
        backgroundColor: green[600]
    },
    error: {
        backgroundColor: theme.palette.error.dark
    },
    info: {
        backgroundColor: theme.palette.primary.main
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
    linkButton: {
        fontWeight: 'bolder'
    }
});

type SnackbarProps = SnackbarDataProps & SnackbarEventProps & WithStyles<CssRules>;

export const Snackbar = withStyles(styles)(connect(mapStateToProps, mapDispatchToProps)(
    (props: SnackbarProps) => {
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
                            <Icon className={classNames(classes.icon, classes.iconVariant)} />
                            {props.message}
                        </span>
                    }
                    action={actions(props)}
                />
            </MaterialSnackbar>
        );
    }
));

const actions = (props: SnackbarProps) => {
    const { link, onClose, onClick, classes } = props;
    const actions = [
        <IconButton
            key="close"
            aria-label="Close"
            color="inherit"
            onClick={e => onClose && onClose(e, '')}>
            <CloseIcon className={classes.icon} />
        </IconButton>
    ];
    if (link) {
        actions.splice(0, 0,
            <Button key="goTo"
                aria-label="goTo"
                size="small"
                color="inherit"
                className={classes.linkButton}
                onClick={() => onClick(link)}>
                Go To
            </Button>
        );
    }
    return actions;
};
