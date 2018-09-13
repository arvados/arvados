// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { CircularProgress } from '@material-ui/core';
import { withProgress } from '~/store/progress-indicator/with-progress';
import { WithProgressStateProps } from '~/store/progress-indicator/with-progress';
import { ProgressIndicatorData } from '~/store/progress-indicator/progress-indicator-reducer';

export const ContentProgress = withProgress(ProgressIndicatorData.CONTENT_PROGRESS)((props: WithProgressStateProps) => 
    props.started ? <CircularProgress /> : null
);
