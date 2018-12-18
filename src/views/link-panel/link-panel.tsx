// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { connect } from "react-redux";
import { RootState } from '~/store/store';
import { openContextMenu, resourceKindToContextMenuKind } from '~/store/context-menu/context-menu-actions';
import { LinkPanelRoot, LinkPanelRootActionProps, LinkPanelRootDataProps } from '~/views/link-panel/link-panel-root';
import { ResourceKind } from '~/models/resource';

const mapStateToProps = (state: RootState): LinkPanelRootDataProps => {
    return {
        resources: state.resources
    };
};

const mapDispatchToProps = (dispatch: Dispatch): LinkPanelRootActionProps => ({
    onContextMenu: (event, resourceUuid) => {
        const kind = resourceKindToContextMenuKind(resourceUuid);
        if (kind) {
            dispatch<any>(openContextMenu(event, {
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