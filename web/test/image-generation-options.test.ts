import { describe, expect, test } from "bun:test";

import { imageAspectForSize, imageAspectOptions, imageQualityForResolution, imageResolutionForValue, imageResolutionOptions, imageSizeForAspect, imageSizeLabel } from "../src/lib/image-generation-options";
import { defaultImageCapabilityConfig } from "../src/lib/model-capabilities";

const landscape = imageAspectOptions.find((item) => item.value === "16:9")!;
const portrait = imageAspectOptions.find((item) => item.value === "9:16")!;

describe("image generation options", () => {
    test("splits fixed-size image variants into a ratio and resolution", () => {
        const profile = defaultImageCapabilityConfig();

        expect(imageAspectForSize("3840x2160")?.value).toBe("16:9");
        expect(imageSizeLabel("2160x3840")).toBe("9:16");
        expect(imageResolutionForValue({ size: "3840x2160", quality: "auto" })).toBe("4k");
        expect(imageResolutionOptions(profile, landscape).map((item) => item.value)).toEqual(["1k", "2k", "4k"]);
        expect(imageSizeForAspect(profile, landscape, "4k")).toBe("3840x2160");
        expect(imageSizeForAspect(profile, portrait, "4k")).toBe("2160x3840");
    });

    test("uses provider quality values when resolution is a separate parameter", () => {
        const profile = defaultImageCapabilityConfig("grok-image");

        expect(imageResolutionOptions(profile, landscape).map((item) => item.value)).toEqual(["1k", "2k"]);
        expect(imageSizeForAspect(profile, landscape, "2k")).toBe("16:9");
        expect(imageQualityForResolution(profile, "2k")).toBe("2k");
    });

    test("maps generic image quality to the matching clarity choice", () => {
        const profile = defaultImageCapabilityConfig();

        expect(imageResolutionForValue({ size: "16:9", quality: "auto" })).toBe("1k");
        expect(imageResolutionForValue({ size: "16:9", quality: "medium" })).toBe("2k");
        expect(imageQualityForResolution(profile, "4k")).toBe("high");
    });
});
