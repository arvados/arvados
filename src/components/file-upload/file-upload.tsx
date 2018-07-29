// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { Button, Grid, StyleRulesCallback, WithStyles } from '@material-ui/core';
import { withStyles } from '@material-ui/core';

export interface Breadcrumb {
    label: string;
}

type CssRules = "item" | "currentItem" | "label";

const styles: StyleRulesCallback<CssRules> = theme => ({
    item: {
        opacity: 0.6
    },
    currentItem: {
        opacity: 1
    },
    label: {
        textTransform: "none"
    }
});

interface FileUploadProps {
}

export const FileUpload = withStyles(styles)(
    ({ classes }: FileUploadProps & WithStyles<CssRules>) =>
    <Grid container alignItems="center" wrap="nowrap">
    {
    }
    </Grid>
);
