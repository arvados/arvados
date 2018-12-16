import { Dispatch } from "redux";
import { setBreadcrumbs } from "~/store/breadcrumbs/breadcrumbs-actions";
import { RootState } from "~/store/store";
import { ServiceRepository } from "~/services/services";
import Axios from "axios";
import { getUserFullname } from "~/models/user";
import { authActions } from "~/store/auth/auth-action";
import { Config, DISCOVERY_URL } from "~/common/config";
import { Session } from "~/models/session";
import { progressIndicatorActions } from "~/store/progress-indicator/progress-indicator-actions";
import { UserDetailsResponse } from "~/services/auth-service/auth-service";


const getSessionOrigin = async (session: Session) => {
    let url = session.remoteHost;
    if (url.indexOf('://') < 0) {
        url = 'https://' + url;
    }
    const origin = new URL(url).origin;
    try {
        const resp = await Axios.get<Config>(`${origin}/${DISCOVERY_URL}`);
        return resp.data.origin;
    } catch (err) {
        try {
            const resp = await Axios.get<any>(`${origin}/status.json`);
            return resp.data.apiBaseURL;
        } catch (err) {
        }
    }
    return null;
};

const getUserDetails = async (origin: string, token: string): Promise<UserDetailsResponse> => {
    const resp = await Axios.get<UserDetailsResponse>(`${origin}/arvados/v1/users/current`, {
        headers: {
            Authorization: `OAuth2 ${token}`
        }
    });
    return resp.data;
};

const validateSessions = () =>
    async (dispatch: Dispatch<any>, getState: () => RootState, services: ServiceRepository) => {
        const sessions = getState().auth.sessions;
        dispatch(progressIndicatorActions.START_WORKING("sessionsValidation"));
        for (const session of sessions) {
            if (!session.validated) {
                const origin = await getSessionOrigin(session);
                const user = await getUserDetails(origin, session.token);
            }
        }
        dispatch(progressIndicatorActions.STOP_WORKING("sessionsValidation"));
    };

export const addSession = (remoteHost: string) =>
    (dispatch: Dispatch, getState: () => RootState, services: ServiceRepository) => {
        const user = getState().auth.user!;
        const clusterId = remoteHost.match(/^(\w+)\./)![1];

        dispatch(authActions.ADD_SESSION({
            loggedIn: false,
            validated: false,
            email: user.email,
            username: getUserFullname(user),
            remoteHost,
            clusterId,
            token: ''
        }));

        services.authService.saveSessions(getState().auth.sessions);
    };

export const loadSiteManagerPanel = () =>
    async (dispatch: Dispatch<any>) => {
        try {
            dispatch(setBreadcrumbs([{ label: 'Site Manager'}]));
            dispatch(validateSessions());
        } catch (e) {
            return;
        }
    };
