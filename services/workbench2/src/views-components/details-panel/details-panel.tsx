// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { IconButton, Tabs, Tab, Typography, Grid, Tooltip } from '@mui/material';
import { CustomStyleRulesCallback } from 'common/custom-theme';
import { WithStyles } from '@mui/styles';
import withStyles from '@mui/styles/withStyles';
import { Transition } from 'react-transition-group';
import { ArvadosTheme } from 'common/custom-theme';
import { connect } from 'react-redux';
import { RootState } from 'store/store';
import { CloseIcon } from 'components/icon/icon';
import { EmptyResource } from 'models/empty';
import { Dispatch } from "redux";
import { ResourceKind, isResourceResource } from "models/resource";
import { ProjectDetails } from "./project-details";
import { RootProjectDetails } from './root-project-details';
import { CollectionDetails } from "./collection-details";
import { ProcessDetails } from "./process-details";
import { EmptyDetails } from "./empty-details";
import { WorkflowDetails } from "./workflow-details";
import { DetailsData } from "./details-data";
import { DetailsResource } from "models/details";
import { Config } from 'common/config';
import { isInlineFileUrlSafe } from "../context-menu/actions/helpers";
import { getResource } from 'store/resources/resources';
import { toggleDetailsPanel, SLIDE_TIMEOUT, openDetailsPanel } from 'store/details-panel/details-panel-action';
import { FileDetails } from 'views-components/details-panel/file-details';
import { getNode } from 'models/tree';
import { resourceIsFrozen } from 'common/frozen-resources';
import { CLOSE_DRAWER } from 'store/details-panel/details-panel-action';

type CssRules =
    | "root"
    | "container"
    | "headerContainer"
    | "headerTitleWrapper"
    | "headerIconWrapper"
    | "headerIcon"
    | "tabContainerWrapper"
    | "tabContainer"
    | "tab";

const styles: CustomStyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        background: theme.palette.background.paper,
        height: '100%',
        overflow: 'hidden',
    },
    container: {
        maxWidth: 'none',
    },
    headerContainer: {
        color: theme.palette.grey["600"],
        margin: `${theme.spacing(1)} 0`,
        textAlign: 'center',
    },
    headerIconWrapper: {
        margin: '0 10px',
    },
    headerTitleWrapper: {
        overflow: 'hidden',
    },
    headerIcon: {
        fontSize: '2.125rem',
    },
    tabContainerWrapper: {
        // Grid container wrapper prevents horizontal overflow
        // Flex styles prevent unscrollable vertical overflow
        overflowY: 'scroll',
        flexGrow: 1,
        flexShrink: 1,
        flexBasis: 'inherit',
    },
    tabContainer: {
        overflow: 'auto',
        padding: theme.spacing(1),
    },
    tab: {
        borderBottom: `1px solid ${theme.palette.grey[300]}`,
    },
});

const EMPTY_RESOURCE: EmptyResource = { kind: undefined, name: 'Projects' };

const getItem = (res: DetailsResource, pathName: string): DetailsData => {
    if ('kind' in res) {
        switch (res.kind) {
            case ResourceKind.PROJECT:
                return new ProjectDetails(res);
            case ResourceKind.COLLECTION:
                return new CollectionDetails(res);
            case ResourceKind.PROCESS:
                return new ProcessDetails(res);
            case ResourceKind.WORKFLOW:
                return new WorkflowDetails(res);
            case ResourceKind.USER:
                if(pathName.includes('projects')) {
                    return new RootProjectDetails(res);
                }
                return new EmptyDetails(EMPTY_RESOURCE);
            default:
                return new EmptyDetails(res as EmptyResource);
        }
    } else {
        return new FileDetails(res);
    }
};

const mapStateToProps = ({ auth, detailsPanel, resources, collectionPanelFiles, selectedResource, properties, router }: RootState) => {
    const resource = getResource(selectedResource.selectedResourceUuid ?? properties.currentRouteUuid)(resources) as DetailsResource | undefined;
    const file = resource
        ? undefined
        : getNode(detailsPanel.resourceUuid)(collectionPanelFiles);

    let isFrozen = false;
    if (isResourceResource(resource)) {
        isFrozen = resourceIsFrozen(resource, resources);
    }

    return {
        isFrozen,
        authConfig: auth.config,
        isOpened: detailsPanel.isOpened,
        tabNr: detailsPanel.tabNr,
        res: resource || (file && file.value) || EMPTY_RESOURCE,
        pathname: router.location ? router.location.pathname : "",
    };
};

const mapDispatchToProps = (dispatch: Dispatch) => ({
    onCloseDrawer: (currentItemId) => {
        dispatch<any>(toggleDetailsPanel(currentItemId));
    },
    setActiveTab: (tabNr: number) => {
        dispatch<any>(openDetailsPanel(undefined, tabNr));
    },
});

export interface DetailsPanelDataProps {
    onCloseDrawer: (currentItemId) => void;
    setActiveTab: (tabNr: number) => void;
    authConfig: Config;
    isOpened: boolean;
    tabNr: number;
    res: DetailsResource;
    isFrozen: boolean;
    pathname: string;
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
                        className={classes.root}>
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
                const { classes, onCloseDrawer, res, tabNr, authConfig, pathname } = this.props;
                let shouldShowInlinePreview = false;
                if (!('kind' in res)) {
                    shouldShowInlinePreview = isInlineFileUrlSafe(
                        res ? res.url : "",
                        authConfig.keepWebServiceUrl,
                        authConfig.keepWebInlineServiceUrl
                    ) || authConfig.clusterConfig.Collections.TrustAllContent;
                }

                const item = getItem(res, pathname);
                return (
                    <Grid
                        data-cy='details-panel'
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
                            justifyContent='space-between'
                            wrap="nowrap">
                            <Grid item className={classes.headerIconWrapper}>
                                {item.getIcon(classes.headerIcon)}
                            </Grid>
                            <Grid item className={classes.headerTitleWrapper}>
                                <Tooltip title={item.getTitle()}>
                                    <Typography variant='h6' noWrap>
                                        {item.getTitle()}
                                    </Typography>
                                </Tooltip>
                            </Grid>
                            <Grid item>
                                <IconButton data-cy="close-details-btn" color="inherit" onClick={()=>onCloseDrawer(CLOSE_DRAWER)} size="large">
                                    <CloseIcon />
                                </IconButton>
                            </Grid>
                        </Grid>
                        <Grid item>
                            <Tabs onChange={this.handleChange}
                                variant='fullWidth'
                                value={(item.getTabLabels().length >= tabNr + 1) ? tabNr : 0}>
                                {item.getTabLabels().map((tabLabel, idx) =>
                                    <Tab className={classes.tab} key={`tab-label-${idx}`} data-cy={`details-panel-tab-${tabLabel}`} disableRipple label={tabLabel} />)
                                }
                            </Tabs>
                        </Grid>
                        <Grid item container className={this.props.classes.tabContainerWrapper}>
                            <Grid item xs className={this.props.classes.tabContainer}>
                                {item.getDetails({ tabNr, showPreview: shouldShowInlinePreview })}
                            </Grid>
                        </Grid>
                    </Grid >
                );
            }
        }
    )
);
