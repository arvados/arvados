// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { snakeCase, camelCase } from 'lodash';
import { CommonResourceService } from 'services/common-service/common-resource-service';
import {
  ListResults,
  ListArguments,
} from 'services/common-service/common-service';
import { AxiosInstance, AxiosRequestConfig } from 'axios';
import { CollectionResource } from 'models/collection';
import { ProjectResource } from 'models/project';
import { ProcessResource } from 'models/process';
import { WorkflowResource } from 'models/workflow';
import { TrashableResourceService } from 'services/common-service/trashable-resource-service';
import { ApiActions } from 'services/api/api-actions';
import { GroupResource } from 'models/group';
import { Session } from 'models/session';

export interface ContentsArguments {
  limit?: number;
  offset?: number;
  order?: string;
  filters?: string;
  recursive?: boolean;
  includeTrash?: boolean;
  excludeHomeProject?: boolean;
}

export interface SharedArguments extends ListArguments {
  include?: string;
}

export type GroupContentsResource =
  | CollectionResource
  | ProjectResource
  | ProcessResource
  | WorkflowResource;

export class GroupsService<
  T extends GroupResource = GroupResource
> extends TrashableResourceService<T> {
  constructor(serverApi: AxiosInstance, actions: ApiActions) {
    super(serverApi, 'groups', actions);
  }

  async contents(
    uuid: string,
    args: ContentsArguments = {},
    session?: Session
  ): Promise<ListResults<GroupContentsResource>> {
    const { filters, order, ...other } = args;
    const params = {
      ...other,
      filters: filters ? `[${filters}]` : undefined,
      order: order ? order : undefined,
    };
    const pathUrl = uuid ? `/${uuid}/contents` : '/contents';

    const cfg: AxiosRequestConfig = {
      params: CommonResourceService.mapKeys(snakeCase)(params),
    };
    if (session) {
      cfg.baseURL = session.baseUrl;
      cfg.headers = { Authorization: 'Bearer ' + session.token };
    }

    const response = await CommonResourceService.defaultResponse(
      this.serverApi.get(this.resourceType + pathUrl, cfg),
      this.actions,
      false
    );

    return {
      ...TrashableResourceService.mapKeys(camelCase)(response),
      clusterId: session && session.clusterId,
    };
  }

  shared(
    params: SharedArguments = {}
  ): Promise<ListResults<GroupContentsResource>> {
    return CommonResourceService.defaultResponse(
      this.serverApi.get(this.resourceType + '/shared', { params }),
      this.actions
    );
  }
}

export enum GroupContentsResourcePrefix {
  COLLECTION = 'collections',
  PROJECT = 'groups',
  PROCESS = 'container_requests',
  WORKFLOW = 'workflows',
}
