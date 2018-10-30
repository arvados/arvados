// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import {
    IconButton,
    Paper,
    StyleRulesCallback,
    withStyles,
    WithStyles,
    Tooltip,
    InputAdornment, Input,
    ClickAwayListener
} from '@material-ui/core';
import SearchIcon from '@material-ui/icons/Search';
import { ArvadosTheme } from '~/common/custom-theme';
import { SearchView } from '~/store/search-bar/search-bar-reducer';
import {
    SearchBarBasicView,
    SearchBarBasicViewDataProps,
    SearchBarBasicViewActionProps
} from '~/views-components/search-bar/search-bar-basic-view';
import {
    SearchBarAutocompleteView,
    SearchBarAutocompleteViewDataProps,
    SearchBarAutocompleteViewActionProps
} from '~/views-components/search-bar/search-bar-autocomplete-view';
import {
    SearchBarAdvancedView,
    SearchBarAdvancedViewDataProps,
    SearchBarAdvancedViewActionProps
} from '~/views-components/search-bar/search-bar-advanced-view';
import { KEY_CODE_DOWN, KEY_CODE_ESC, KEY_CODE_UP, KEY_ENTER } from "~/common/codes";

type CssRules = 'container' | 'containerSearchViewOpened' | 'input' | 'view';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => {
    return {
        container: {
            position: 'relative',
            width: '100%',
            borderRadius: theme.spacing.unit / 2
        },
        containerSearchViewOpened: {
            position: 'relative',
            width: '100%',
            borderRadius: `${theme.spacing.unit / 2}px ${theme.spacing.unit / 2}px 0 0`
        },
        input: {
            border: 'none',
            padding: `0px ${theme.spacing.unit}px`
        },
        view: {
            position: 'absolute',
            width: '100%',
            zIndex: 1
        }
    };
};

export type SearchBarDataProps = SearchBarViewDataProps
    & SearchBarAutocompleteViewDataProps
    & SearchBarAdvancedViewDataProps
    & SearchBarBasicViewDataProps;

interface SearchBarViewDataProps {
    searchValue: string;
    currentView: string;
    isPopoverOpen: boolean;
    debounce?: number;
}

export type SearchBarActionProps = SearchBarViewActionProps
    & SearchBarAutocompleteViewActionProps
    & SearchBarAdvancedViewActionProps
    & SearchBarBasicViewActionProps;

interface SearchBarViewActionProps {
    onChange: (event: React.ChangeEvent<HTMLInputElement>) => void;
    onSubmit: (event: React.FormEvent<HTMLFormElement>) => void;
    onSetView: (currentView: string) => void;
    closeView: () => void;
    openSearchView: () => void;
    loadRecentQueries: () => string[];
    moveUp: () => void;
    moveDown: () => void;
}

type SearchBarViewProps = SearchBarDataProps & SearchBarActionProps & WithStyles<CssRules>;

const handleKeyDown = (e: React.KeyboardEvent, props: SearchBarViewProps) => {
    if (e.keyCode === KEY_CODE_DOWN) {
        e.preventDefault();
        if (!props.isPopoverOpen) {
            props.openSearchView();
        } else {
            props.moveDown();
        }
    } else if (e.keyCode === KEY_CODE_UP) {
        e.preventDefault();
        props.moveUp();
    } else if (e.keyCode === KEY_CODE_ESC) {
        e.preventDefault();
        props.closeView();
    } else if (e.keyCode === KEY_ENTER) {
        if (props.currentView === SearchView.BASIC) {
            e.preventDefault();
            props.onSearch(props.selectedItem.query);
        } else if (props.currentView === SearchView.AUTOCOMPLETE) {
            if (props.selectedItem.id !== props.searchValue) {
                e.preventDefault();
                props.navigateTo(props.selectedItem.id);
            }
        }
    }
};

export const SearchBarView = withStyles(styles)(
    (props : SearchBarViewProps) => {
        const { classes, isPopoverOpen } = props;
        return (
            <ClickAwayListener onClickAway={props.closeView}>
                <Paper className={isPopoverOpen ? classes.containerSearchViewOpened : classes.container} >
                    <form onSubmit={props.onSubmit}>
                        <Input
                            className={classes.input}
                            onChange={props.onChange}
                            placeholder="Search"
                            value={props.searchValue}
                            fullWidth={true}
                            disableUnderline={true}
                            onClick={props.openSearchView}
                            onKeyDown={e => handleKeyDown(e, props)}
                            endAdornment={
                                <InputAdornment position="end">
                                    <Tooltip title='Search'>
                                        <IconButton type="submit">
                                            <SearchIcon />
                                        </IconButton>
                                    </Tooltip>
                                </InputAdornment>
                            } />
                    </form>
                    <div className={classes.view}>
                        {isPopoverOpen && getView({...props})}
                    </div>
                </Paper >
            </ClickAwayListener>
        );
    }
);

const getView = (props: SearchBarViewProps) => {
    switch (props.currentView) {
        case SearchView.AUTOCOMPLETE:
            return <SearchBarAutocompleteView
                navigateTo={props.navigateTo}
                searchResults={props.searchResults}
                searchValue={props.searchValue}
                selectedItem={props.selectedItem} />;
        case SearchView.ADVANCED:
            return <SearchBarAdvancedView
                closeAdvanceView={props.closeAdvanceView}
                tags={props.tags} />;
        default:
            return <SearchBarBasicView
                onSetView={props.onSetView}
                onSearch={props.onSearch}
                loadRecentQueries={props.loadRecentQueries}
                savedQueries={props.savedQueries}
                deleteSavedQuery={props.deleteSavedQuery}
                editSavedQuery={props.editSavedQuery}
                selectedItem={props.selectedItem} />;
    }
};
