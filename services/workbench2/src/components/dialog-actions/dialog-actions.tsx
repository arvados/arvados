// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { DialogActions as MuiDialogActions } from '@material-ui/core/';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { withStyles } from '@material-ui/core';

const styles: CustomStyleRulesCallback<'root'> = theme => {
    const margin = theme.spacing(3);
    return {
        root: {
            marginRight: margin,
            marginBottom: margin,
            marginLeft: margin,
        },
    };
};
export const DialogActions = withStyles(styles)(MuiDialogActions);
