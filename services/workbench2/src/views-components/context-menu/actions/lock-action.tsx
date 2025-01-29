// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from "react";
import { ListItemIcon, ListItemText, ListItem, Tooltip } from "@mui/material";
import { FreezeIcon, UnfreezeIcon } from "components/icon/icon";
import { connect } from "react-redux";
import { RootState } from "store/store";
import { resourceIsFrozen } from "common/frozen-resources";
import { getResource } from "store/resources/resources";
import { GroupResource } from "models/group";
import { memoize } from "lodash";
import { ResourcesState } from "store/resources/resources";

const toolbarIconClass = {
    width: '1rem',
    marginLeft: '-0.5rem',
    marginTop: '0.25rem',
}

const mapStateToProps = (state: RootState): Pick<ToggleLockActionProps, 'selectedResourceUuid' | 'contextMenuResourceUuid' | 'resources'> => ({
    contextMenuResourceUuid: state.contextMenu.resource?.uuid || '',
    selectedResourceUuid: state.selectedResourceUuid,
    resources: state.resources,
});

type ToggleLockActionProps = {
    isInToolbar: boolean;
    selectedResourceUuid: string;
    contextMenuResourceUuid: string,
    resources: ResourcesState,
    onClick: () => void;
};

export const ToggleLockAction = connect(mapStateToProps)(memoize((props: ToggleLockActionProps) => {
    const lockResourceUuid = props.isInToolbar ? props.selectedResourceUuid : props.contextMenuResourceUuid;
    const resource = getResource<GroupResource>(lockResourceUuid)(props.resources);
    const isLocked = resource ? resourceIsFrozen(resource, props.resources) : false;

    return (
        <Tooltip title={isLocked ? "Unfreeze project" : "Freeze project"}>
            <ListItem button onClick={props.onClick} data-cy="toggle-lock-action">
                <ListItemIcon style={props.isInToolbar ? toolbarIconClass : {}}>
                    {isLocked
                        ? <UnfreezeIcon />
                        : <FreezeIcon />}
                </ListItemIcon>
                {!props.isInToolbar &&
                    <ListItemText style={{ textDecoration: 'none' }}>
                        {isLocked
                            ? <>Unfreeze project</>
                            : <>Freeze project</>}
                    </ListItemText>}
            </ListItem >
        </Tooltip>
    );
}));