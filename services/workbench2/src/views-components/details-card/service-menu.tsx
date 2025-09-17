// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { RootState } from 'store/store';
import { connect } from 'react-redux';
import { Dispatch } from 'redux';
import { Button, Menu, MenuItem, Tooltip } from '@mui/material';
import { ExpandIcon } from 'components/icon/icon';
import { PublishedPort } from 'models/container';
import { showErrorSnackbar } from 'store/snackbar/snackbar-actions';

const mapStateToProps = ({ auth }: RootState) => ({
    userToken: auth.apiToken,
});

const mapDispatchToProps = (dispatch: Dispatch) => ({
    showErrorSnackbar: (message: string) => dispatch<any>(showErrorSnackbar(message)),
});

type ServiceMenuProps = {
    services: PublishedPort[];
    buttonClass?: string;
    userToken: string | undefined;
    showErrorSnackbar: (message: string) => void;
};

export const ServiceMenu = connect(
    mapStateToProps,
    mapDispatchToProps
)(({ services, buttonClass, userToken, showErrorSnackbar }: ServiceMenuProps) => {
    const [anchorEl, setAnchorEl] = React.useState<null | HTMLElement>(null);
    const open = Boolean(anchorEl);
    const handleOpen = (event: React.MouseEvent<HTMLButtonElement>) => {
        setAnchorEl(event.currentTarget);
    };
    const handleClose = () => {
        setAnchorEl(null);
    };
    const handleClick = (service: PublishedPort) => async () => {
        handleClose();

        if (!service.initial_url) {
            showErrorSnackbar("Service URL not set");
            return;
        }

        if (service.access === 'public') {
            // Open public links as-is
            window.open(service.initial_url, "_blank", "noopener");
        } else if (service.access === 'private') {
            // Open private links with user token
            if (userToken) {
                try {
                    const url = await injectTokenParam(service.initial_url, userToken);
                    window.open(url, "_blank", "noopener");
                } catch(e) {
                    showErrorSnackbar("Failed to open service: " + e.message);
                }
            } else {
                showErrorSnackbar("User token not available");
            }
        } else {
            showErrorSnackbar("Published port access value not valid");
        }
    };

    if (services.length) {
        if (services.length === 1) {
            const service = services[0];

            return (
                <Tooltip arrow disableInteractive title={`Connect to ${service.label || "service"}`}>
                    <Button
                        className={buttonClass}
                        variant="contained"
                        size="small"
                        color="primary"
                        id="service-button"
                        onClick={handleClick(service)}
                    >
                        <span>Connect to {service.label || "service"}</span>
                    </Button>
                </Tooltip>
            );
        } else if (services.length > 1) {
            return <>
                <Button
                    className={buttonClass}
                    variant="contained"
                    size="small"
                    color="primary"
                    id="service-button"
                    aria-controls={open ? 'service-menu' : undefined}
                    aria-haspopup="true"
                    aria-expanded={open ? 'true' : undefined}
                    onClick={handleOpen}
                    endIcon={<ExpandIcon />}
                >
                    <span>Connect to service</span>
                </Button>
                <Menu
                    id="service-menu"
                    anchorEl={anchorEl}
                    open={open}
                    onClose={handleClose}
                    MenuListProps={{
                        'aria-labelledby': 'service-button',
                    }}
                >
                    {services.map((service: PublishedPort) => (
                        <MenuItem onClick={handleClick(service)}>
                            <span>{service.label}</span>
                        </MenuItem>
                    ))}
                </Menu>
            </>;
        }
    }

    // Return empty fragment when no services
    return <></>;
});

export const injectTokenParam = (url: string, token: string): Promise<string> => {
    if (url.length) {
        if (token.length) {
            const originalUrl = new URL(url);

            // Remove leading ? for easier manipulation
            const search = originalUrl.search.replace(/^\?/, '');

            // Everything after ?
            const params = `${search}${originalUrl.hash}`;

            // Since search and hash seems to not normalize anything,
            // we should expect href to always end exactly with both.
            // This sanity check should always pass
            if (originalUrl.href.endsWith(params)) {
                // It seems easier to lop off search/params and inject token
                // instead of handling user:pass schemes
                const baseUrl = originalUrl.href
                    // Trim the params from the URL
                    .substring(0, originalUrl.href.length - params.length)
                    // Remove trailing ?
                    .replace(/\?$/, '');

                // Prepend arvados token to search and construct search string
                const searchWithToken = [`arvados_api_token=${token}`, search]
                    // Remove empty elements from array to prevent extra &s with empty search
                    .filter(e => String(e).trim())
                    .join('&');

                return Promise.resolve(`${baseUrl}?${searchWithToken}${originalUrl.hash}`);
            } else {
                // Original url does not end with search+hash, cannot add token
                console.error("Failed to add token to malformed URL: " + url);
                return Promise.reject("Malformed URL");
            }
        } else {
            return Promise.reject("User token required");
        }
    } else {
        return Promise.reject("URL cannot be empty");
    }
};
