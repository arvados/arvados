// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import List from "@material-ui/core/List/List";
import ListItem from "@material-ui/core/ListItem/ListItem";
import { StyleRulesCallback, Theme, withStyles, WithStyles } from '@material-ui/core/styles';
import { ReactElement } from "react";
import Collapse from "@material-ui/core/Collapse/Collapse";

type CssRules = 'list';

const styles: StyleRulesCallback<CssRules> = (theme: Theme) => ({
    list: {
        paddingBottom: '3px', 
        paddingTop: '3px',
    }
});

export interface TreeItem<T> {
    data: T;
    id: string;
    open: boolean;
    active: boolean;
    items?: Array<TreeItem<T>>;
}

interface TreeProps<T> {
    items?: Array<TreeItem<T>>;
    render: (item: TreeItem<T>) => ReactElement<{}>;
    toggleItem: (id: string) => any;
    level?: number;
}

class Tree<T> extends React.Component<TreeProps<T> & WithStyles<CssRules>, {}> {
    render(): ReactElement<any> {
        const level = this.props.level ? this.props.level : 0;
        const {classes, render, toggleItem, items} = this.props;
        return <List component="div" className={classes.list}>
            {items && items.map((it: TreeItem<T>, idx: number) =>
             <div key={`item/${level}/${idx}`}>      
                <ListItem button onClick={() => toggleItem(it.id)} className={classes.list} style={{paddingLeft: (level + 1) * 15}}>  
                    {
                        it.active ? 
                            (it.items && it.items.length > 0 ? <i style={{color: '#4285F6', position: 'absolute'}} className={it.open ? "fas fa-caret-down" : "fas fa-caret-right"} /> : null) : 
                            (it.items && it.items.length > 0 ? <i style={{position: 'absolute'}} className={it.open ? "fas fa-caret-down" : "fas fa-caret-right"} /> : null)
                    }
                    {render(it)}
                </ListItem>
                {it.items && it.items.length > 0 &&
                <Collapse in={it.open} timeout="auto" unmountOnExit>
                    <Tree items={it.items}
                        render={render}
                        toggleItem={toggleItem}
                        level={level + 1}/>
                </Collapse>}
             </div>)}
        </List>
    }
}


export default withStyles(styles)(Tree);
