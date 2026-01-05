// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { TextField } from "@mui/material";
import { getFieldErrors, Validator } from "validators/validators";
import { usePrevious } from "common/usePrevious";

interface DialogTextFieldProps {
    label: string;
    defaultValue: string;
    validators?: Validator[];
    setValue: React.Dispatch<React.SetStateAction<string>>;
}

export const DialogTextField = React.memo(({ label, defaultValue, validators, setValue }: DialogTextFieldProps) => {
    const [thisValue, setThisValue] = React.useState(defaultValue);
    const errs = validators ? getFieldErrors(thisValue, validators) : []
    const prevErr = usePrevious(errs)

    // set parent value when error state changes
    // necessary to set for submit button disable/enable
    React.useEffect(() => {
        if (prevErr && Boolean(prevErr.length) !== Boolean(errs.length)) {
            setValue(thisValue.trim())
        }
    }, [thisValue])

    return (
        <TextField
            value={thisValue}
            onChange={(e) => setThisValue(e.target.value)}
            autoFocus
            required
            error={errs.length > 0}
            helperText={errs.join(', ') || ''}
            margin="dense"
            id="name"
            name="name"
            type="text"
            fullWidth
            variant="standard"
            label={label}
            onBlur={() => setValue(thisValue)}
        />
    )
})