// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { Validator } from 'validators/validators';
import { DialogTextField } from "components/dialog-form/dialog-text-field";
import {
    COLLECTION_NAME_VALIDATION, COLLECTION_NAME_VALIDATION_ALLOW_SLASH,
} from "validators/validators";
import { connect } from "react-redux";
import { RootState } from "store/store";

type DialogCollectionNameFieldProps = {
    defaultValue?: string;
    validators: Validator[];
    submitErr?: string;
    setSubmitErr?: (errMsg: string) => void;
    setValue: (value: string) => void;
}

export const DialogCollectionNameField = connect(
    (state: RootState) => {
        return {
            validators: (state.auth.config.clusterConfig.Collections.ForwardSlashNameSubstitution === "" ?
                COLLECTION_NAME_VALIDATION : COLLECTION_NAME_VALIDATION_ALLOW_SLASH)
        };
    })(({ defaultValue, setValue, validators, submitErr, setSubmitErr }: DialogCollectionNameFieldProps) => {
        return <span data-cy='name-field'>
            <DialogTextField
                label='Collection Name'
                defaultValue={defaultValue || ''}
                setValue={setValue}
                validators={validators}
                submitErr={submitErr}
                setSubmitErr={setSubmitErr}
            />
        </span>
    })
