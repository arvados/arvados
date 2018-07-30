// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { connect } from "react-redux";
import { CollectionPanelFiles as Component, CollectionPanelFilesProps } from "../../components/collection-panel-files/collection-panel-files";
import { RootState } from "../../store/store";
import { TreeItemStatus } from "../../components/tree/tree";

const mapStateToProps = (state: RootState): Pick<CollectionPanelFilesProps, "items"> => ({
    items: [{
        active: false,
        data: {
            name: "Directory 1",
            type: "directory"
        },
        id: "Directory 1",
        open: true,
        status: TreeItemStatus.LOADED,
        items: [{
            active: false,
            data: {
                name: "Directory 1.1",
                type: "directory"
            },
            id: "Directory 1.1",
            open: false,
            status: TreeItemStatus.LOADED,
            items: []
        }, {
            active: false,
            data: {
                name: "File 1.1",
                type: "file",
                size: 20033
            },
            id: "File 1.1",
            open: false,
            status: TreeItemStatus.LOADED,
            items: []
        }]
    }, {
        active: false,
        data: {
            name: "Directory 2",
            type: "directory"
        },
        id: "Directory 2",
        open: false,
        status: TreeItemStatus.LOADED
    }]
});


export const CollectionPanelFiles = connect(mapStateToProps)(Component);
