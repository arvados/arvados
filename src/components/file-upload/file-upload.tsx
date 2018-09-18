// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import * as classnames from 'classnames';
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
import { formatFileSize, formatProgress, formatUploadSpeed } from "~/common/formatters";
import { UploadFile } from '~/store/file-uploader/file-uploader-actions';

type CssRules = "root" | "dropzone" | "dropzoneWrapper" | "container" | "uploadIcon";

import './file-upload.css';
import { DOMElement, RefObject } from "react";

const styles: StyleRulesCallback<CssRules> = theme => ({
    root: {
    },
    dropzone: {
        width: "100%",
        height: "100%",
        overflow: "auto"
    },
    dropzoneWrapper: {
        width: "100%",
        height: "200px",
        position: "relative",
        border: "1px solid rgba(0, 0, 0, 0.42)"
    },
    container: {
        height: "100%"
    },
    uploadIcon: {
        verticalAlign: "middle"
    }
});

interface FileUploadPropsData {
    files: UploadFile[];
    disabled: boolean;
    onDrop: (files: File[]) => void;
}

interface FileUploadState {
    focused: boolean;
}

export type FileUploadProps = FileUploadPropsData & WithStyles<CssRules>;

export const FileUpload = withStyles(styles)(
    class extends React.Component<FileUploadProps, FileUploadState> {
        constructor(props: FileUploadProps) {
            super(props);
            this.state = {
                focused: false
            };
        }
        render() {
            const { classes, onDrop, disabled, files } = this.props;
            return <div className={"file-upload-dropzone " + classes.dropzoneWrapper}>
                <div className={classnames("dropzone-border-left", { "dropzone-border-left-active": this.state.focused })}/>
                <div className={classnames("dropzone-border-right", { "dropzone-border-right-active": this.state.focused })}/>
                <div className={classnames("dropzone-border-top", { "dropzone-border-top-active": this.state.focused })}/>
                <div className={classnames("dropzone-border-bottom", { "dropzone-border-bottom-active": this.state.focused })}/>
                <Dropzone className={classes.dropzone}
                    onDrop={files => onDrop(files)}
                    onClick={(e) => {
                        const el = document.getElementsByClassName("file-upload-dropzone")[0];
                        const inputs = el.getElementsByTagName("input");
                        if (inputs.length > 0) {
                            inputs[0].focus();
                        }
                    }}
                    disabled={disabled}
                    inputProps={{
                        onFocus: () => {
                            this.setState({
                                focused: true
                            });
                        },
                        onBlur: () => {
                            this.setState({
                                focused: false
                            });
                        }
                }}>
                    {files.length === 0 &&
                        <Grid container justify="center" alignItems="center" className={classes.container}>
                            <Grid item component={"span"}>
                                <Typography variant={"subheading"}>
                                    <CloudUploadIcon className={classes.uploadIcon} /> Drag and drop data or click to browse
                            </Typography>
                            </Grid>
                        </Grid>}
                    {files.length > 0 &&
                        <Table style={{ width: "100%" }}>
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
            </div>;
        }
    }
);
