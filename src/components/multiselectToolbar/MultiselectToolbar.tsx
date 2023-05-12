// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React, { ReactElement } from 'react';
import { connect } from 'react-redux';
import { StyleRulesCallback, withStyles, WithStyles, Toolbar, Button } from '@material-ui/core';
import { ArvadosTheme } from 'common/custom-theme';
import { RootState } from 'store/store';
import { CopyToClipboardSnackbar } from 'components/copy-to-clipboard-snackbar/copy-to-clipboard-snackbar';
import { TCheckedList } from 'components/data-table/data-table';

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
    fn: (checkedList) => ReactElement;
};

export type MultiselectToolbarProps = {
    buttons: Array<MultiselectToolbarAction>;
    checkedList: TCheckedList;
};

export const defaultActions: Array<MultiselectToolbarAction> = [
    {
        fn: (checkedList) => MSToolbarCopyButton(checkedList),
    },
];

const MSToolbarCopyButton = (checkedList) => {
    let stringifiedSelectedList: string = '';
    for (const [key, value] of Object.entries(checkedList)) {
        if (value === true) {
            stringifiedSelectedList += key + '\n';
        }
    }
    return <CopyToClipboardSnackbar value={stringifiedSelectedList} children={<div>Copy</div>} />;
};

export const MultiselectToolbar = connect(mapStateToProps)(
    withStyles(styles)((props: MultiselectToolbarProps & WithStyles<CssRules>) => {
        const { classes, buttons, checkedList } = props;
        return (
            <Toolbar className={classes.root}>
                {buttons.map((btn, i) => (
                    <Button key={i} className={classes.button}>
                        {btn.fn(checkedList)}
                    </Button>
                ))}
            </Toolbar>
        );
    })
);

function mapStateToProps(state: RootState) {
    return {
        checkedList: state.multiselect.checkedList,
    };
}
