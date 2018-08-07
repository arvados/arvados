// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import {
    Grid,
    StyleRulesCallback,
    Table, TableBody, TableCell, TableHead, TableRow,
    Typography,
    WithStyles
} from '@material-ui/core';
import { withStyles } from '@material-ui/core';
import Dropzone from 'react-dropzone';
import { CloudUploadIcon } from "../icon/icon";
import { formatFileSize, formatProgress, formatUploadSpeed } from "../../common/formatters";
import { UploadFile } from "../../store/collections/uploader/collection-uploader-actions";

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
    files: UploadFile[];
    disabled: boolean;
    onDrop: (files: File[]) => void;
}

export const FileUpload = withStyles(styles)(
    ({ classes, files, disabled, onDrop }: FileUploadProps & WithStyles<CssRules>) =>
    <Grid container direction={"column"}>
        <Typography variant={"subheading"}>
            Upload data
        </Typography>
        <Dropzone className={classes.dropzone} onDrop={files => onDrop(files)} disabled={disabled}>
            {files.length === 0 &&
            <Grid container justify="center" alignItems="center" className={classes.container}>
                <Grid item component={"span"}>
                    <Typography variant={"subheading"}>
                        <CloudUploadIcon className={classes.uploadIcon}/> Drag and drop data or click to browse
                    </Typography>
                </Grid>
            </Grid>}
            {files.length > 0 &&
                <Table style={{width: "100%"}}>
                    <TableHead>
                        <TableRow>
                            <TableCell>File name</TableCell>
                            <TableCell>File size</TableCell>
                            <TableCell>Upload speed</TableCell>
                            <TableCell>Upload progress</TableCell>
                        </TableRow>
                    </TableHead>
                    <TableBody>
                    {files.map(f =>
                        <TableRow key={f.id}>
                            <TableCell>{f.file.name}</TableCell>
                            <TableCell>{formatFileSize(f.file.size)}</TableCell>
                            <TableCell>{formatUploadSpeed(f.prevLoaded, f.loaded, f.prevTime, f.currentTime)}</TableCell>
                            <TableCell>{formatProgress(f.loaded, f.total)}</TableCell>
                        </TableRow>
                    )}
                    </TableBody>
                </Table>
            }
        </Dropzone>
    </Grid>
);
