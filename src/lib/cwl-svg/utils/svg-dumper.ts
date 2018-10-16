export class SvgDumper {

    private containerElements = ["svg", "g"];
    private embeddableStyles  = {
        "rect": ["fill", "stroke", "stroke-width"],
        "path": ["fill", "stroke", "stroke-width"],
        "circle": ["fill", "stroke", "stroke-width"],
        "line": ["stroke", "stroke-width"],
        "text": ["fill", "font-size", "text-anchor", "font-family"],
        "polygon": ["stroke", "fill"]
    };

    constructor(private svg: SVGSVGElement) {
        this.svg = svg
    }

    dump({padding} = {padding: 50}): string {
        this.adaptViewbox(this.svg, padding);
        const clone = this.svg.cloneNode(true) as SVGSVGElement;

        const portLabels: any = clone.querySelectorAll(".port .label");


        for (const label of portLabels) {
            label.parentNode.removeChild(label);
        }

        this.treeShakeStyles(clone, this.svg);

        // Remove panning handle so we don't have to align it
        const panHandle = clone.querySelector(".pan-handle");
        if (panHandle) {
            clone.removeChild(panHandle);
        }

        return new XMLSerializer().serializeToString(clone);

    }

    private adaptViewbox(svg: SVGSVGElement, padding = 50) {
        const workflow = svg.querySelector(".workflow");
        const rect     = workflow!.getBoundingClientRect();

        const origin = this.getPointOnSVG(rect.left, rect.top);

        const viewBox  = this.svg.viewBox.baseVal;
        viewBox.x      = origin.x - padding / 2;
        viewBox.y      = origin.y - padding / 2;
        viewBox.height = rect.height + padding;
        viewBox.width  = rect.width + padding;

    }

    private getPointOnSVG(x: number, y: number): SVGPoint {
        const svgCTM = this.svg.getScreenCTM();
        const point  = this.svg.createSVGPoint();
        point.x      = x;
        point.y      = y;

        return point.matrixTransform(svgCTM!.inverse());

    }

    private treeShakeStyles(clone: SVGElement, original: SVGElement) {

        const children             = clone.childNodes;
        const originalChildrenData = original.childNodes as NodeListOf<SVGElement>;


        for (let childIndex = 0; childIndex < children.length; childIndex++) {

            const child   = children[childIndex] as SVGElement;
            const tagName = child.tagName;

            if (this.containerElements.indexOf(tagName) !== -1) {
                this.treeShakeStyles(child, originalChildrenData[childIndex]);
            } else if (tagName in this.embeddableStyles) {

                const styleDefinition = window.getComputedStyle(originalChildrenData[childIndex]);

                let styleString = "";
                for (let st = 0; st < this.embeddableStyles[tagName].length; st++) {
                    styleString +=
                        this.embeddableStyles[tagName][st]
                        + ":"
                        + styleDefinition.getPropertyValue(this.embeddableStyles[tagName][st])
                        + "; ";
                }

                child.setAttribute("style", styleString);
            }
        }
    }
}

