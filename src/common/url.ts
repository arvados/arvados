export function getUrlParameter(search: string, name: string) {
    const safeName = name.replace(/[\[]/, '\\[').replace(/[\]]/, '\\]');
    const regex = new RegExp('[\\?&]' + safeName + '=([^&#]*)');
    const results = regex.exec(search);
    return results === null ? '' : decodeURIComponent(results[1].replace(/\+/g, ' '));
}

export function normalizeURLPath(url: string) {
    const u = new URL(url);
    u.pathname = u.pathname.replace(/\/\//, '/');
    if (u.pathname[u.pathname.length - 1] === '/') {
        u.pathname = u.pathname.substr(0, u.pathname.length - 1);
    }
    return u.toString();
}
