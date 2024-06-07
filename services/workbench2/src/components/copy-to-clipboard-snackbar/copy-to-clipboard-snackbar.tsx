// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { connect, DispatchProp } from 'react-redux';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { Tooltip, WithStyles, withStyles } from '@material-ui/core';
import { ArvadosTheme } from 'common/custom-theme';
import CopyToClipboard from 'react-copy-to-clipboard';
import { snackbarActions, SnackbarKind } from 'store/snackbar/snackbar-actions';
import { CopyIcon } from 'components/icon/icon';

type CssRules = 'copyIcon';

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    copyIcon: {
        marginLeft: theme.spacing(1),
        color: theme.palette.grey['500'],
        cursor: 'pointer',
        display: 'inline',
        '& svg': {
            fontSize: '1rem',
            verticalAlign: 'middle',
        },
    },
});

interface CopyToClipboardDataProps {
    children?: React.ReactNode;
    value: string;
}

type CopyToClipboardProps = CopyToClipboardDataProps & WithStyles<CssRules> & DispatchProp;

export const CopyToClipboardSnackbar = connect()(
    withStyles(styles)(
        class CopyToClipboardSnackbar extends React.Component<CopyToClipboardProps> {
            onCopy = () => {
                this.props.dispatch(
                    snackbarActions.OPEN_SNACKBAR({
                        message: 'Copied',
                        hideDuration: 2000,
                        kind: SnackbarKind.SUCCESS,
                    })
                );
            };

            render() {
                const { children, value, classes } = this.props;
                return (
                    <Tooltip title='Copy link to clipboard' onClick={(ev) => ev.stopPropagation()}>
                        <span className={classes.copyIcon}>
                            <CopyToClipboard text={value} onCopy={this.onCopy}>
                                {children || <CopyIcon />}
                            </CopyToClipboard>
                        </span>
                    </Tooltip>
                );
            }
        }
    )
);
