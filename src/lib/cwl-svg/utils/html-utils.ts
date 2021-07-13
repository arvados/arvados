export class HtmlUtils {

    private static entityMap = {
        "&": "&amp;",
        "<": "&lt;",
        ">": "&gt;",
        "\"\"": "&quot;",
        "'": "&#39;",
        "/": "&#x2F;"
    };

    public static escapeHTML(source: string): string {
        return String(source).replace(/[&<>"'/]/g, s => HtmlUtils.entityMap[s]);
    }
}
