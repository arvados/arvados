// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { IconButton, Tabs, Tab, Typography, Grid, Tooltip } from '@material-ui/core';
import { StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core/styles';
import { Transition } from 'react-transition-group';
import { ArvadosTheme } from '~/common/custom-theme';
import * as classnames from "classnames";
import { connect } from 'react-redux';
import { RootState } from '~/store/store';
import { detailsPanelActions } from "~/store/details-panel/details-panel-action";
import { CloseIcon } from '~/components/icon/icon';
import { EmptyResource } from '~/models/empty';
import { Dispatch } from "redux";
import { ResourceKind } from "~/models/resource";
import { ProjectDetails } from "./project-details";
import { CollectionDetails } from "./collection-details";
import { ProcessDetails } from "./process-details";
import { EmptyDetails } from "./empty-details";
import { DetailsData } from "./details-data";
import { DetailsResource } from "~/models/details";
import { getResource } from '~/store/resources/resources';
import { ResourceData } from "~/store/resources-data/resources-data-reducer";
import { getResourceData } from "~/store/resources-data/resources-data";

type CssRules = 'root' | 'container' | 'opened' | 'headerContainer' | 'headerIcon' | 'tabContainer';

const DRAWER_WIDTH = 320;
const SLIDE_TIMEOUT = 500;
const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        background: theme.palette.background.paper,
        borderLeft: `1px solid ${theme.palette.divider}`,
        height: '100%',
        overflow: 'hidden',
        transition: `width ${SLIDE_TIMEOUT}ms ease`,
        width: 0,
    },
    opened: {
        width: DRAWER_WIDTH,
    },
    container: {
        maxWidth: 'none',
        width: DRAWER_WIDTH,
    },
    headerContainer: {
        color: theme.palette.grey["600"],
        margin: `${theme.spacing.unit}px 0`,
        textAlign: 'center',
    },
    headerIcon: {
        fontSize: '2.125rem',
    },
    tabContainer: {
        overflow: 'auto',
        padding: theme.spacing.unit * 3,
    },
});

const getItem = (resource: DetailsResource, resourceData?: ResourceData): DetailsData => {
    const res = resource || { kind: undefined, name: 'Projects' };
    switch (res.kind) {
        case ResourceKind.PROJECT:
            return new ProjectDetails(res);
        case ResourceKind.COLLECTION:
            return new CollectionDetails(res, resourceData);
        case ResourceKind.PROCESS:
            return new ProcessDetails(res);
        default:
            return new EmptyDetails(res as EmptyResource);
    }
};

const mapStateToProps = ({ detailsPanel, resources, resourcesData }: RootState) => {
    const resource = getResource(detailsPanel.resourceUuid)(resources) as DetailsResource;
    const resourceData = getResourceData(detailsPanel.resourceUuid)(resourcesData);
    return {
        isOpened: detailsPanel.isOpened,
        item: getItem(resource, resourceData)
    };
};

const mapDispatchToProps = (dispatch: Dispatch) => ({
    onCloseDrawer: () => {
        dispatch(detailsPanelActions.TOGGLE_DETAILS_PANEL());
    }
});

export interface DetailsPanelDataProps {
    onCloseDrawer: () => void;
    isOpened: boolean;
    item: DetailsData;
}

type DetailsPanelProps = DetailsPanelDataProps & WithStyles<CssRules>;

export const DetailsPanel = withStyles(styles)(
    connect(mapStateToProps, mapDispatchToProps)(
        class extends React.Component<DetailsPanelProps> {
            state = {
                tabsValue: 0
            };

            handleChange = (event: any, value: boolean) => {
                this.setState({ tabsValue: value });
            }

            render() {
                const { classes, isOpened } = this.props;
                return (
                    <Grid
                        container
                        direction="column"
                        className={classnames([classes.root, { [classes.opened]: isOpened }])}>
                        <Transition
                            in={isOpened}
                            timeout={SLIDE_TIMEOUT}
                            unmountOnExit>
                            {this.renderContent()}
                        </Transition>
                    </Grid>
                );
            }

            renderContent() {
                const { classes, onCloseDrawer, item } = this.props;
                const { tabsValue } = this.state;
                return <Grid
                    container
                    direction="column"
                    item
                    xs
                    className={classes.container} >
                    <Grid
                        item
                        className={classes.headerContainer}
                        container
                        alignItems='center'
                        justify='space-around'
                        wrap="nowrap">
                        <Grid item xs={2}>
                            {item.getIcon(classes.headerIcon)}
                        </Grid>
                        <Grid item xs={8}>
                            <Tooltip title={item.getTitle()}>
                                <Typography variant="title" noWrap>
                                    {item.getTitle()}
                                </Typography>
                            </Tooltip>
                        </Grid>
                        <Grid item>
                            <IconButton color="inherit" onClick={onCloseDrawer}>
                                <CloseIcon />
                            </IconButton>
                        </Grid>
                    </Grid>
                    <Grid item>
                        <Tabs value={tabsValue} onChange={this.handleChange}>
                            <Tab disableRipple label="Details" />
                            <Tab disableRipple label="Activity" disabled />
                        </Tabs>
                    </Grid>
                    <Grid item xs className={this.props.classes.tabContainer} >
                        {tabsValue === 0
                            ? item.getDetails()
                            : null}
                    </Grid>
                </Grid >;
            }
        }
    )
);
