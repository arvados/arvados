export interface Session {
    clusterId: string;
    remoteHost: string;
    username: string;
    email: string;
    token: string;
    loggedIn: boolean;
    validated: boolean;
}
