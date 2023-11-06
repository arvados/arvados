// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { Chips } from 'components/chips/chips';
import { Input as MuiInput, withStyles, WithStyles } from '@material-ui/core';
import { StyleRulesCallback } from '@material-ui/core/styles';
import { InputProps } from '@material-ui/core/Input';

interface ChipsInputProps<Value> {
    values: Value[];
    getLabel?: (value: Value) => string;
    onChange: (value: Value[]) => void;
    onPartialInput?: (value: boolean) => void;
    handleFocus?: (e: any) => void;
    handleBlur?: (e: any) => void;
    chipsClassName?: string;
    createNewValue: (value: string) => Value;
    inputComponent?: React.ComponentType<InputProps>;
    inputProps?: InputProps;
    deletable?: boolean;
    orderable?: boolean;
    disabled?: boolean;
    pattern?: RegExp;
}

type CssRules = 'chips' | 'input' | 'inputContainer';

const styles: StyleRulesCallback = ({ spacing }) => ({
    chips: {
        minHeight: spacing.unit * 5,
        zIndex: 1,
        position: 'relative',
    },
    input: {
        zIndex: 1,
        marginBottom: 8,
        position: 'relative',
    },
    inputContainer: {
        marginTop: -34
    },
});

export const ChipsInput = withStyles(styles)(
    class ChipsInput<Value> extends React.Component<ChipsInputProps<Value> & WithStyles<CssRules>> {

        state = {
            text: '',
        };

        filler = React.createRef<HTMLDivElement>();
        timeout = -1;

        setText = (event: React.ChangeEvent<HTMLInputElement>) => {
            this.setState({ text: event.target.value }, () => {
                // Update partial input status
                this.props.onPartialInput && this.props.onPartialInput(this.state.text !== '');

                // If pattern is provided, check for delimiter
                if (this.props.pattern) {
                    const matches = this.state.text.match(this.props.pattern);
                    // Only create values if 1 match and the last character is a delimiter
                    //   (user pressed an invalid character at the end of a token)
                    //   or if multiple matches (user pasted text)
                    if (matches &&
                            (
                                matches.length > 1 ||
                                (matches.length === 1 && !this.state.text.endsWith(matches[0]))
                            )) {
                        this.createNewValue(matches.map((i) => i));
                    }
                }
            });
        }

        handleKeyPress = (e: React.KeyboardEvent<HTMLInputElement>) => {
            // Handle special keypresses
            if (e.key === 'Enter') {
                this.createNewValue();
                e.preventDefault();
            } else if (e.key === 'Backspace') {
                this.deleteLastValue();
            }
        }

        createNewValue = (matches?: string[]) => {
            if (this.state.text) {
                if (matches && matches.length > 0) {
                    const newValues = matches.map((v) => this.props.createNewValue(v));
                    this.setState({ text: '' });
                    this.props.onChange([...this.props.values, ...newValues]);
                } else {
                    const newValue = this.props.createNewValue(this.state.text);
                    this.setState({ text: '' });
                    this.props.onChange([...this.props.values, newValue]);
                }
                this.props.onPartialInput && this.props.onPartialInput(false);
            }
        }

        deleteLastValue = () => {
            if (this.state.text.length === 0 && this.props.values.length > 0) {
                this.props.onChange(this.props.values.slice(0, -1));
            }
        }

        updateCursorPosition = () => {
            if (this.timeout) {
                clearTimeout(this.timeout);
            }
            this.timeout = window.setTimeout(() => this.setState({ ...this.state }));
        }

        getInputStyles = (): React.CSSProperties => ({
            width: this.filler.current
                ? this.filler.current.offsetWidth
                : '100%',
            right: this.filler.current
                ? `calc(${this.filler.current.offsetWidth}px - 100%)`
                : 0,

        })

        componentDidMount() {
            this.updateCursorPosition();
        }

        render() {
            return <>
                {this.renderChips()}
                {this.renderInput()}
            </>;
        }

        renderChips() {
            const { classes, ...props } = this.props;
            return <div className={[classes.chips, this.props.chipsClassName].join(' ')}>
                <Chips
                    {...props}
                    clickable={!props.disabled}
                    filler={<div ref={this.filler} />}
                />
            </div>;
        }

        renderInput() {
            const { inputProps: InputProps, inputComponent: Input = MuiInput, classes } = this.props;
            return <Input
                {...InputProps}
                value={this.state.text}
                onChange={this.setText}
                disabled={this.props.disabled}
                onKeyDown={this.handleKeyPress}
                onFocus={this.props.handleFocus}
                onBlur={this.props.handleBlur}
                inputProps={{
                    ...(InputProps && InputProps.inputProps),
                    className: classes.input,
                    style: this.getInputStyles(),
                }}
                fullWidth
                className={classes.inputContainer} />;
        }

        componentDidUpdate(prevProps: ChipsInputProps<Value>) {
            if (prevProps.values !== this.props.values) {
                this.updateCursorPosition();
            }
        }
        componentWillUnmount() {
            clearTimeout(this.timeout);
        }
    });
