import { RootState } from '~/store/store';

export const getUserUuid = (state: RootState) => {
    const user = state.auth.user;
    if (user) {
        return user.uuid;
    } else {
        return undefined;
    }
};
