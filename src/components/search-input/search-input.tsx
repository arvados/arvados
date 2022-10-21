// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React, {useState, useEffect} from 'react';
import { IconButton, StyleRulesCallback, withStyles, WithStyles, FormControl, InputLabel, Input, InputAdornment, Tooltip } from '@material-ui/core';
import SearchIcon from '@material-ui/icons/Search';

type CssRules = 'container' | 'input' | 'button';

const styles: StyleRulesCallback<CssRules> = theme => {
    return {
        container: {
            position: 'relative',
            width: '100%'
        },
        input: {
            border: 'none',
            borderRadius: theme.spacing.unit / 4,
            boxSizing: 'border-box',
            padding: theme.spacing.unit,
            paddingRight: theme.spacing.unit * 4,
            width: '100%',
        },
        button: {
            position: 'absolute',
            top: theme.spacing.unit / 2,
            right: theme.spacing.unit / 2,
            width: theme.spacing.unit * 3,
            height: theme.spacing.unit * 3
        }
    };
};

interface SearchInputDataProps {
    value: string;
    label?: string;
    selfClearProp: string;
}

interface SearchInputActionProps {
    onSearch: (value: string) => any;
    debounce?: number;
}

type SearchInputProps = SearchInputDataProps & SearchInputActionProps & WithStyles<CssRules>;

export const DEFAULT_SEARCH_DEBOUNCE = 1000;

const SearchInputComponent = (props: SearchInputProps) => {
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
    }, [props.value, props.label]); // eslint-disable-line react-hooks/exhaustive-deps

    useEffect(() => {
        if (selfClearProp !== props.selfClearProp) {
            setValue("");
            setSelfClearProp(props.selfClearProp);
            handleChange({ target: { value: "" } } as any);
        }
    }, [props.selfClearProp]); // eslint-disable-line react-hooks/exhaustive-deps

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

    return <form onSubmit={handleSubmit}>
        <FormControl>
            <InputLabel>{label}</InputLabel>
            <Input
                type="text"
                data-cy="search-input"
                value={value}
                onChange={handleChange}
                endAdornment={
                    <InputAdornment position="end">
                        <Tooltip title='Search'>
                            <IconButton
                                onClick={handleSubmit}>
                                <SearchIcon />
                            </IconButton>
                        </Tooltip>
                    </InputAdornment>
                } />
        </FormControl>
    </form>;
}

export const SearchInput = withStyles(styles)(SearchInputComponent);