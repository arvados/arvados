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
    WithStyles,
    IconButton
} from '@material-ui/core';
import { withStyles } from '@material-ui/core';
import Dropzone from 'react-dropzone';
import { CloudUploadIcon, RemoveIcon } from "../icon/icon";
import { formatFileSize, formatProgress, formatUploadSpeed } from "common/formatters";
import { UploadFile } from 'store/file-uploader/file-uploader-actions';

type CssRules = "root" | "dropzone" | "dropzoneWrapper" | "container" | "uploadIcon"
    | "dropzoneBorder" | "dropzoneBorderLeft" | "dropzoneBorderRight" | "dropzoneBorderTop" | "dropzoneBorderBottom"
    | "dropzoneBorderHorzActive" | "dropzoneBorderVertActive" | "deleteButton" | "deleteButtonDisabled" | "deleteIcon";

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
    dropzoneBorder: {
        content: "",
        position: "absolute",
        transition: "transform 200ms cubic-bezier(0.0, 0, 0.2, 1) 0ms",
        pointerEvents: "none",
        backgroundColor: "#6a1b9a"
    },
    dropzoneBorderLeft: {
        left: -1,
        top: -1,
        bottom: -1,
        width: 2,
        transform: "scaleY(0)",
    },
    dropzoneBorderRight: {
        right: -1,
        top: -1,
        bottom: -1,
        width: 2,
        transform: "scaleY(0)",
    },
    dropzoneBorderTop: {
        left: 0,
        right: 0,
        top: -1,
        height: 2,
        transform: "scaleX(0)",
    },
    dropzoneBorderBottom: {
        left: 0,
        right: 0,
        bottom: -1,
        height: 2,
        transform: "scaleX(0)",
    },
    dropzoneBorderHorzActive: {
        transform: "scaleY(1)"
    },
    dropzoneBorderVertActive: {
        transform: "scaleX(1)"
    },
    container: {
        height: "100%"
    },
    uploadIcon: {
        verticalAlign: "middle"
    },
    deleteButton: {
        cursor: "pointer"
    },
    deleteButtonDisabled: {
        cursor: "not-allowed"
    },
    deleteIcon: {
        marginLeft: "-6px"
    }
});

interface FileUploadPropsData {
    files: UploadFile[];
    disabled: boolean;
    onDrop: (files: File[]) => void;
    onDelete: (file: UploadFile) => void;
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
        onDelete = (event: React.MouseEvent<HTMLTableCellElement>, file: UploadFile): void => {
            const { onDelete, disabled } = this.props;

            event.stopPropagation();

            if (!disabled) {
                onDelete(file);
            }
        }
        render() {
            const { classes, onDrop, disabled, files } = this.props;
            return <div className={"file-upload-dropzone " + classes.dropzoneWrapper}>
                <div className={classnames(classes.dropzoneBorder, classes.dropzoneBorderLeft, { [classes.dropzoneBorderHorzActive]: this.state.focused })} />
                <div className={classnames(classes.dropzoneBorder, classes.dropzoneBorderRight, { [classes.dropzoneBorderHorzActive]: this.state.focused })} />
                <div className={classnames(classes.dropzoneBorder, classes.dropzoneBorderTop, { [classes.dropzoneBorderVertActive]: this.state.focused })} />
                <div className={classnames(classes.dropzoneBorder, classes.dropzoneBorderBottom, { [classes.dropzoneBorderVertActive]: this.state.focused })} />
                <Dropzone className={classes.dropzone}
                    onDrop={files => onDrop(files)}
                    onClick={() => {
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
                                <Typography variant='subtitle1'>
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
                                    <TableCell>Delete</TableCell>
                                </TableRow>
                            </TableHead>
                            <TableBody>
                                {files.map(f =>
                                    <TableRow key={f.id}>
                                        <TableCell>{f.file.name}</TableCell>
                                        <TableCell>{formatFileSize(f.file.size)}</TableCell>
                                        <TableCell>{formatUploadSpeed(f.prevLoaded, f.loaded, f.prevTime, f.currentTime)}</TableCell>
                                        <TableCell>{formatProgress(f.loaded, f.total)}</TableCell>
                                        <TableCell>
                                            <IconButton
                                                aria-label="Remove"
                                                onClick={(event: React.MouseEvent<HTMLTableCellElement>) => this.onDelete(event, f)}
                                                className={disabled ? classnames(classes.deleteButtonDisabled, classes.deleteIcon) : classnames(classes.deleteButton, classes.deleteIcon)}
                                            >
                                                <RemoveIcon />
                                            </IconButton>
                                        </TableCell>
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
