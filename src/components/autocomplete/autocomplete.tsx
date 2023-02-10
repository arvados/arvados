// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import {
    Input as MuiInput,
    Chip as MuiChip,
    Popper as MuiPopper,
    Paper as MuiPaper,
    FormControl, InputLabel, StyleRulesCallback, withStyles, RootRef, ListItemText, ListItem, List, FormHelperText, Tooltip
} from '@material-ui/core';
import { PopperProps } from '@material-ui/core/Popper';
import { WithStyles } from '@material-ui/core/styles';
import { noop } from 'lodash';

export interface AutocompleteProps<Item, Suggestion> {
    label?: string;
    value: string;
    items: Item[];
    disabled?: boolean;
    suggestions?: Suggestion[];
    error?: boolean;
    helperText?: string;
    autofocus?: boolean;
    onChange: (event: React.ChangeEvent<HTMLInputElement>) => void;
    onBlur?: (event: React.FocusEvent<HTMLInputElement>) => void;
    onFocus?: (event: React.FocusEvent<HTMLInputElement>) => void;
    onCreate?: () => void;
    onDelete?: (item: Item, index: number) => void;
    onSelect?: (suggestion: Suggestion) => void;
    renderChipValue?: (item: Item) => string;
    renderChipTooltip?: (item: Item) => string;
    renderSuggestion?: (suggestion: Suggestion) => React.ReactNode;
}

export interface AutocompleteState {
    suggestionsOpen: boolean;
    selectedSuggestionIndex: number;
}

export class Autocomplete<Value, Suggestion> extends React.Component<AutocompleteProps<Value, Suggestion>, AutocompleteState> {

    state = {
        suggestionsOpen: false,
        selectedSuggestionIndex: 0,
    };

    containerRef = React.createRef<HTMLDivElement>();
    inputRef = React.createRef<HTMLInputElement>();

    render() {
        return (
            <RootRef rootRef={this.containerRef}>
                <FormControl fullWidth error={this.props.error}>
                    {this.renderLabel()}
                    {this.renderInput()}
                    {this.renderHelperText()}
                    {this.renderSuggestions()}
                </FormControl>
            </RootRef>
        );
    }

    renderLabel() {
        const { label } = this.props;
        return label && <InputLabel>{label}</InputLabel>;
    }

    renderInput() {
        return <Input
            disabled={this.props.disabled}
            autoFocus={this.props.autofocus}
            inputRef={this.inputRef}
            value={this.props.value}
            startAdornment={this.renderChips()}
            onFocus={this.handleFocus}
            onBlur={this.handleBlur}
            onChange={this.props.onChange}
            onKeyPress={this.handleKeyPress}
            onKeyDown={this.handleNavigationKeyPress}
        />;
    }

    renderHelperText() {
        return <FormHelperText>{this.props.helperText}</FormHelperText>;
    }

    renderSuggestions() {
        const { suggestions = [] } = this.props;
        return (
            <Popper
                open={this.isSuggestionBoxOpen()}
                anchorEl={this.inputRef.current}
                key={suggestions.length}>
                <Paper onMouseDown={this.preventBlur}>
                    <List dense style={{ width: this.getSuggestionsWidth() }}>
                        {suggestions.map(
                            (suggestion, index) =>
                                <ListItem
                                    button
                                    key={index}
                                    onClick={this.handleSelect(suggestion)}
                                    selected={index === this.state.selectedSuggestionIndex}>
                                    {this.renderSuggestion(suggestion)}
                                </ListItem>
                        )}
                    </List>
                </Paper>
            </Popper>
        );
    }

    isSuggestionBoxOpen() {
        const { suggestions = [] } = this.props;
        return this.state.suggestionsOpen && suggestions.length > 0;
    }

    handleFocus = (event: React.FocusEvent<HTMLInputElement>) => {
        const { onFocus = noop } = this.props;
        this.setState({ suggestionsOpen: true });
        onFocus(event);
    }

    handleBlur = (event: React.FocusEvent<HTMLInputElement>) => {
        setTimeout(() => {
            const { onBlur = noop } = this.props;
            this.setState({ suggestionsOpen: false });
            onBlur(event);
        });
    }

    handleKeyPress = (event: React.KeyboardEvent<HTMLInputElement>) => {
        const { onCreate = noop, onSelect = noop, suggestions = [] } = this.props;
        const { selectedSuggestionIndex } = this.state;
        if (event.key === 'Enter') {
            if (this.isSuggestionBoxOpen() && selectedSuggestionIndex < suggestions.length) {
                // prevent form submissions when selecting a suggestion
                event.preventDefault();
                onSelect(suggestions[selectedSuggestionIndex]);
            } else if (this.props.value.length > 0) {
                onCreate();
            }
        }
    }

    handleNavigationKeyPress = ({ key }: React.KeyboardEvent<HTMLInputElement>) => {
        if (key === 'ArrowUp') {
            this.updateSelectedSuggestionIndex(-1);
        } else if (key === 'ArrowDown') {
            this.updateSelectedSuggestionIndex(1);
        }
    }

    updateSelectedSuggestionIndex(value: -1 | 1) {
        const { suggestions = [] } = this.props;
        this.setState(({ selectedSuggestionIndex }) => ({
            selectedSuggestionIndex: (selectedSuggestionIndex + value) % suggestions.length
        }));
    }

    renderChips() {
        const { items, onDelete } = this.props;

        /**
         * If input startAdornment prop is not undefined, input's label will stay above the input.
         * If there is not items, we want the label to go back to placeholder position.
         * That why we return without a value instead of returning a result of a _map_ which is an empty array.
         */
        if (items.length === 0) {
            return;
        }

        return items.map(
            (item, index) => {
                const tooltip = this.props.renderChipTooltip ? this.props.renderChipTooltip(item) : '';
                if (tooltip.length) {
                    return <Tooltip title={tooltip}>
                        <Chip
                            label={this.renderChipValue(item)}
                            key={index}
                            onDelete={onDelete && !this.props.disabled ? (() =>  onDelete(item, index)) : undefined} />
                    </Tooltip>
                } else {
                    return <Chip
                        label={this.renderChipValue(item)}
                        key={index}
                        onDelete={onDelete && !this.props.disabled ? (() =>  onDelete(item, index)) : undefined} />
                }
            }
        );
    }

    renderChipValue(value: Value) {
        const { renderChipValue } = this.props;
        return renderChipValue ? renderChipValue(value) : JSON.stringify(value);
    }

    preventBlur = (event: React.MouseEvent<HTMLElement>) => {
        event.preventDefault();
    }

    handleClickAway = (event: React.MouseEvent<HTMLElement>) => {
        if (event.target !== this.inputRef.current) {
            this.setState({ suggestionsOpen: false });
        }
    }

    handleSelect(suggestion: Suggestion) {
        return () => {
            const { onSelect = noop } = this.props;
            const { current } = this.inputRef;
            if (current) {
                current.focus();
            }
            onSelect(suggestion);
        };
    }

    renderSuggestion(suggestion: Suggestion) {
        const { renderSuggestion } = this.props;
        return renderSuggestion
            ? renderSuggestion(suggestion)
            : <ListItemText>{JSON.stringify(suggestion)}</ListItemText>;
    }

    getSuggestionsWidth() {
        return this.containerRef.current ? this.containerRef.current.offsetWidth : 'auto';
    }
}

type ChipClasses = 'root';

const chipStyles: StyleRulesCallback<ChipClasses> = theme => ({
    root: {
        marginRight: theme.spacing.unit / 4,
        height: theme.spacing.unit * 3,
    }
});

const Chip = withStyles(chipStyles)(MuiChip);

type PopperClasses = 'root';

const popperStyles: StyleRulesCallback<ChipClasses> = theme => ({
    root: {
        zIndex: theme.zIndex.modal,
    }
});

const Popper = withStyles(popperStyles)(
    ({ classes, ...props }: PopperProps & WithStyles<PopperClasses>) =>
        <MuiPopper {...props} className={classes.root} />
);

type InputClasses = 'root';

const inputStyles: StyleRulesCallback<InputClasses> = () => ({
    root: {
        display: 'flex',
        flexWrap: 'wrap',
    },
    input: {
        minWidth: '20%',
        flex: 1,
    },
});

const Input = withStyles(inputStyles)(MuiInput);

const Paper = withStyles({
    root: {
        maxHeight: '80vh',
        overflowY: 'auto',
    }
})(MuiPaper);
