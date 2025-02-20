// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { connect } from "react-redux";
import { RootState } from 'store/store';
import {
    openContextMenuAndSelect,
} from 'store/context-menu/context-menu-actions';
import {
    LinkPanelRoot,
    LinkPanelRootActionProps,
    LinkPanelRootDataProps
} from 'views/link-panel/link-panel-root';
import { ResourceKind } from 'models/resource';
import { resourceToMenuKind } from 'common/resource-to-menu-kind';

const mapStateToProps = (state: RootState): LinkPanelRootDataProps => {
    return {
        resources: state.resources
    };
};

const mapDispatchToProps = (dispatch: Dispatch): LinkPanelRootActionProps => ({
    onContextMenu: (event, resourceUuid) => {
        const kind = dispatch<any>(resourceToMenuKind(resourceUuid));
        if (kind) {
            dispatch<any>(openContextMenuAndSelect(event, {
                name: '',
                uuid: resourceUuid,
                ownerUuid: '',
                kind: ResourceKind.LINK,
                menuKind: kind
            }));
        }
    },
    onItemClick: (resourceUuid: string) => { return; },
    onItemDoubleClick: uuid => { return; }
});

export const LinkPanel = connect(mapStateToProps, mapDispatchToProps)(LinkPanelRoot);