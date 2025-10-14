// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React, {useState, useEffect} from 'react';
import {
    IconButton,
    FormControl,
    InputLabel,
    Input,
    InputAdornment,
    Tooltip,
} from '@mui/material';
import SearchIcon from '@mui/icons-material/Search';
import { connect } from 'react-redux';
import { RootState } from 'store/store';

interface SearchInputDataProps {
    currentPath: string;
    value?: string;
    label?: string;
    width?: string;
}

interface SearchInputActionProps {
    onSearch: (value: string) => any;
    debounce?: number;
}

type SearchInputProps = SearchInputDataProps & SearchInputActionProps;

export const DEFAULT_SEARCH_DEBOUNCE = 750;

export const SearchInput = connect((state: RootState) => ({currentPath: state.router.location?.pathname}))(
    (props: SearchInputProps) => {
    const [searchTimeout, setSearchTimeout] = useState<NodeJS.Timeout | undefined>(undefined);
    const [value, setValue] = useState("");

    useEffect(() => {
        if (props.value && props.value !== value) setValue(props.value);
        return () => {
            if (searchTimeout) clearTimeout(searchTimeout);
        };
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

    useEffect(() => {
        setValue('')
        if (searchTimeout) clearTimeout(searchTimeout);
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [props.currentPath]);

    useEffect(() => {
        if (searchTimeout) clearTimeout(searchTimeout);
        setSearchTimeout(setTimeout(() => {
            props.onSearch(value);
        }, props.debounce || DEFAULT_SEARCH_DEBOUNCE));
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [value]);

    const handleSubmit = (event: React.FormEvent<HTMLElement>) => {
        event.preventDefault();
        props.onSearch(value);
    };

    return (
        <form onSubmit={handleSubmit}>
            <FormControl variant="standard" style={{ width: props.width || '14rem', marginTop: '-10px'}}>
                <InputLabel>{props.label || 'Search'}</InputLabel>
                <Input
                    type="text"
                    data-cy="search-input"
                    value={value}
                    onChange={(ev) => setValue(ev.target.value)}
                    endAdornment={
                        <InputAdornment position="end" style={{marginRight: '-0.6rem'}}>
                            <Tooltip title='Search'>
                                <IconButton onClick={handleSubmit} size="large">
                                    <SearchIcon />
                                </IconButton>
                            </Tooltip>
                        </InputAdornment>
                    } />
            </FormControl>
        </form>
    );
});
