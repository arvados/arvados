// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { StyleRulesCallback, WithStyles, withStyles } from '@material-ui/core/styles';
import { ArvadosTheme } from '~/common/custom-theme';
import { List, ListItem, ListItemIcon, Collapse } from "@material-ui/core";
import { SidePanelRightArrowIcon, IconType } from '../icon/icon';
import * as classnames from "classnames";
import { ListItemTextIcon } from '../list-item-text-icon/list-item-text-icon';
import { Dispatch } from "redux";
import { RouteComponentProps, withRouter } from "react-router";

type CssRules = 'active' | 'row' | 'root' | 'list' | 'iconClose' | 'iconOpen' | 'toggableIconContainer' | 'toggableIcon';

const styles: StyleRulesCallback<CssRules> = (theme: ArvadosTheme) => ({
    root: {
        overflowY: 'auto',
        minWidth: '240px',
        whiteSpace: 'nowrap',
        marginTop: '52px',
        display: 'flex',
        flexGrow: 1,
    },
    list: {
        padding: '5px 0px 5px 14px',
        minWidth: '240px',
    },
    row: {
        display: 'flex',
        alignItems: 'center',
    },
    toggableIconContainer: {
        color: theme.palette.grey["700"],
        height: '14px',
        width: '14px'
    },
    toggableIcon: {
        fontSize: '14px'
    },
    active: {
        color: theme.palette.primary.main,
    },
    iconClose: {
        transition: 'all 0.1s ease',
    },
    iconOpen: {
        transition: 'all 0.1s ease',
        transform: 'rotate(90deg)',
    }
});

export interface SidePanelItem {
    id: string;
    name: string;
    url: string;
    icon: IconType;
    open?: boolean;
    margin?: boolean;
    openAble?: boolean;
    activeAction?: (dispatch: Dispatch, uuid?: string) => void;
}

interface SidePanelDataProps {
    toggleOpen: (id: string) => void;
    toggleActive: (id: string) => void;
    sidePanelItems: SidePanelItem[];
    onContextMenu: (event: React.MouseEvent<HTMLElement>, item: SidePanelItem) => void;
}

type SidePanelProps = RouteComponentProps<{}> & SidePanelDataProps & WithStyles<CssRules>;

export const SidePanel = withStyles(styles)(withRouter(
    class extends React.Component<SidePanelProps> {
        render() {
            const { classes, toggleOpen, toggleActive, sidePanelItems, children } = this.props;
            const { root, row, list, toggableIconContainer } = classes;

            const path = this.props.location.pathname.split('/');
            const activeUrl = path.length > 1 ? "/" + path[1] : "/";
            return (
                <div className={root}>
                    <List>
                        {sidePanelItems.map(it => {
                            const active = it.url === activeUrl;
                            return <span key={it.name}>
                                <ListItem button className={list} onClick={() => toggleActive(it.id)}
                                          onContextMenu={this.handleRowContextMenu(it)}>
                                    <span className={row}>
                                        {it.openAble ? (
                                            <i onClick={() => toggleOpen(it.id)} className={toggableIconContainer}>
                                                <ListItemIcon
                                                    className={this.getToggableIconClassNames(it.open, active)}>
                                                    < SidePanelRightArrowIcon/>
                                                </ListItemIcon>
                                            </i>
                                        ) : null}
                                        <ListItemTextIcon icon={it.icon} name={it.name} isActive={active}
                                                          hasMargin={it.margin}/>
                                    </span>
                                </ListItem>
                                {it.openAble ? (
                                    <Collapse in={it.open} timeout="auto" unmountOnExit>
                                        {children}
                                    </Collapse>
                                ) : null}
                            </span>;
                        })}
                    </List>
                </div>
            );
        }

        getToggableIconClassNames = (isOpen?: boolean, isActive ?: boolean) => {
            const { iconOpen, iconClose, active, toggableIcon } = this.props.classes;
            return classnames(toggableIcon, {
                [iconOpen]: isOpen,
                [iconClose]: !isOpen,
                [active]: isActive
            });
        }

        handleRowContextMenu = (item: SidePanelItem) =>
            (event: React.MouseEvent<HTMLElement>) =>
                item.openAble ? this.props.onContextMenu(event, item) : null
    }
));
