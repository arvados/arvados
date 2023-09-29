export class TemplateParser {

    static parse(tpl: any) {
        const ns = "http://www.w3.org/2000/svg";
        const node = document.createElementNS(ns, "g");
        node.innerHTML = tpl;
        return node.firstElementChild;
    }
}
