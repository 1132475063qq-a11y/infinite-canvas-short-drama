import { describe, expect, test } from "bun:test";

import {
    imageAspectForSize,
    imageAspectOptions,
    imageQualityForResolution,
    imageQualityOptions,
    imageResolutionConfigChange,
    imageResolutionOptionForValue,
    imageResolutionOptions,
    imageSizeForAspect,
    imageSizeLabel,
} from "../src/lib/image-generation-options";
import { defaultImageCapabilityConfig } from "../src/lib/model-capabilities";

const landscape = imageAspectOptions.find((item) => item.value === "16:9")!;
const portrait = imageAspectOptions.find((item) => item.value === "9:16")!;
const ultrawide = imageAspectOptions.find((item) => item.value === "21:9")!;

describe("image generation options", () => {
    test("splits fixed-size image variants into a ratio and resolution", () => {
        const profile = defaultImageCapabilityConfig();

        expect(imageAspectForSize("3840x2160")?.value).toBe("16:9");
        expect(imageAspectForSize("2352x1008")?.value).toBe("21:9");
        expect(imageSizeLabel("2160x3840")).toBe("9:16");
        expect(imageSizeLabel("2352x1008")).toBe("21:9");
        expect(imageResolutionOptions(profile, landscape).map((item) => item.label)).toEqual(["1K", "2K", "4K"]);
        expect(imageResolutionOptions(profile, ultrawide).map((item) => item.label)).toEqual(["1K"]);
        expect(imageSizeForAspect(profile, landscape, "4k")).toBe("3840x2160");
        expect(imageSizeForAspect(profile, portrait, "4k")).toBe("2160x3840");

        const fixedSizeResolution = imageResolutionOptionForValue(profile, { size: "2048x1152", quality: "auto" }, landscape)!;
        expect(fixedSizeResolution.source).toBe("size");
        expect(imageResolutionConfigChange(fixedSizeResolution)).toEqual({ key: "size", value: "2048x1152" });
    });

    test("uses provider quality values when resolution is a separate parameter", () => {
        const profile = defaultImageCapabilityConfig("grok-image");

        expect(imageResolutionOptions(profile, landscape).map((item) => item.label)).toEqual(["1K", "2K"]);
        expect(imageSizeForAspect(profile, landscape, "2k")).toBe("16:9");
        expect(imageQualityForResolution(profile, "2k")).toBe("2k");

        const grokResolution = imageResolutionOptionForValue(profile, { size: "16:9", quality: "2k" }, landscape)!;
        expect(grokResolution.source).toBe("quality");
        expect(imageResolutionConfigChange(grokResolution)).toEqual({ key: "quality", value: "2k" });
    });

    test("keeps generic image quality distinct from K resolution labels", () => {
        const profile = defaultImageCapabilityConfig();

        expect(imageQualityOptions(profile)).toEqual([
            { value: "auto", label: "自动" },
            { value: "low", label: "低" },
            { value: "medium", label: "中" },
            { value: "high", label: "高" },
        ]);
        expect(imageQualityForResolution(profile, "4k")).toBeUndefined();
    });
});
