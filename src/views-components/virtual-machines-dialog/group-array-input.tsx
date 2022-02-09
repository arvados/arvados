// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { StringArrayCommandInputParameter } from 'models/workflow';
import { Field } from 'redux-form';
import { GenericInputProps } from 'views/run-process-panel/inputs/generic-input';
import { ChipsInput } from 'components/chips-input/chips-input';
import { identity } from 'lodash';
import { withStyles, WithStyles, FormGroup, Input, InputLabel, FormControl } from '@material-ui/core';

export interface StringArrayInputProps {
  name: string;
  input: StringArrayCommandInputParameter;
  required: boolean;
}

type CssRules = 'chips';

const styles = {
    chips: {
        marginTop: "16px",
    },
};

export const GroupArrayInput = ({name, input}: StringArrayInputProps) =>
    <Field
        name={name}
        commandInput={input}
        component={StringArrayInputComponent as any}
        />;

const StringArrayInputComponent = (props: GenericInputProps) => {
  return <FormGroup>
        <FormControl fullWidth error={props.meta.error}>
          <InputLabel shrink={props.meta.active || props.input.value.length > 0}>{props.commandInput.id}</InputLabel>
          <StyledInputComponent {...props} />
        </FormControl>
    </FormGroup>;
    };

const StyledInputComponent = withStyles(styles)(
  class InputComponent extends React.PureComponent<GenericInputProps & WithStyles<CssRules>>{
      render() {
          const { classes } = this.props;
          const { commandInput, input, meta } = this.props;
          return <ChipsInput
              deletable={!commandInput.disabled}
              orderable={!commandInput.disabled}
              disabled={commandInput.disabled}
              values={input.value}
              onChange={this.handleChange}
              handleFocus={input.onFocus}
              createNewValue={identity}
              inputComponent={Input}
              chipsClassName={classes.chips}
              pattern={/[_a-z][-0-9_a-z]*/ig}
              inputProps={{
                  error: meta.error,
              }} />;
      }

      handleChange = (values: {}[]) => {
        const { input, meta } = this.props;
          if (!meta.touched) {
              input.onBlur(values);
          }
          input.onChange(values);
      }

  }
);
