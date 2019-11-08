// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Dispatch } from "redux";
import { connect } from "react-redux";
import { openContextMenu, resourceKindToContextMenuKind } from '~/store/context-menu/context-menu-actions';
import { SubprocessPanelRoot, SubprocessActionProps } from '~/views/subprocess-panel/subprocess-panel-root';
import { ResourceKind } from '~/models/resource';

const mapDispatchToProps = (dispatch: Dispatch): SubprocessActionProps => ({
    onContextMenu: (event, resourceUuid) => {
        const kind = resourceKindToContextMenuKind(resourceUuid);
        if (kind) {
            dispatch<any>(openContextMenu(event, {
                name: '',
                uuid: resourceUuid,
                ownerUuid: '',
                kind: ResourceKind.PROCESS,
                menuKind: kind
            }));
        }
    },
    onItemClick: (resourceUuid: string) => { return; },
    onItemDoubleClick: uuid => { return; }
});

export const SubprocessPanel = connect(mapDispatchToProps)(SubprocessPanelRoot);