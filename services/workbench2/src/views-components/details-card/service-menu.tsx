// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Button, Menu, MenuItem, Tooltip } from '@mui/material';
import { ExpandIcon } from 'components/icon/icon';
import { PublishedPort } from 'models/container';
import React from 'react';

type ServiceMenuProps = {
    services: PublishedPort[];
    buttonClass?: string;
};

export const ServiceMenu = ({ services, buttonClass }: ServiceMenuProps) => {
    const [anchorEl, setAnchorEl] = React.useState<null | HTMLElement>(null);
    const open = Boolean(anchorEl);
    const handleOpen = (event: React.MouseEvent<HTMLButtonElement>) => {
      setAnchorEl(event.currentTarget);
    };
    const handleClose = () => {
      setAnchorEl(null);
    };
    const handleClick = (service: PublishedPort) => () => {
        handleClose();
        window.open(service.initial_url, "_blank", "noopener");
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
                    aria-controls={open ? 'basic-menu' : undefined}
                    aria-haspopup="true"
                    aria-expanded={open ? 'true' : undefined}
                    onClick={handleOpen}
                    endIcon={<ExpandIcon />}
                >
                    <span>Connect to service</span>
                </Button>
                <Menu
                    id="basic-menu"
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
  }
