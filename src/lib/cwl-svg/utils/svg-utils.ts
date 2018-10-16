export class SVGUtils {
    static matrixToTransformAttr(matrix: SVGMatrix): string {
        const {a, b, c, d, e, f} = matrix;
        return `matrix(${a}, ${b}, ${c}, ${d}, ${e}, ${f})`;
    }

    static createMatrix(): SVGMatrix {
        return document.createElementNS("http://www.w3.org/2000/svg", "svg").createSVGMatrix();

    }
}