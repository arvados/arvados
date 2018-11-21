// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from "react";
import { Toolbar, IconButton, Tooltip, Grid } from "@material-ui/core";
import { DetailsIcon } from "~/components/icon/icon";
import { Breadcrumbs } from "~/views-components/breadcrumbs/breadcrumbs";
import { connect } from 'react-redux';
import { RootState } from '~/store/store';
import { matchWorkflowRoute, matchSshKeysRoute, matchRepositoriesRoute } from '~/routes/routes';
import { toggleDetailsPanel } from '~/store/details-panel/details-panel-action';

interface MainContentBarProps {
    onDetailsPanelToggle: () => void;
    buttonVisible: boolean;
}

const isWorkflowPath = ({ router }: RootState) => {
    const pathname = router.location ? router.location.pathname : '';
    const match = matchWorkflowRoute(pathname);
    return !!match;
};

const isRepositoriesPath = ({ router }: RootState) => {
    const pathname = router.location ? router.location.pathname : '';
    const match = matchRepositoriesRoute(pathname);
    return !!match;
};

const isSshKeysPath = ({ router }: RootState) => {
    const pathname = router.location ? router.location.pathname : '';
    const match = matchSshKeysRoute(pathname);
    return !!match;
};

export const MainContentBar = connect((state: RootState) => ({
    buttonVisible: !isWorkflowPath(state) && !isSshKeysPath(state) && !isRepositoriesPath(state)
}), {
        onDetailsPanelToggle: toggleDetailsPanel
    })((props: MainContentBarProps) =>
        <Toolbar>
            <Grid container>
                <Grid container item xs alignItems="center">
                    <Breadcrumbs />
                </Grid>
                <Grid item>
                    {props.buttonVisible && <Tooltip title="Additional Info">
                        <IconButton color="inherit" onClick={props.onDetailsPanelToggle}>
                            <DetailsIcon />
                        </IconButton>
                    </Tooltip>}
                </Grid>
            </Grid>
        </Toolbar>);
