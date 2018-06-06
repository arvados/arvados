// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import List from "@material-ui/core/List/List";
import ListItem from "@material-ui/core/ListItem/ListItem";
import { ReactElement } from "react";

interface TreeProps<T> {
    items: T[],
    render: (item: T) => ReactElement<{}>
}

class Tree<T> extends React.Component<TreeProps<T>, {}> {
    render() {
        return <List>
            {this.props.items && this.props.items.map((it: T, idx: number) =>
                <ListItem key={`item/${idx}`} button>
                    {this.props.render(it)}
                </ListItem>
            )}
        </List>
    }
}

export default Tree;
