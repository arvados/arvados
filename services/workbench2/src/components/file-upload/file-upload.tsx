// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import classnames from 'classnames';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import {
    Grid,
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableRow,
    Typography,
    IconButton,
} from '@mui/material';
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { CloudUploadIcon, RemoveIcon } from "../icon/icon";
import { formatFileSize, formatProgress, formatUploadSpeed } from "common/formatters";
import { UploadFile } from 'store/file-uploader/file-uploader-actions';
import { UploadInput, FileUploadType } from 'components/file-upload/upload-input';

type CssRules = "dropzoneWrapper" | "container" | "inputContainer" | "uploadIcon"
    | "dropzoneBorder" | "dropzoneBorderLeft" | "dropzoneBorderRight" | "dropzoneBorderTop" | "dropzoneBorderBottom"
    | "dropzoneBorderHorzActive" | "dropzoneBorderVertActive" | "deleteButton" | "deleteButtonDisabled" | "deleteIcon";

const styles: CustomStyleRulesCallback<CssRules> = theme => ({
    dropzoneWrapper: {
        width: "100%",
        height: "200px",
        position: "relative",
        border: "1px solid rgba(0, 0, 0, 0.42)",
        boxSizing: 'border-box',
        overflowY: "scroll",
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
        height: "100%",
        padding: '16px',
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
    },
    inputContainer: {
        width: '80%',
        marginTop: '1rem',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-around',
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
        onDelete = (event: React.MouseEvent<HTMLButtonElement>, file: UploadFile): void => {
            const { onDelete, disabled } = this.props;

            event.stopPropagation();

            if (!disabled) {
                onDelete(file);
            }

            let interval = setInterval(() => {
                const key = Object.keys((window as any).cancelTokens).find(key => key.indexOf(file.file.name) > -1);

                if (key) {
                    clearInterval(interval);
                    (window as any).cancelTokens[key]();
                    delete (window as any).cancelTokens[key];
                }
            }, 100);

        }

        fileInputRef = React.createRef<HTMLInputElement>();
        folderInputRef = React.createRef<HTMLInputElement>();

        handleDrop = async (event) => {
            event.preventDefault();

            const items = event.dataTransfer.items;
            const entries: any[] = [];

            for (let i = 0; i < items.length; i++) {
                const entry = items[i].webkitGetAsEntry?.();
                if (entry) entries.push(entry);
            }

            const filesArrays = await Promise.all(entries.map((entry) => traverseFileTree(entry)));
            const allFiles = filesArrays.flat();

            this.props.onDrop(allFiles as any); // includes `file.relativePath` if needed
        }

        handleInputChange = (event) => {
                const files = Array.from(event.target.files);
                this.props.onDrop(files as any);
            };

        getInputProps = () => ({
            disabled: this.props.disabled,
            handleInputChange: this.handleInputChange,
            onFocus: ()=>this.setState({ focused: true }),
            onBlur: ()=>this.setState({ focused: false }),
        })

        render() {
            const { classes, disabled, files } = this.props;
            return (
                <div className={"file-upload-dropzone " + classes.dropzoneWrapper} >
                    <div className={classnames(classes.dropzoneBorder, classes.dropzoneBorderLeft, { [classes.dropzoneBorderHorzActive]: this.state.focused })} />
                    <div className={classnames(classes.dropzoneBorder, classes.dropzoneBorderRight, { [classes.dropzoneBorderHorzActive]: this.state.focused })} />
                    <div className={classnames(classes.dropzoneBorder, classes.dropzoneBorderTop, { [classes.dropzoneBorderVertActive]: this.state.focused })} />
                    <div className={classnames(classes.dropzoneBorder, classes.dropzoneBorderBottom, { [classes.dropzoneBorderVertActive]: this.state.focused })} />
                    <div
                        onDrop={this.handleDrop}
                        onDragOver={(e) => e.preventDefault()}
                        onClick={() => {
                            const el = document.getElementsByClassName("file-upload-dropzone")[0];
                            const inputs = el.getElementsByTagName("input");
                            if (inputs.length > 0) {
                                inputs[0].focus();
                            }
                        }}
                        data-cy="drag-and-drop"
                        >
                        {files.length === 0 &&
                            <Grid container justifyContent="center" alignItems="center" className={classes.container}>
                                <Grid item component={"span"}>
                                    <Typography variant='subtitle1'>
                                        <CloudUploadIcon className={classes.uploadIcon} /> Drag and drop data or click to browse
                                    </Typography>
                                </Grid>
                                <Grid item component={"div"} className={classes.inputContainer}>
                                    <UploadInput type={FileUploadType.FOLDER} inputRef={this.folderInputRef} {...this.getInputProps()} />
                                    <UploadInput type={FileUploadType.FILE} inputRef={this.fileInputRef} {...this.getInputProps()} />
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
                                                    onClick={(event: React.MouseEvent<HTMLButtonElement, MouseEvent>) => this.onDelete(event, f)}
                                                    className={disabled ? classnames(classes.deleteButtonDisabled, classes.deleteIcon) : classnames(classes.deleteButton, classes.deleteIcon)}
                                                    size="large">
                                                    <RemoveIcon />
                                                </IconButton>
                                            </TableCell>
                                        </TableRow>
                                    )}
                                </TableBody>
                            </Table>
                        }
                    </div>
                </div>
            );
        }
    }
);

function traverseFileTree(item, path = '') {
    return new Promise((resolve) => {
        if (item.isFile) {
            item.file((file) => {
                file.relativePath = path + file.name;
                resolve([file]);
            });
        } else if (item.isDirectory) {
            const dirReader = item.createReader();
            dirReader.readEntries(async (entries) => {
                const files = await Promise.all(entries.map((entry) => traverseFileTree(entry, path + item.name + '/')));
                resolve(files.flat());
            });
        }
    });
}
