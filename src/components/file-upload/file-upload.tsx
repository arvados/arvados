// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { Grid, List, ListItem, ListItemText, StyleRulesCallback, Typography, WithStyles } from '@material-ui/core';
import { withStyles } from '@material-ui/core';
import Dropzone from 'react-dropzone';
import { CloudUploadIcon } from "../icon/icon";
import { formatFileSize } from "../../common/formatters";

type CssRules = "root" | "dropzone" | "container" | "uploadIcon";

const styles: StyleRulesCallback<CssRules> = theme => ({
    root: {
    },
    dropzone: {
        width: "100%",
        height: "200px",
        overflow: "auto",
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
    files: File[];
    onDrop: (files: File[]) => void;
}

export const FileUpload = withStyles(styles)(
    ({ classes, files, onDrop }: FileUploadProps & WithStyles<CssRules>) =>
    <Grid container direction={"column"}>
        <Typography variant={"subheading"}>
            Upload data
        </Typography>
        <Dropzone className={classes.dropzone} onDrop={files => onDrop(files)}>
            <Grid container justify="center" alignItems="center" className={classes.container} direction={"row"}>
                <Grid item component={"span"} style={{width: "100%", textAlign: "center"}}>
                    <Typography variant={"subheading"}>
                        <CloudUploadIcon className={classes.uploadIcon}/> Drag and drop data or click to browse
                    </Typography>
                </Grid>

                <Grid item style={{width: "100%"}}>
                    <List>
                    {files.map((f, idx) =>
                        <ListItem button key={idx}>
                            <ListItemText
                                primary={f.name} primaryTypographyProps={{variant: "body2"}}
                                secondary={formatFileSize(f.size)}/>
                        </ListItem>)}
                    </List>
                </Grid>
            </Grid>
        </Dropzone>
    </Grid>
);
