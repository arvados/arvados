export function getUrlParameter(search: string, name: string) {
    const safeName = name.replace(/[\[]/, '\\[').replace(/[\]]/, '\\]');
    const regex = new RegExp('[\\?&]' + safeName + '=([^&#]*)');
    const results = regex.exec(search);
    return results === null ? '' : decodeURIComponent(results[1].replace(/\+/g, ' '));
}
