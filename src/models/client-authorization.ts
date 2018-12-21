export interface ClientAuthorizationResource {
    uuid: string;
    apiToken: string;
    apiClientId: number;
    userId: number;
    createdByIpAddress: string;
    lastUsedByIpAddress: string;
    lastUsedAt: string;
    expiresAt: string;
    ownerUuid: string;
    scopes: string[];
}
