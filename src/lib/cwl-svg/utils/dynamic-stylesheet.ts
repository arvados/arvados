import {Workflow} from "..";

export class DynamicStylesheet {
    private styleElement: HTMLStyleElement;
    private scopedSelector: string;
    private innerStyle = "";

    constructor(workflow: Workflow) {

        this.styleElement      = document.createElement("style");
        this.styleElement.type = "text/css";

        this.scopedSelector = `svg.${workflow.svgID}`;

        document.getElementsByTagName("head")[0].appendChild(this.styleElement);
    }

    remove() {
        this.styleElement.remove();
    }

    set(style: string) {
        this.innerStyle = style;

        this.styleElement.innerHTML = `
            ${this.scopedSelector} {
                ${this.innerStyle}
            }
        `
    }




}
