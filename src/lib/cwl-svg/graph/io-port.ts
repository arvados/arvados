export class IOPort {

    static radius = 7;

    /**
     * @param x1
     * @param y1
     * @param x2
     * @param y2
     * @param {"right" | "left" | string} forceDirection
     * @returns {string}
     */
    public static makeConnectionPath(x1: any, y1: any, x2: any, y2: any, forceDirection: "right" | "left" | string = "right"): string | undefined {

        if (!forceDirection) {
            return `M ${x1} ${y1} C ${(x1 + x2) / 2} ${y1} ${(x1 + x2) / 2} ${y2} ${x2} ${y2}`;
        } else if (forceDirection === "right") {
            const outDir = x1 + Math.abs(x1 - x2) / 2;
            const inDir  = x2 - Math.abs(x1 - x2) / 2;

            return `M ${x1} ${y1} C ${outDir} ${y1} ${inDir} ${y2} ${x2} ${y2}`;
        } else if (forceDirection === "left") {
            const outDir = x1 - Math.abs(x1 - x2) / 2;
            const inDir  = x2 + Math.abs(x1 - x2) / 2;

            return `M ${x1} ${y1} C ${outDir} ${y1} ${inDir} ${y2} ${x2} ${y2}`;
        }

        return undefined;
    }
}
