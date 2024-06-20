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

interface SearchInputDataProps {
    value: string;
    label?: string;
    selfClearProp: string;
}

interface SearchInputActionProps {
    onSearch: (value: string) => any;
    debounce?: number;
}

type SearchInputProps = SearchInputDataProps & SearchInputActionProps;

export const DEFAULT_SEARCH_DEBOUNCE = 1000;

export const SearchInput = (props: SearchInputProps) => {
    const [timeout, setTimeout] = useState(0);
    const [value, setValue] = useState("");
    const [label, setLabel] = useState("Search");
    const [selfClearProp, setSelfClearProp] = useState("");

    useEffect(() => {
        if (props.value) {
            setValue(props.value);
        }
        if (props.label) {
            setLabel(props.label);
        }

        return () => {
            setValue("");
            clearTimeout(timeout);
        };
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [props.value, props.label]); 

    useEffect(() => {
        if (selfClearProp !== props.selfClearProp) {
            setValue("");
            setSelfClearProp(props.selfClearProp);
            handleChange({ target: { value: "" } } as any);
        }
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [props.selfClearProp]); 

    const handleSubmit = (event: React.FormEvent<HTMLElement>) => {
        event.preventDefault();
        clearTimeout(timeout);
        props.onSearch(value);
    };

    const handleChange = (event: React.ChangeEvent<HTMLInputElement>) => {
        const { target: { value: eventValue } } = event;
        clearTimeout(timeout);
        setValue(eventValue);

        setTimeout(window.setTimeout(
            () => {
                props.onSearch(eventValue);
            },
            props.debounce || DEFAULT_SEARCH_DEBOUNCE
        ));
    };

    return (
        <form onSubmit={handleSubmit}>
            <FormControl variant="standard" style={{ width: '14rem'}}>
                <InputLabel>{label}</InputLabel>
                <Input
                    type="text"
                    data-cy="search-input"
                    value={value}
                    onChange={handleChange}
                    endAdornment={
                        <InputAdornment position="end">
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
};
