// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { connect } from 'react-redux';
import { StyleRulesCallback, withStyles, WithStyles, Toolbar } from '@material-ui/core';
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

export type MultiselectToolbarActions = MultiselectToolbarAction[];

// type MultiselectToolbarProps = MultiselectToolbarActions & WithStyles<CssRules>;
type MultiselectToolbarProps = WithStyles<CssRules>;

export default connect(mapStateToProps)(
    withStyles(styles)((props: MultiselectToolbarProps) => {
        const { classes } = props;
        return (
            <Toolbar className={classes.root}>
                <div className={classes.item}>test1</div>
                <div className={classes.item}>test2</div>
                <div className={classes.item}>test3</div>
            </Toolbar>
        );
    })
);

function mapStateToProps(state: RootState) {
    return {
        state: state,
    };
}
