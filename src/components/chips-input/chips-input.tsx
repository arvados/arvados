// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { Chips } from '~/components/chips/chips';
import { Input as MuiInput, withStyles, WithStyles } from '@material-ui/core';
import { StyleRulesCallback } from '@material-ui/core/styles';
import { InputProps } from '@material-ui/core/Input';

interface ChipsInputProps<Value> {
    value: Value[];
    getLabel?: (value: Value) => string;
    onChange: (value: Value[]) => void;
    createNewValue: (value: string) => Value;
    inputComponent?: React.ComponentType<InputProps>;
    inputProps?: InputProps;
    deletable?: boolean;
    orderable?: boolean;
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
            this.setState({ text: event.target.value });
        }

        handleKeyPress = ({ key }: React.KeyboardEvent<HTMLInputElement>) => {
            if (key === 'Enter') {
                this.createNewValue();
            } else if (key === 'Backspace') {
                this.deleteLastValue();
            }
        }

        createNewValue = () => {
            if (this.state.text) {
                const newValue = this.props.createNewValue(this.state.text);
                this.setState({ text: '' });
                this.props.onChange([...this.props.value, newValue]);
            }
        }

        deleteLastValue = () => {
            if (this.state.text.length === 0 && this.props.value.length > 0) {
                this.props.onChange(this.props.value.slice(0, -1));
            }
        }

        updateCursorPosition = () => {
            if (this.timeout) {
                clearTimeout(this.timeout);
            }
            this.timeout = setTimeout(() => this.setState({ ...this.state }));
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
            const { classes, value, ...props } = this.props;
            return <div className={classes.chips}>
                <Chips
                    {...props}
                    values={value}
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
                onKeyDown={this.handleKeyPress}
                inputProps={{
                    ...(InputProps && InputProps.inputProps),
                    className: classes.input,
                    style: this.getInputStyles(),
                }}
                fullWidth
                className={classes.inputContainer} />;
        }

        componentDidUpdate(prevProps: ChipsInputProps<Value>) {
            if (prevProps.value !== this.props.value) {
                this.updateCursorPosition();
            }
        }
        componentWillUnmount() {
            clearTimeout(this.timeout);
        }
    });
