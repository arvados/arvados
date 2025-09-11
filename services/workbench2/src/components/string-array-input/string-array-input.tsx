// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0


import React, { useState, KeyboardEvent, ChangeEvent, useEffect } from 'react';
import { TextField, Chip, Box, IconButton } from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import { WrappedFieldProps } from 'redux-form';

interface StringArrayInputProps extends WrappedFieldProps {
    label?: string;
}

export const StringArrayInput = ({ input, label, meta }: StringArrayInputProps) => {
    const [currentValue, setCurrentValue] = useState<string>('');
    const [error, setError] = useState<string | undefined>(undefined);
    const [touched, setTouched] = useState(false);

    // Update error state when meta.error changes
    useEffect(() => {
        setError(meta.error);
    }, [meta.error]);

    const handleKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
        if (e.key === 'Enter' && currentValue.trim()) {
            e.preventDefault();
            addToChips((input.value as string[]), currentValue.trim());
        }
    };

    const handleDelete = (chipToDelete: string) => {
        const newValues = ((input.value as string[]) || []).filter((chip) => chip !== chipToDelete);
        input.onChange(newValues);
    };

    const handleAddClick = () => {
        if (currentValue.trim()) {
            addToChips((input.value as string[]), currentValue.trim());
        }
    };

    const addToChips = (currentValues: string[], newValue: string) => {
        if (duplicateValueError(currentValues, newValue)) {
            return;
        }
        const newChips = [...currentValues, newValue];
        input.onChange(newChips);
        setCurrentValue('');
        setError(undefined);
    };

    const duplicateValueError = (currentValues: string[], newValue: string) => {
        if (currentValues.includes(newValue)) {
            const errorMsg = `Value "${newValue}" already exists`;
            setError(errorMsg);
            setTouched(true);
            return true;
        }
        return false;
    };

    return (
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
            <TextField
                label={label}
                value={currentValue}
                onFocus={() => setTouched(true)}
                onBlur={() => setTouched(false)}
                onChange={(e: ChangeEvent<HTMLInputElement>) => {
                    setCurrentValue(e.target.value);
                    setError(undefined);
                }}
                onKeyDown={handleKeyDown}
                InputProps={{
                    endAdornment: (
                        <IconButton
                            onClick={handleAddClick}
                            edge='end'
                        >
                            <AddIcon />
                        </IconButton>
                    ),
                }}
                error={Boolean(touched && error)}
                helperText={touched && error ? error : ''}
            />

            <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1 }}>
                {((input.value as string[]) || []).map((val, idx) => (
                    <Chip
                        key={idx}
                        label={val}
                        onDelete={() => handleDelete(val)}
                    />
                ))}
            </Box>
        </Box>
    );
};
