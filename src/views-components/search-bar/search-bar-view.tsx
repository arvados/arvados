// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { compose } from 'redux';
import {
    IconButton,
    Paper,
    StyleRulesCallback,
    withStyles,
    WithStyles,
    Tooltip,
    InputAdornment, Input,
} from '@material-ui/core';
import SearchIcon from '@material-ui/icons/Search';
import ArrowDropDownIcon from '@material-ui/icons/ArrowDropDown';
import { ArvadosTheme } from 'common/custom-theme';
import { SearchView } from 'store/search-bar/search-bar-reducer';
import {
    SearchBarBasicView,
    SearchBarBasicViewDataProps,
    SearchBarBasicViewActionProps
} from 'views-components/search-bar/search-bar-basic-view';
import {
    SearchBarAutocompleteView,
    SearchBarAutocompleteViewDataProps,
    SearchBarAutocompleteViewActionProps
} from 'views-components/search-bar/search-bar-autocomplete-view';
import {
    SearchBarAdvancedView,
    SearchBarAdvancedViewDataProps,
    SearchBarAdvancedViewActionProps
} from 'views-components/search-bar/search-bar-advanced-view';
import { KEY_CODE_DOWN, KEY_CODE_ESC, KEY_CODE_UP, KEY_ENTER } from "common/codes";
import { debounce } from 'debounce';
import { Vocabulary } from 'models/vocabulary';
import { connectVocabulary } from '../resource-properties-form/property-field-common';

type CssRules = 'container' | 'containerSearchViewOpened' | 'input' | 'view';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => {
    return {
        container: {
            position: 'relative',
            width: '100%',
            borderRadius: theme.spacing.unit / 2,
            zIndex: theme.zIndex.modal,
        },
        containerSearchViewOpened: {
            position: 'relative',
            width: '100%',
            borderRadius: `${theme.spacing.unit / 2}px ${theme.spacing.unit / 2}px 0 0`,
            zIndex: theme.zIndex.modal,
        },
        input: {
            border: 'none',
            padding: `0`
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
    vocabulary?: Vocabulary;
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
    setAdvancedDataFromSearchValue: (search: string, vocabulary?: Vocabulary) => void;
}

type SearchBarViewProps = SearchBarDataProps & SearchBarActionProps & WithStyles<CssRules>;

const handleKeyDown = (e: React.KeyboardEvent, props: SearchBarViewProps) => {
    if (e.keyCode === KEY_CODE_DOWN) {
        e.preventDefault();
        if (!props.isPopoverOpen) {
            props.onSetView(SearchView.AUTOCOMPLETE);
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

const handleInputClick = (e: React.MouseEvent, props: SearchBarViewProps) => {
    if (props.searchValue) {
        props.onSetView(SearchView.AUTOCOMPLETE);
        props.openSearchView();
    } else {
        props.onSetView(SearchView.BASIC);
    }
};

const handleDropdownClick = (e: React.MouseEvent, props: SearchBarViewProps) => {
    e.stopPropagation();
    if (props.isPopoverOpen && props.currentView === SearchView.ADVANCED) {
        props.closeView();
    } else {
        props.setAdvancedDataFromSearchValue(props.searchValue, props.vocabulary);
        props.onSetView(SearchView.ADVANCED);
    }
};

export const SearchBarView = compose(connectVocabulary, withStyles(styles))(
    class extends React.Component<SearchBarViewProps> {

        debouncedSearch = debounce(() => {
            this.props.onSearch(this.props.searchValue);
        }, 1000);

        handleChange = (event: React.ChangeEvent<HTMLInputElement>) => {
            this.debouncedSearch();
            this.props.onChange(event);
        }

        handleSubmit = (event: React.FormEvent<HTMLFormElement>) => {
            this.debouncedSearch.clear();
            this.props.onSubmit(event);
        }

        componentWillUnmount() {
            this.debouncedSearch.clear();
        }

        render() {
            const { children, ...props } = this.props;
            const { classes, isPopoverOpen } = this.props;
            return (
                <>

                    {isPopoverOpen &&
                        <Backdrop onClick={props.closeView} />}

                    <Paper className={isPopoverOpen ? classes.containerSearchViewOpened : classes.container} >
                        <form onSubmit={this.handleSubmit}>
                            <Input
                                data-cy='searchbar-input-field'
                                className={classes.input}
                                onChange={this.handleChange}
                                placeholder="Search"
                                value={props.searchValue}
                                fullWidth={true}
                                disableUnderline={true}
                                onClick={e => handleInputClick(e, props)}
                                onKeyDown={e => handleKeyDown(e, props)}
                                startAdornment={
                                    <InputAdornment position="start">
                                        <Tooltip title='Search'>
                                            <IconButton type="submit">
                                                <SearchIcon />
                                            </IconButton>
                                        </Tooltip>
                                    </InputAdornment>
                                }
                                endAdornment={
                                    <InputAdornment position="end">
                                        <Tooltip title='Advanced search'>
                                            <IconButton onClick={e => handleDropdownClick(e, props)}>
                                                <ArrowDropDownIcon />
                                            </IconButton>
                                        </Tooltip>
                                    </InputAdornment>
                                } />
                        </form>
                        <div className={classes.view}>
                            {isPopoverOpen && getView({ ...props })}
                        </div>
                    </Paper >
                </>
            );
        }
    });

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
                tags={props.tags}
                saveQuery={props.saveQuery} />;
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

const Backdrop = withStyles<'backdrop'>(theme => ({
    backdrop: {
        position: 'fixed',
        top: 0,
        right: 0,
        bottom: 0,
        left: 0,
        zIndex: theme.zIndex.modal
    }
}))(
    ({ classes, ...props }: WithStyles<'backdrop'> & React.HTMLProps<HTMLDivElement>) =>
        <div className={classes.backdrop} {...props} />);
