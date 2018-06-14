// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import List from "@material-ui/core/List/List";
import ListItem from "@material-ui/core/ListItem/ListItem";
import { StyleRulesCallback, Theme, withStyles, WithStyles } from '@material-ui/core/styles';
import { ReactElement } from "react";
import Collapse from "@material-ui/core/Collapse/Collapse";
import CircularProgress from '@material-ui/core/CircularProgress';
import { inherits } from 'util';

type CssRules = 'list' | 'activeArrow' | 'arrow' | 'arrowRotate' | 'arrowTransition' | 'loader';

const styles: StyleRulesCallback<CssRules> = (theme: Theme) => ({
    list: {
        paddingBottom: '3px',
        paddingTop: '3px',
    },
    activeArrow: {
        color: '#4285F6',
        position: 'absolute',
    },
    arrow: {
        position: 'absolute',
    },
    arrowTransition: { 
        transition: 'all 0.3s ease',
    },
    arrowRotate: {
        transition: 'all 0.3s ease',
        transform: 'rotate(-90deg)',
    },
    loader: {
        position: 'absolute',
        transform: 'translate(0px)',
        top: '3px'  
    }
});

export interface TreeItem<T> {
    data: T;
    id: string;
    open: boolean;
    active: boolean;
    isLoaded: boolean;
    items?: Array<TreeItem<T>>;
}

interface TreeProps<T> {
    items?: Array<TreeItem<T>>;
    render: (item: TreeItem<T>, level?: number) => ReactElement<{}>;
    toggleItem: (id: string) => any;
    level?: number;
}

class Tree<T> extends React.Component<TreeProps<T> & WithStyles<CssRules>, {}> {
    renderArrow (arrowClass: string, open: boolean){
        return <i className={`${arrowClass} ${open ? `fas fa-caret-down ${this.props.classes.arrowTransition}` : `fas fa-caret-down ${this.props.classes.arrowRotate}`}`} />
    }
    render(): ReactElement<any> {
        const level = this.props.level ? this.props.level : 0;
        const {classes, render, toggleItem, items} = this.props;
        const {list, arrow, activeArrow, loader} = classes;
        return <List component="div" className={list}>
            {items && items.map((it: TreeItem<T>, idx: number) =>
             <div key={`item/${level}/${idx}`}>
                <ListItem button onClick={() => toggleItem(it.id)} className={list} style={{paddingLeft: (level + 1) * 20}}>
                    {it.isLoaded ? this.renderArrow(it.active ? activeArrow : arrow, it.open) : <CircularProgress size={10} className={loader}/> }
                    {render(it, level)}
                </ListItem>
                {it.items && it.items.length > 0 &&
                <Collapse in={it.open} timeout="auto" unmountOnExit>
                    <StyledTree
                        items={it.items}
                        render={render}
                        toggleItem={toggleItem}
                        level={level + 1}/>
                </Collapse>}
             </div>)}
        </List>
    }
}

const StyledTree = withStyles(styles)(Tree);
export default StyledTree
