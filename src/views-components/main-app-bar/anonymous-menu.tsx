// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { Button } from '@material-ui/core';
import { DispatchProp, connect } from 'react-redux';
import { login } from 'store/auth/auth-action';

export const AnonymousMenu = connect()(
    ({ dispatch }: DispatchProp<any>) =>
        <Button
            color="inherit"
            onClick={() => dispatch(login("", "", "", {}))}>
            Sign in
        </Button>);
