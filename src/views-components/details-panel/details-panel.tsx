// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { IconButton, Tabs, Tab, Typography, Grid, Tooltip } from '@material-ui/core';
import { StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core/styles';
import { Transition } from 'react-transition-group';
import { ArvadosTheme } from 'common/custom-theme';
import classnames from "classnames";
import { connect } from 'react-redux';
import { RootState } from 'store/store';
import { CloseIcon } from 'components/icon/icon';
import { EmptyResource } from 'models/empty';
import { Dispatch } from "redux";
import { ResourceKind } from "models/resource";
import { ProjectDetails } from "./project-details";
import { CollectionDetails } from "./collection-details";
import { ProcessDetails } from "./process-details";
import { EmptyDetails } from "./empty-details";
import { DetailsData } from "./details-data";
import { DetailsResource } from "models/details";
import { getResource } from 'store/resources/resources';
import { toggleDetailsPanel, SLIDE_TIMEOUT, openDetailsPanel } from 'store/details-panel/details-panel-action';
import { FileDetails } from 'views-components/details-panel/file-details';
import { getNode } from 'models/tree';

type CssRules = 'root' | 'container' | 'opened' | 'headerContainer' | 'headerIcon' | 'tabContainer';

const DRAWER_WIDTH = 320;
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
        padding: theme.spacing.unit * 1,
    },
});

const EMPTY_RESOURCE: EmptyResource = { kind: undefined, name: 'Projects' };

const getItem = (res: DetailsResource): DetailsData => {
    if ('kind' in res) {
        switch (res.kind) {
            case ResourceKind.PROJECT:
                return new ProjectDetails(res);
            case ResourceKind.COLLECTION:
                return new CollectionDetails(res);
            case ResourceKind.PROCESS:
                return new ProcessDetails(res);
            default:
                return new EmptyDetails(res);
        }
    } else {
        return new FileDetails(res);
    }
};

const mapStateToProps = ({ detailsPanel, resources, collectionPanelFiles }: RootState) => {
    const resource = getResource(detailsPanel.resourceUuid)(resources) as DetailsResource | undefined;
    const file = resource
        ? undefined
        : getNode(detailsPanel.resourceUuid)(collectionPanelFiles);
    return {
        isOpened: detailsPanel.isOpened,
        tabNr: detailsPanel.tabNr,
        res: resource || (file && file.value) || EMPTY_RESOURCE,
    };
};

const mapDispatchToProps = (dispatch: Dispatch) => ({
    onCloseDrawer: () => {
        dispatch<any>(toggleDetailsPanel());
    },
    setActiveTab: (tabNr: number) => {
        dispatch<any>(openDetailsPanel(undefined, tabNr));
    },
});

export interface DetailsPanelDataProps {
    onCloseDrawer: () => void;
    setActiveTab: (tabNr: number) => void;
    isOpened: boolean;
    tabNr: number;
    res: DetailsResource;
}

type DetailsPanelProps = DetailsPanelDataProps & WithStyles<CssRules>;

export const DetailsPanel = withStyles(styles)(
    connect(mapStateToProps, mapDispatchToProps)(
        class extends React.Component<DetailsPanelProps> {
            shouldComponentUpdate(nextProps: DetailsPanelProps) {
                if ('etag' in nextProps.res && 'etag' in this.props.res &&
                    nextProps.res.etag === this.props.res.etag &&
                    nextProps.isOpened === this.props.isOpened &&
                    nextProps.tabNr === this.props.tabNr) {
                    return false;
                }
                return true;
            }

            handleChange = (event: any, value: number) => {
                this.props.setActiveTab(value);
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
                            {isOpened ? this.renderContent() : <div />}
                        </Transition>
                    </Grid>
                );
            }

            renderContent() {
                const { classes, onCloseDrawer, res, tabNr } = this.props;
                const item = getItem(res);
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
                                <Typography variant='h6' noWrap>
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
                        <Tabs onChange={this.handleChange}
                            value={(item.getTabLabels().length >= tabNr+1) ? tabNr : 0}>
                            { item.getTabLabels().map((tabLabel, idx) =>
                                <Tab key={`tab-label-${idx}`} disableRipple label={tabLabel} />)
                            }
                        </Tabs>
                    </Grid>
                    <Grid item xs className={this.props.classes.tabContainer} >
                        {item.getDetails(tabNr)}
                    </Grid>
                </Grid >;
            }
        }
    )
);
