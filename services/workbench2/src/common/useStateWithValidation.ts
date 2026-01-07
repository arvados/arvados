// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { useEffect, useState } from "react";
import { getFieldErrors, Validator } from "validators/validators";

/**
 * A custom hook that manages state for a string value with automatic field validation.
 *
 * @param value - The initial value for the state
 * @param validators - Array of validator functions to apply to the value
 * @param fieldName - Optional field name to prepend to error messages for better debugging
 * @returns A tuple containing:
 *   - The current value
 *   - A setter function to update the value
 *   - An array of validation error messages
 *
 * @example
 * ```tsx
 * const [name, setName, nameErrors] = useStateWithValidation('', REQUIRED_LENGTH255_VALIDATION);
 * ```
 */
export const useStateWithValidation = (value: string, validators: Validator[], fieldName?: string) => {
    const [thisValue, setThisValue] = useState(value);
    const [errors, setErrors] = useState<string[]>(() => getFieldErrors(value, validators, fieldName));

    useEffect(() => {
        const errs = getFieldErrors(thisValue, validators, fieldName);
        setErrors(errs);
    }, [thisValue]);

    return [thisValue, setThisValue, errors] as const;
}