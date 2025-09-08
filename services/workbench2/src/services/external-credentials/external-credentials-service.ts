// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { ListArguments, CommonService } from "services/common-service/common-service";
import { AxiosInstance } from "axios";
import { ApiActions } from "services/api/api-actions";
import { ListResults } from "services/common-service/common-service";
import { ExternalCredential } from "models/external-credential";
import { ExternalCredentialCreateFormDialogData } from "store/external-credentials/external-credential-create-actions";

export class ExternalCredentialsService extends CommonService<ExternalCredential> {
    constructor(serverApi: AxiosInstance, actions: ApiActions) {
            super(serverApi, "credentials", actions);
        }

        async list( args?: ListArguments, showErrors?: boolean ): Promise<ListResults<ExternalCredential>> {
            return super.list(args, showErrors);
        }

        create(data: ExternalCredentialCreateFormDialogData, showErrors?: boolean): Promise<ExternalCredential> {
            return super.create(data, showErrors);
        }

        delete( uuid: string, showErrors?: boolean ): Promise<ExternalCredential> {
            return super.delete(uuid, showErrors);
        }
}
