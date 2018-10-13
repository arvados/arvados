// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { Chips } from '~/components/chips/chips';
import { Input, withStyles, WithStyles } from '@material-ui/core';
import { StyleRulesCallback } from '@material-ui/core/styles';

interface ChipsInputProps<Value> {
    values: Value[];
    getLabel?: (value: Value) => string;
    onChange: (value: Value[]) => void;
    createNewValue: (value: string) => Value;
}

type CssRules = 'chips' | 'input' | 'inputContainer';

const styles: StyleRulesCallback = () => ({
    chips: {
        minHeight: '40px',
        zIndex: 1,
        position: 'relative',
    },
    input: {
        position: 'relative',
        top: '-5px',
        zIndex: 1,
    },
    inputContainer: {
        top: '-24px',
    },
});

export const ChipsInput = withStyles(styles)(
    class ChipsInput<Value> extends React.Component<ChipsInputProps<Value> & WithStyles<CssRules>> {

        state = {
            text: '',
        };

        filler = React.createRef<HTMLDivElement>();
        timeout = -1;

        componentWillUnmount (){
            clearTimeout(this.timeout);
        }

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
                this.props.onChange([...this.props.values, newValue]);
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
            this.timeout = setTimeout(() => this.forceUpdate());
        }

        render() {
            this.updateCursorPosition();
            return <>
                <div className={this.props.classes.chips}>
                    <Chips {...this.props} filler={<div ref={this.filler} />} />
                </div>
                <Input
                    value={this.state.text}
                    onChange={this.setText}
                    onKeyDown={this.handleKeyPress}
                    inputProps={{
                        className: this.props.classes.input,
                        style: this.getInputStyles(),
                    }}
                    fullWidth
                    className={this.props.classes.inputContainer} />
            </>;
        }

        getInputStyles = (): React.CSSProperties => ({
            width: this.filler.current
                ? this.filler.current.offsetWidth + 8
                : '100%',
            right: this.filler.current
                ? `calc(${this.filler.current.offsetWidth}px - 100%)`
                : 0,

        })
    });
