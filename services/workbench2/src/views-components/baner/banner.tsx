// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React, { useState, useCallback, useEffect } from 'react';
import { Dialog, DialogContent, DialogActions, Button, StyleRulesCallback, withStyles, WithStyles } from "@material-ui/core";
import { connect } from "react-redux";
import { RootState } from "store/store";
import bannerActions from "store/banner/banner-action";
import { ArvadosTheme } from 'common/custom-theme';
import servicesProvider from 'common/service-provider';
import { Dispatch } from 'redux';

type CssRules = 'dialogContent' | 'dialogContentIframe';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    dialogContent: {
        minWidth: '550px',
        minHeight: '500px',
        display: 'block'
    },
    dialogContentIframe: {
        minWidth: '550px',
        minHeight: '500px'
    }
});

interface BannerProps {
    isOpen: boolean;
    bannerUUID?: string;
    keepWebInlineServiceUrl: string;
};

type BannerComponentProps = BannerProps & WithStyles<CssRules> & {
    openBanner: Function,
    closeBanner: Function,
};

const mapStateToProps = (state: RootState): BannerProps => ({
    isOpen: state.banner.isOpen,
    bannerUUID: state.auth.config.clusterConfig.Workbench.BannerUUID,
    keepWebInlineServiceUrl: state.auth.config.keepWebInlineServiceUrl,
});

const mapDispatchToProps = (dispatch: Dispatch) => ({
    openBanner: () => dispatch<any>(bannerActions.openBanner()),
    closeBanner: () => dispatch<any>(bannerActions.closeBanner()),
});

export const BANNER_LOCAL_STORAGE_KEY = 'bannerFileData';

export const BannerComponent = (props: BannerComponentProps) => {
    const { 
        isOpen,
        openBanner,
        closeBanner,
        bannerUUID,
        keepWebInlineServiceUrl
    } = props;
    const [bannerContents, setBannerContents] = useState(`<h1>Loading ...</h1>`)

    const onConfirm = useCallback(() => {
        closeBanner();
    }, [closeBanner])

    useEffect(() => {
        if (!!bannerUUID && bannerUUID !== "") {
            servicesProvider.getServices().collectionService.files(bannerUUID)
                .then(results => {
                    const bannerFileData = results.find(({name}) => name === 'banner.html');
                    const result = localStorage.getItem(BANNER_LOCAL_STORAGE_KEY);

                    if (result && result === JSON.stringify(bannerFileData) && !isOpen) {
                        return;
                    }

                    if (bannerFileData) {
                        servicesProvider.getServices()
                            .collectionService.getFileContents(bannerFileData)
                            .then(data => {
                                setBannerContents(data);
                                openBanner();
                                localStorage.setItem(BANNER_LOCAL_STORAGE_KEY, JSON.stringify(bannerFileData));
                            });
                    }
                });
        }
    }, [bannerUUID, keepWebInlineServiceUrl, openBanner, isOpen]);

    return (
        <Dialog open={isOpen}>
            <div data-cy='confirmation-dialog'>
                <DialogContent className={props.classes.dialogContent}>
                    <div dangerouslySetInnerHTML={{ __html: bannerContents }}></div>
                </DialogContent>
                <DialogActions style={{ margin: '0px 24px 24px' }}>
                    <Button
                        data-cy='confirmation-dialog-ok-btn'
                        variant='contained'
                        color='primary'
                        type='submit'
                        onClick={onConfirm}>
                        Close
                    </Button>
                </DialogActions>
            </div>
        </Dialog>
    );
}

export const Banner = withStyles(styles)(connect(mapStateToProps, mapDispatchToProps)(BannerComponent));
