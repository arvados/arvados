// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { Button, Grid, StyleRulesCallback, WithStyles, Typography, Tooltip } from '@material-ui/core';
import ChevronRightIcon from '@material-ui/icons/ChevronRight';
import { withStyles } from '@material-ui/core';
import { IllegalNamingWarning } from '../warning/warning';
import { IconType, FreezeIcon } from 'components/icon/icon';
import grey from '@material-ui/core/colors/grey';
import { ResourcesState } from 'store/resources/resources';
import classNames from 'classnames';
import { ArvadosTheme } from 'common/custom-theme';
import { extractUuidKind, ResourceKind } from 'models/resource';
import { ClassNameMap } from '@material-ui/core/styles/withStyles';

export interface Breadcrumb {
    label: string;
    icon?: IconType;
    uuid: string;
}

type CssRules = "item" | "defaultItem" | "processItem" | "collectionItem" | "parentItem" | "currentItem" | "label" | "icon" | "frozenIcon";

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    item: {
        borderRadius: '16px',
    },
    defaultItem: {
        backgroundColor: theme.customs.colors.grey300,
        '&:hover': {
            backgroundColor: theme.customs.colors.grey400,
        }
    },
    processItem: {
        backgroundColor: theme.customs.colors.lightGreen300,
        '&:hover': {
            backgroundColor: theme.customs.colors.lightGreen400,
        }
    },
    collectionItem: {
        backgroundColor: theme.customs.colors.cyan100,
        '&:hover': {
            backgroundColor: theme.customs.colors.cyan200,
        }
    },
    parentItem: {
        opacity: 0.75
    },
    currentItem: {
        opacity: 1
    },
    label: {
        textTransform: "none"
    },
    icon: {
        fontSize: 20,
        color: grey["600"],
        marginRight: '10px',
    },
    frozenIcon: {
        fontSize: 20,
        color: grey["600"],
        marginLeft: '10px',
    },
});

export interface BreadcrumbsProps {
    items: Breadcrumb[];
    resources: ResourcesState;
    onClick: (breadcrumb: Breadcrumb) => void;
    onContextMenu: (event: React.MouseEvent<HTMLElement>, breadcrumb: Breadcrumb) => void;
}

const getBreadcrumbClass = (item: Breadcrumb, classes: ClassNameMap<CssRules>): string => {
    switch (extractUuidKind(item.uuid)) {
        case ResourceKind.PROCESS:
            return classes.processItem;
        case ResourceKind.COLLECTION:
            return classes.collectionItem;
        default:
            return classes.defaultItem;
    }
};

export const Breadcrumbs = withStyles(styles)(
    ({ classes, onClick, onContextMenu, items, resources }: BreadcrumbsProps & WithStyles<CssRules>) =>
    <Grid container data-cy='breadcrumbs' alignItems="center" wrap="nowrap">
    {
        items.map((item, index) => {
            const isLastItem = index === items.length - 1;
            const isFirstItem = index === 0;
            const Icon = item.icon || (() => (null));
            return (
                <React.Fragment key={index}>
                    {isFirstItem ? null : <IllegalNamingWarning name={item.label} />}
                    <Tooltip title={item.label}>
                        <Button
                            data-cy={
                                isFirstItem
                                ? 'breadcrumb-first'
                                : isLastItem
                                    ? 'breadcrumb-last'
                                    : false}
                            className={classNames(
                                isLastItem ? classes.currentItem : classes.parentItem,
                                classes.item,
                                getBreadcrumbClass(item, classes)
                            )}
                            onClick={() => onClick(item)}
                            onContextMenu={event => onContextMenu(event, item)}>
                            <Icon className={classes.icon} />
                            <Typography
                                noWrap
                                color="inherit"
                                className={classes.label}>
                                {item.label}
                            </Typography>
                            {
                                (resources[item.uuid] as any)?.frozenByUuid ? <FreezeIcon className={classes.frozenIcon} /> : null
                            }
                        </Button>
                    </Tooltip>
                    {!isLastItem && <ChevronRightIcon color="inherit" className={classes.parentItem} />}
                </React.Fragment>
            );
        })
    }
    </Grid>
);
