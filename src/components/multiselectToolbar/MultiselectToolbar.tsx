// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { connect } from 'react-redux';
import { StyleRulesCallback, withStyles, WithStyles, Toolbar, Button } from '@material-ui/core';
import { ArvadosTheme } from 'common/custom-theme';
import { RootState } from 'store/store';

type CssRules = 'root' | 'item';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        display: 'flex',
        flexDirection: 'row',
    },
    item: {
        color: theme.palette.text.primary,
        margin: '0.5rem',
    },
});

type MultiselectToolbarAction = {
    name: string;
    fn: () => void;
};

export type MultiselectToolbarActions = {
    actions: Array<MultiselectToolbarAction>;
};

export const defaultActions: Array<MultiselectToolbarAction> = [
    {
        name: 'foo',
        fn: () => console.log('yo'),
    },
];

type MultiselectToolbarProps = MultiselectToolbarActions & WithStyles<CssRules>;

export const MultiselectToolbar = connect(mapStateToProps)(
    withStyles(styles)((props: MultiselectToolbarProps) => {
        const { classes, actions } = props;
        return (
            <Toolbar className={classes.root}>
                {actions.map((action, i) => (
                    <Button key={i} onClick={action.fn}>
                        {action.name}
                    </Button>
                ))}
            </Toolbar>
        );
    })
);

function mapStateToProps(state: RootState) {
    return {
        state: state,
    };
}
