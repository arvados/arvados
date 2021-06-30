
// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import classNames from 'classnames';
import { withRouter, RouteComponentProps } from 'react-router';
import { StyleRulesCallback, Button, WithStyles, withStyles } from "@material-ui/core";
import { ReRunProcessIcon } from 'components/icon/icon';

type CssRules = 'button' | 'buttonRight';

const styles: StyleRulesCallback<CssRules> = theme => ({
    button: {
        boxShadow: 'none',
        padding: '2px 10px 2px 5px',
        fontSize: '0.75rem'
    },
    buttonRight: {
        marginLeft: 'auto',
    },
});

export const RefreshButton = ({ history, classes }: RouteComponentProps & WithStyles<CssRules>) =>
    <Button
        color="primary"
        size="small"
        variant="contained"
        onClick={() => {
            history.replace(window.location.pathname);
        }}
        className={classNames(classes.buttonRight, classes.button)}>
        <ReRunProcessIcon />
        Refresh
    </Button>;

export default withStyles(styles)(withRouter(RefreshButton));