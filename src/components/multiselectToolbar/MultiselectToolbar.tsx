// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React, { ReactElement } from 'react';
import { connect } from 'react-redux';
import { StyleRulesCallback, withStyles, WithStyles, Toolbar, Button } from '@material-ui/core';
import { ArvadosTheme } from 'common/custom-theme';
import { RootState } from 'store/store';
import { CopyToClipboardSnackbar } from 'components/copy-to-clipboard-snackbar/copy-to-clipboard-snackbar';

type CssRules = 'root' | 'button';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        display: 'flex',
        flexDirection: 'row',
    },
    button: {
        color: theme.palette.text.primary,
        margin: '0.5rem',
    },
});

type MultiselectToolbarAction = {
    name: string;
    fn: () => ReactElement;
};

export type MultiselectToolbarActions = {
    actions: Array<MultiselectToolbarAction>;
};

export const defaultActions: Array<MultiselectToolbarAction> = [
    {
        name: 'foo',
        fn: () => MSToolbarCopyButton({ button: { border: '1px solid blue' } }),
    },
];

const MSToolbarCopyButton = (classes) => {
    return (
        <Button className={classes.button}>
            <CopyToClipboardSnackbar value='foo' children={<div>Copy</div>} />
        </Button>
    );
};

type MultiselectToolbarProps = MultiselectToolbarActions & WithStyles<CssRules>;

export const MultiselectToolbar = connect(mapStateToProps)(
    withStyles(styles)((props: MultiselectToolbarProps) => {
        const { classes, actions } = props;
        return (
            <Toolbar className={classes.root}>
                {actions.map((action, i) => (
                    <Button key={i} className={classes.button} onClick={action.fn}>
                        {action.name}
                    </Button>
                ))}
                <MSToolbarCopyButton />
                {/* <CopyToClipboardSnackbar value='foo' children={<div>test</div>} /> */}
            </Toolbar>
        );
    })
);

function mapStateToProps(state: RootState) {
    return {
        // state: state,
    };
}
