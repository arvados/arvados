// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { RootState } from 'store/store';
import React from 'react';
import { connect } from 'react-redux';
import { NotFoundPanelRoot, NotFoundPanelRootDataProps } from 'views/not-found-panel/not-found-panel-root';
import { Grid } from '@mui/material';
import { DefaultView } from "components/default-view/default-view";
import { IconType } from 'components/icon/icon';

const mapStateToProps = (state: RootState): NotFoundPanelRootDataProps => {
    return {
        location: state.router.location,
        clusterConfig: state.auth.config.clusterConfig,
    };
};

const mapDispatchToProps = null;

export const NotFoundPanel = connect(mapStateToProps, mapDispatchToProps)
    (NotFoundPanelRoot) as any;

export interface NotFoundViewDataProps {
    messages: string[];
    icon?: IconType;
}

// TODO: optionally pass in the UUID and check if the
// reason the item is not found is because
// it or a parent project is actually in the trash.
// If so, offer to untrash the item or the parent project.
export const NotFoundView =
    ({ messages, icon: Icon }: NotFoundViewDataProps) =>
        <Grid
            container
            alignItems="center"
            justifyContent="center"
            style={{ minHeight: "100%" }}
            data-cy="not-found-view">
            <DefaultView
                icon={Icon}
                messages={messages}
            />
        </Grid>;
