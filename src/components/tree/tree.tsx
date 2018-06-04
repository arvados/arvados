// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import List from "@material-ui/core/List/List";
import ListItem from "@material-ui/core/ListItem/ListItem";
import { ReactElement } from "react";
import Collapse from "@material-ui/core/Collapse/Collapse";

export interface TreeItem<T> {
    data: T;
    id: string;
    open: boolean;
    items?: Array<TreeItem<T>>;
}

interface TreeProps<T> {
    items?: Array<TreeItem<T>>;
    render: (item: T) => ReactElement<{}>;
    toggleItem: (id: string) => any;
    level?: number;
}

class Tree<T> extends React.Component<TreeProps<T>, {}> {
    render(): ReactElement<any> {
        const level = this.props.level ? this.props.level : 0;
        return <List component="div">
            {this.props.items && this.props.items.map((it: TreeItem<T>, idx: number) =>
             <div key={`item/${level}/${idx}`}>      
                <ListItem button onClick={() => this.props.toggleItem(it.id)} style={{paddingLeft: (level + 1) * 20}}>  
                    <i style={{marginRight: "10px"}} className={it.open ? "fas fa-caret-down" : "fas fa-caret-right"} />
                    {this.props.render(it.data)}
                </ListItem>
                {it.items && it.items.length > 0 &&
                <Collapse in={it.open} timeout="auto" unmountOnExit>
                    <Tree items={it.items}
                        render={this.props.render}
                        toggleItem={this.props.toggleItem}
                        level={level + 1}/>
                </Collapse>}
             </div>)}
        </List>
    }
}

export default Tree;
