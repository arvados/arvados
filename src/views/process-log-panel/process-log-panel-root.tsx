// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { Grid } from '@material-ui/core';
import { Process } from '~/store/processes/process';
import { ProcessLogMainCard } from '~/views/process-log-panel/process-log-main-card';
import { ProcessLogFormDataProps, ProcessLogFormActionProps } from '~/views/process-log-panel/process-log-form';
import { DefaultView } from '~/components/default-view/default-view';
import { ProcessIcon } from '~/components/icon/icon';
import { CodeSnippetDataProps } from '~/components/code-snippet/code-snippet';

export type ProcessLogPanelRootDataProps = {
    process?: Process;
} & ProcessLogFormDataProps & CodeSnippetDataProps;

export type ProcessLogPanelRootActionProps = {
    onContextMenu: (event: React.MouseEvent<HTMLElement>) => void;
} & ProcessLogFormActionProps;

export type ProcessLogPanelRootProps = ProcessLogPanelRootDataProps & ProcessLogPanelRootActionProps;

export const ProcessLogPanelRoot = (props: ProcessLogPanelRootProps) =>
    props.process
        ? <Grid container spacing={16}>
            <ProcessLogMainCard 
                process={props.process} 
                {...props} />
        </Grid> 
        : <Grid container
            alignItems='center'
            justify='center'>
            <DefaultView
                icon={ProcessIcon}
                messages={['Process Log not found']} />
        </Grid>;
