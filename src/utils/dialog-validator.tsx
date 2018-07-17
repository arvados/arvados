// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core';

type ValidatorProps = {
  value: string,
  onChange: (isValid: boolean | string) => void;
  render: (hasError: boolean) => React.ReactElement<any>;
  isRequired: boolean;
  duplicatedName?: string;
};

interface ValidatorState {
  isLengthValid: boolean;
}

class Validator extends React.Component<ValidatorProps & WithStyles<CssRules>> {
  state: ValidatorState = {
    isLengthValid: true
  };

  componentWillReceiveProps(nextProps: ValidatorProps) {
    const { value } = nextProps;

    if (this.props.value !== value) {
      this.setState({
        isLengthValid: value.length < MAX_INPUT_LENGTH
      }, () => this.onChange());
    }
  }

  onChange() {
    const { value, onChange, isRequired } = this.props;
    const { isLengthValid } = this.state;
    const isValid = value && isLengthValid && (isRequired || (!isRequired && value.length > 0));

    onChange(isValid);
  }

  render() {
    const { classes, isRequired, value, duplicatedName } = this.props;
    const { isLengthValid } = this.state;

    return (
      <span>
        {this.props.render(!isLengthValid && (isRequired || (!isRequired && value.length > 0)))}
        {!isLengthValid ? <span className={classes.formInputError}>This field should have max 255 characters.</span> : null}
        {duplicatedName ? <span className={classes.formInputError}>Project with this name already exists</span> : null}
      </span>
    );
  }
}

const MAX_INPUT_LENGTH = 255;

type CssRules = "formInputError";

const styles: StyleRulesCallback<CssRules> = theme => ({
  formInputError: {
    color: "#ff0000",
    marginLeft: "5px",
    fontSize: "11px",
  }
});

export default withStyles(styles)(Validator);