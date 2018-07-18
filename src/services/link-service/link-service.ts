// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import CommonResourceService from "../../common/api/common-resource-service";
import { Link } from "../../models/link";
import { AxiosInstance } from "../../../node_modules/axios";

export default class LinkService<T extends Link = Link> extends CommonResourceService<T> {
    constructor(serverApi: AxiosInstance) {
        super(serverApi, "links");
    }
}