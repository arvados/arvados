// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { Grid, StyleRulesCallback, Typography, WithStyles } from '@material-ui/core';
import { withStyles } from '@material-ui/core';
import Dropzone from 'react-dropzone';
import { CloudUploadIcon } from "../icon/icon";

type CssRules = "root" | "dropzone" | "container" | "uploadIcon";

const styles: StyleRulesCallback<CssRules> = theme => ({
    root: {
    },
    dropzone: {
        width: "100%",
        height: "100px",
        border: "1px dashed black",
        borderRadius: "5px"
    },
    container: {
        height: "100%"
    },
    uploadIcon: {
        verticalAlign: "middle"
    }
});

interface FileUploadProps {
}

export const FileUpload = withStyles(styles)(
    ({ classes }: FileUploadProps & WithStyles<CssRules>) =>
    <Grid container direction={"column"}>
        <Typography variant={"subheading"}>
            Upload data
        </Typography>
        <Dropzone className={classes.dropzone}>
            <Grid container justify="center" alignItems="center" className={classes.container}>
                <Grid item component={"span"}>
                    <Typography variant={"subheading"}>
                        <CloudUploadIcon className={classes.uploadIcon}/> Drag and drop data or <a>browse</a>
                    </Typography>
                </Grid>
            </Grid>
        </Dropzone>
    </Grid>
);
