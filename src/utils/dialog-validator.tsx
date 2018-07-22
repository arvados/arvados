// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core';

type CssRules = "formInputError";

const styles: StyleRulesCallback<CssRules> = theme => ({
    formInputError: {
        color: "#ff0000",
        marginLeft: "5px",
        fontSize: "11px",
    }
});

type ValidatorProps = {
    value: string,
    onChange: (isValid: boolean | string) => void;
    render: (hasError: boolean) => React.ReactElement<any>;
    isRequired: boolean;
};

interface ValidatorState {
    isPatternValid: boolean;
    isLengthValid: boolean;
}

const nameRegEx = /^[a-zA-Z0-9-_ ]+$/;
const maxInputLength = 60;

export const Validator = withStyles(styles)(
    class extends React.Component<ValidatorProps & WithStyles<CssRules>> {
        state: ValidatorState = {
            isPatternValid: true,
            isLengthValid: true
        };

        componentWillReceiveProps(nextProps: ValidatorProps) {
            const { value } = nextProps;

            if (this.props.value !== value) {
                this.setState({
                    isPatternValid: value.match(nameRegEx),
                    isLengthValid: value.length < maxInputLength
                }, () => this.onChange());
            }
        }

        onChange() {
            const { value, onChange, isRequired } = this.props;
            const { isPatternValid, isLengthValid } = this.state;
            const isValid = value && isPatternValid && isLengthValid && (isRequired || (!isRequired && value.length > 0));

            onChange(isValid);
        }

        render() {
            const { classes, isRequired, value } = this.props;
            const { isPatternValid, isLengthValid } = this.state;

            return (
                <span>
            {this.props.render(!(isPatternValid && isLengthValid) && (isRequired || (!isRequired && value.length > 0)))}
                    {!isPatternValid && (isRequired || (!isRequired && value.length > 0)) ?
                        <span className={classes.formInputError}>This field allow only alphanumeric characters, dashes, spaces and underscores.<br/></span> : null}
                    {!isLengthValid ?
                        <span className={classes.formInputError}>This field should have max 60 characters.</span> : null}
          </span>
            );
        }
    }
);
