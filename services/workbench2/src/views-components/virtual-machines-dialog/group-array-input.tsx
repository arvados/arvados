// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { StringArrayCommandInputParameter } from 'models/workflow';
import { Field, GenericField } from 'redux-form';
import { GenericInputProps } from 'views/run-process-panel/inputs/generic-input';
import { ChipsInput } from 'components/chips-input/chips-input';
import { identity } from 'lodash';
import { withStyles, WithStyles, FormGroup, Input, InputLabel, FormControl, FormHelperText } from '@material-ui/core';
import classnames from "classnames";
import { ArvadosTheme } from 'common/custom-theme';

export interface GroupArrayDataProps {
  hasPartialGroupInput?: boolean;
  setPartialGroupInput?: (value: boolean) => void;
}

interface GroupArrayFieldProps {
  commandInput: StringArrayCommandInputParameter;
}

const GroupArrayField = Field as new () => GenericField<GroupArrayDataProps & GroupArrayFieldProps>;

export interface GroupArrayInputProps {
  name: string;
  input: StringArrayCommandInputParameter;
  required: boolean;
}

type CssRules = 'chips' | 'partialInputHelper' | 'partialInputHelperVisible';

const styles = (theme: ArvadosTheme) => ({
    chips: {
        marginTop: "16px",
    },
    partialInputHelper: {
        textAlign: 'right' as 'right',
        visibility: 'hidden' as 'hidden',
        color: theme.palette.error.dark,
    },
    partialInputHelperVisible: {
        visibility: 'visible' as 'visible',
    }
});

export const GroupArrayInput = ({name, input, setPartialGroupInput, hasPartialGroupInput}: GroupArrayInputProps & GroupArrayDataProps) => {
  return <GroupArrayField
      name={name}
      commandInput={input}
      component={GroupArrayInputComponent as any}
      setPartialGroupInput={setPartialGroupInput}
      hasPartialGroupInput={hasPartialGroupInput}
      />;
}

const GroupArrayInputComponent = (props: GenericInputProps & GroupArrayDataProps) => {
  return <FormGroup>
        <FormControl fullWidth error={props.meta.error}>
          <InputLabel shrink={props.meta.active || props.input.value.length > 0}>{props.commandInput.id}</InputLabel>
          <StyledInputComponent {...props} />
        </FormControl>
    </FormGroup>;
    };

const StyledInputComponent = withStyles(styles)(
  class InputComponent extends React.PureComponent<GenericInputProps & WithStyles<CssRules> & GroupArrayDataProps>{
      render() {
          const { classes } = this.props;
          const { commandInput, input, meta, hasPartialGroupInput } = this.props;
          return <>
            <ChipsInput
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
                onPartialInput={this.props.setPartialGroupInput}
                inputProps={{
                    error: meta.error || hasPartialGroupInput,
                }} />
                <FormHelperText className={classnames([classes.partialInputHelper, ...(hasPartialGroupInput ? [classes.partialInputHelperVisible] : [])])}>
                  Press enter to complete group name
                </FormHelperText>
          </>;
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
