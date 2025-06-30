// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { Box, Typography, IconButton } from '@mui/material';
import DriveFolderUploadIcon from '@mui/icons-material/DriveFolderUpload';
import UploadFileIcon from '@mui/icons-material/UploadFile';
import { ArvadosTheme } from 'common/custom-theme';

type CssRules = 'label' | 'icon';

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    label: {
        cursor: 'pointer',
    },
    icon: {
        color: theme.customs.colors.grey900,
    },
});

export enum FileUploadType {
    FOLDER = 'folder',
    FILE = 'file',
}

export type UploadInputProps = {
    type: FileUploadType;
    disabled: boolean;
    inputRef: React.RefObject<HTMLInputElement>;
    handleInputChange: (event: React.ChangeEvent<HTMLInputElement>) => void;
    onFocus: () => void;
    onBlur: () => void;
};

export const UploadInput = withStyles(styles)(({ type, disabled, inputRef, handleInputChange, onFocus, onBlur, classes }: UploadInputProps & WithStyles<CssRules>) => {
    return (
        <label className={classes.label}>
            <Box
                display='flex'
                flexDirection='column'
                alignItems='center'
                gap={1}
            >
                <IconButton
                    component='span'
                    sx={{
                        width: 80,
                        height: 80,
                        borderRadius: 2,
                        bgcolor: 'grey.100',
                        '&:hover': { bgcolor: 'grey.200' },
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                    }}
                >
                    {type === FileUploadType.FOLDER ? <DriveFolderUploadIcon fontSize='large' className={classes.icon} /> : <UploadFileIcon fontSize='large' className={classes.icon} />}
                </IconButton>
                <Typography variant='body2'>{type === FileUploadType.FOLDER ? 'Upload Folder' : 'Upload Files'}</Typography>
                {type === FileUploadType.FOLDER ? (
                    <input
                        type='file'
                        ref={inputRef}
                        disabled={disabled}
                        onChange={handleInputChange}
                        onFocus={onFocus}
                        onBlur={onBlur}
                        multiple
                        hidden
                        {...({ webkitDirectory: 'true', directory: 'true' } as any)}
                    />
                ) : (
                    <input
                        type='file'
                        ref={inputRef}
                        disabled={disabled}
                        onChange={handleInputChange}
                        onFocus={onFocus}
                        onBlur={onBlur}
                        multiple
                        hidden
                    />
                )}
            </Box>
        </label>
    );
});
