// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { serverApi } from "../../common/server-api";
import { Dispatch } from "redux";
import actions from "../../store/project/project-action";
import { Project } from "../../models/project";

interface GroupsResponse {
    offset: number;
    limit: number;
    items: Array<{
        href: string;
        kind: string;
        etag: string;
        uuid: string;
        owner_uuid: string;
        created_at: string;
        modified_by_client_uuid: string;
        modified_by_user_uuid: string;
        modified_at: string;
        name: string;
        group_class: string;
        description: string;
        writable_by: string[];
        delete_at: string;
        trash_at: string;
        is_trashed: boolean;
    }>;
}

export default class ProjectService {
    public getTopProjectList = () => (dispatch: Dispatch) => {
        dispatch(actions.TOP_PROJECTS_REQUEST());
        serverApi.get<GroupsResponse>('/groups').then(groups => {
            const projects = groups.data.items.map(g => ({
                name: g.name,
                createdAt: g.created_at,
                modifiedAt: g.modified_at,
                href: g.href,
                uuid: g.uuid,
                ownerUuid: g.owner_uuid
            } as Project));
            dispatch(actions.TOP_PROJECTS_SUCCESS(projects));
        });
    }
}
